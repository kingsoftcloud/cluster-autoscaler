package kce

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"gopkg.in/gcfg.v1"
	"io"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/config"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/patrickmn/go-cache" //add add
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
	//kubeletapis "k8s.io/kubelet/pkg/apis"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/utils/gpu"
)

const (
	KsyunEni               apiv1.ResourceName = "ksyun.com/eni"
	KsyunEtcd              apiv1.ResourceName = "ksyun.com/etcd"
	DefaultNamespace                          = "kube-system"
	DefaultScaleUpCMName                      = "scaleup-v2"
	DefaultEniPluginCMName                    = "kce-eni-plugin-cm"
	DefaultEniNumKey                          = "default-eni-number"
	DefaultEtcdNumKey                         = "default-etcd-number"
	DefaultMasterMinCPU                       = "default-master-min-cpu"

	CacheExpirationSec   = 30
	CacheCleanUpInterval = 10
)

type KceManager struct {
	service *autoScalingWrapper
	spec     cloudprovider.NodeGroupDiscoveryOptions
	specLock sync.RWMutex
	//todo Add kubeClient
	externalClient *kubernetes.Clientset
	cfg      *config.CloudConfig
	cache *cache.Cache
}

type ClusterInfo struct {
	Region    string `json:"region"`
	AccountId string `json:"account_id"`
	ClusterID string `json:"id"`
}
type OpenApiClient struct {
	SecurityToken   string
	debug           bool
	httpClient      *http.Client
}

type KceTemplate struct {
	InstanceType *KceInstance
	Tags   []*autoscaling.TagDescription
	Labels map[string]string
}

type KceInstance struct {
	VCPU     int64 `json:"VCPU"`
	MemoryMb int64 `json:"MemoryGb"`
	///*目前不支持GPU，获取了CPU的数据*/
	GPU int64 `json:"VCPU"`
}

func (m *KceManager) ValidateASG(asg *kce_asg.KceAsg) bool {
	return m.service.ValidateAsgById(asg)
}

// SetAsgSize sets ASG size.
func (m *KceManager) SetKcgSize(kcg *KceNodeGroup, size int) error {
	params := &kce_asg.SetDesiredCapacityInput{
		AutoScalingGroupName: String(kcg.Asg.Name),
		DesiredCapacity:      String(strconv.Itoa(size)),
	}
	klog.V(0).Infof("KCE CA setting Asg %s size to %d", kcg.Asg.Name, size)
	_, err := m.service.SetDesiredCapacity(params, kcg.Asg)
	if err != nil {
		return err
	}
	return nil
}

// GetAsgNodes returns Kce nodes.
func (m *KceManager) GetKceNodes(asg *kce_asg.KceAsg) ([]*InstancesSet, error) {
	list, err := m.service.InstancesByAsg(asg)
	if err == nil {
		return list.Instances, nil
	}
	return nil, err
}

// GetAsgNodes returns Kce nodes.
func (m *KceManager) GetAsgTargetSize(asg *kce_asg.KceAsg) (int, error) {
	list, err := m.service.InstancesByAsg(asg)
	if err != nil {
		return 0, err
	}
	return list.DesiredCapacity, err
}

// DeleteInstances detach nodes.
func (m *KceManager) DeleteInstances(asg *kce_asg.KceAsg, instanceIDs []string, hostNames []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}

	return m.service.DetachInstances(asg, instanceIDs, hostNames)
}

func (m *KceManager) getKceTemplate(asg *kce_asg.KceAsg) (*KceTemplate, error) {
	klog.V(0).Infof("KCE manager get template by ASG name : %s", asg.Name)
	template, err := m.service.GetInstanceTemplate(asg)
	if err != nil {
		return nil, err
	}
	return template, nil
}

func (m *KceManager) GetUnprotectedNodesForScaleDown(nodes []*apiv1.Node) ([]*apiv1.Node, error) {
	klog.V(0).Infof("KceManager get scale down protected nodes.")
	nodes, err := m.service.ScaleDownProtectionCheck(nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (m *KceManager) buildNodeFromTemplate(asg *kce_asg.KceAsg, template *KceTemplate) (*apiv1.Node, error) {
	node := apiv1.Node{}
	nodeName := fmt.Sprintf("%s-asg-%d", asg.Name, rand.Int63())

	node.ObjectMeta = metav1.ObjectMeta{
		Name:     nodeName,
		SelfLink: fmt.Sprintf("/api/v1/nodes/%s", nodeName),
		Labels:   map[string]string{},
	}

	node.Status = apiv1.NodeStatus{
		Capacity: apiv1.ResourceList{},
	}

	// TODO: get a real value.
	node.Status.Capacity[apiv1.ResourcePods] = *resource.NewQuantity(110, resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceCPU] = *resource.NewQuantity(template.InstanceType.VCPU, resource.DecimalSI)
	node.Status.Capacity[gpu.ResourceNvidiaGPU] = *resource.NewQuantity(template.InstanceType.GPU, resource.DecimalSI)
	node.Status.Capacity[apiv1.ResourceMemory] = *resource.NewQuantity(template.InstanceType.MemoryMb*1024*1024, resource.DecimalSI)
	configmap, err := m.externalClient.CoreV1().ConfigMaps(DefaultNamespace).Get(context.TODO(),DefaultEniPluginCMName, metav1.GetOptions{})
	klog.V(0).Infof("Get configmap kind %s apiVersion %s and data %s", configmap.Kind, configmap.APIVersion,configmap.Data)
	if err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("Get configmap %s/%s error: %v", DefaultNamespace, DefaultEniPluginCMName, err)
		}
	} else {
		masterRoleCpuStr, ok := configmap.Data[DefaultMasterMinCPU]
		if ok {
			masterRoleCpuInt, err := strconv.Atoi(masterRoleCpuStr)
			if err == nil {
				if template.InstanceType.VCPU >= int64(masterRoleCpuInt) {
					if value, ok := configmap.Data[DefaultEniNumKey]; ok {
						defaultEni, err := strconv.Atoi(value)
						if err == nil {
							klog.Infof("ASG %s set capacity, key:%s, value:%v", asg.Name, KsyunEni, int64(defaultEni))
							node.Status.Capacity[KsyunEni] = *resource.NewQuantity(int64(defaultEni), resource.DecimalSI)
						}
					}
				} else {
					if value, ok := configmap.Data[DefaultEtcdNumKey]; ok {
						defaultEtcd, err := strconv.Atoi(value)
						if err == nil {
							klog.Infof("ASG %s set capacity, key:%s, value:%v", asg.Name, KsyunEtcd, int64(defaultEtcd))
							node.Status.Capacity[KsyunEtcd] = *resource.NewQuantity(int64(defaultEtcd), resource.DecimalSI)
						}
					}
				}
			}
		}
	}

	// TODO: use proper allocatable!!
	node.Status.Allocatable = node.Status.Capacity

	// GenericLabels
	node.Labels = cloudprovider.JoinStringMaps(node.Labels, buildGenericLabels(template, nodeName))

	// NodeLabels
	if template.Labels != nil {
		klog.V(0).Infof("Set ASG %s template labels %v", asg.Name, template.Labels)
		node.Labels = cloudprovider.JoinStringMaps(node.Labels, template.Labels)
	}

	if template.Tags != nil {
		klog.V(0).Infof("Set ASG %s template taints %v", asg.Name, template.Tags)
		node.Spec.Taints = extractTaintsFromAsg(template.Tags)
	}

	node.Status.Conditions = cloudprovider.BuildReadyConditions()
	return &node, nil

}

func buildGenericLabels(template *KceTemplate, nodeName string) map[string]string {
	result := make(map[string]string)
	result[kubeletapis.LabelArch] = cloudprovider.DefaultArch
	result[kubeletapis.LabelOS] = cloudprovider.DefaultOS
	//result[apiv1.LabelZoneRegion] = template.Region
	//result[apiv1.LabelZoneFailureDomain] = template.Zone
	return result
}

func extractTaintsFromAsg(tags []*autoscaling.TagDescription) []apiv1.Taint {
	taints := make([]apiv1.Taint, 0)
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		k := *tag.Key
		v := *tag.Value
		// The tag value must be in the format <tag>:NoSchedule
		r, _ := regexp.Compile("(.*):(?:NoSchedule|NoExecute|PreferNoSchedule)")
		if r.MatchString(v) {
			values := strings.SplitN(v, ":", 2)
			if len(values) > 1 {
				taints = append(taints, apiv1.Taint{
					Key:    k,
					Value:  values[0],
					Effect: apiv1.TaintEffect(values[1]),
				})
			}
		} else {
			klog.V(3).Info("ExtractTaintsFromAsg: invalid taints %s=%s", k, v)
		}
	}
	return taints
}

func NewOpenApiClient( ) (*OpenApiClient, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}
	return &OpenApiClient{
		debug:           false,
		httpClient:      httpClient,
	}, nil
}

func getKceClusterinfo(kubeclient *kubernetes.Clientset) (*ClusterInfo, error) {
	cm, err := kubeclient.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(context.TODO(),"cluster-cm", metav1.GetOptions{})
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	jsonData, err := json.Marshal(cm.Data)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	cluster := new(ClusterInfo)
	if err := json.Unmarshal(jsonData, &cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func CreateKceManager(configReader io.Reader, discoveryOpts cloudprovider.NodeGroupDiscoveryOptions, externalClient *kubernetes.Clientset) (*KceManager, error) {
	//Create kce manager.
	cfg := &config.CloudConfig{}
	var err error
	if configReader != nil {
		if err := gcfg.ReadInto(cfg, configReader); err != nil {
			klog.Errorf("Couldn't read config: %v", err)
			return nil, err
		}
	}
	if cfg.IsValid() == false {
		klog.Errorf("Please check whether you have \" +\"provided correct AccessKeyId,AccessKeySecret,RegionId or STS Token")
		//return nil, "please check whether you have " +"provided correct AccessKeyId,AccessKeySecret,RegionId or STS Token"
		return nil, err
	}
	asw, err := newAutoScalingWrapper(cfg,externalClient)// newwrapper
	if err != nil {
		klog.Errorf("Failed to create NewAutoScalingWrapper because of %s", err)
		return nil, err
	}
	manager := &KceManager{
		service:        asw,
		//spec:           discoveryOpts,
		specLock:       sync.RWMutex{},
		externalClient: externalClient,
		cfg: 			cfg,
		cache:          cache.New(CacheExpirationSec*time.Second, CacheCleanUpInterval*time.Second),
	}
	if discoveryOpts.StaticDiscoverySpecified() {
		manager.spec = discoveryOpts
	}
	return manager, nil
}

func syncNodeGroupsFromCM(manager *KceManager) {
	conf, err := manager.externalClient.CoreV1().ConfigMaps(DefaultNamespace).Get(context.TODO(),DefaultScaleUpCMName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			manager.specLock.Lock()
			manager.spec.NodeGroupSpecs = make([]string, 0)
			manager.specLock.Unlock()
		}
		klog.Warningf("Get configmap %s/%s error: %v", DefaultNamespace, DefaultScaleUpCMName, err)
	} else {
		if nodesJsonStr, ok := conf.Data["group"]; ok {
			if len(nodesJsonStr) != 0 {
				nodeGroupSpecs, err := convertToNodeGroupSpecs(nodesJsonStr)
				if err == nil {
					manager.specLock.Lock()
					manager.spec.NodeGroupSpecs = nodeGroupSpecs
					manager.specLock.Unlock()
					klog.Infof("Get NodeGroups form %s/%s, NodeGroups: %v", DefaultNamespace, DefaultScaleUpCMName, nodeGroupSpecs)
				}
			}
		}
	}
}

func convertToNodeGroupSpecs(nodesJsonStr string) ([]string, error) {
	var jsonArrayObject = &ScaleUpConfigMapData{}
	var nodeGroupSpecs []string
	err := json.Unmarshal([]byte(nodesJsonStr), &jsonArrayObject)
	if err != nil {
		klog.Errorf("Convert configmap %s/%s error: %v", DefaultNamespace, DefaultScaleUpCMName, err)
		return nil, err
	}
	for _, nodeGroup := range jsonArrayObject.Nodes {
		nodeGroupStr, err := json.Marshal(nodeGroup)
		if err != nil {
			klog.Errorf("Convert node info error: %v", err)
			return nil, err
		}
		nodeGroupSpecs = append(nodeGroupSpecs, string(nodeGroupStr))
	}
	return nodeGroupSpecs, nil
}
