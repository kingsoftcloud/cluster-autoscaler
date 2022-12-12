package kce

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/autoscaler/cluster-autoscaler/config/dynamic"
	"os"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/config"
	"k8s.io/autoscaler/cluster-autoscaler/utils/errors"
)

const (
	// ProviderName is the cloud provider name for Kingsoft
	ProviderName             = "kce"
	NodeGroupIdZoneSeparator = "@"
	NodeGroupIdTemplate      = "%s" + NodeGroupIdZoneSeparator + "%s"
)
// kceCloudProvider implements CloudProvider interface.
type KceCloudProvider struct {
	kceManager      *KceManager   
	resourceLimiter *cloudprovider.ResourceLimiter
	Asgs            []*KceNodeGroup

}

func buildStaticallyDiscoveringProvider(kceManager *KceManager, specs []string, resourceLimiter *cloudprovider.ResourceLimiter) (*KceCloudProvider, error) {
	acp := &KceCloudProvider{
		kceManager:         kceManager,
		Asgs:            make([]*KceNodeGroup, 0),
		resourceLimiter: resourceLimiter,
	}
	for _, spec := range specs {
		if err := acp.addNodeGroup(spec); err != nil {
			klog.Warningf("Failed to add node group to KCE cloud provider with spec: %s", spec)
			return nil, err
		}
	}
	return acp, nil
}

// add node group defined in string spec. Format:
// minNodes:maxNodes:asgName
func (kce *KceCloudProvider) addNodeGroup(spec string) error {
	NodeGroup, err := buildAsgFromSpec(spec, kce.kceManager)
	if err != nil {
		klog.Errorf("Failed to build ASG from spec,because of %s", err.Error())
		return err
	}
	kce.Asgs = append(kce.Asgs, NodeGroup)
	return nil
}

func buildAsgFromSpec(value string, manager *KceManager) (*KceNodeGroup, error) {
	spec, err := dynamic.SpecFromString(value, true)
		klog.V(0).Infof("ASG spec information",spec)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse node group spec: %v. ", err)
	}
	_, err = manager.service.ValidateAsg(&kce_asg.KceAsg{Name: spec.Name})
	if err != nil {
		klog.Errorf("Your scaling group %s does not exist", spec.Name)
		return nil, err
	}

	KceNodeGroup,buildError := buildAsg(manager, spec.MinSize, spec.MaxSize, spec.Name, manager.cfg.RegionId)
	if buildError != nil {
		return KceNodeGroup, nil
	}
	return KceNodeGroup, err
}

func buildAsg(manager *KceManager, minSize int, maxSize int, id string, regionId string) (*KceNodeGroup , error){
	AsgBuild := &kce_asg.KceAsg{
		MinSize:  minSize,
		MaxSize:  maxSize,
		Name:       id,
		//regionId: regionId,
	}
	//获取ASG的ProjectId
	projectIds, err := manager.service.getProjectIdByAsgId(AsgBuild)
	if(err != nil) {
		klog.Errorf("Get ASG %s projectIds error: %v.", AsgBuild.Name, err)
		return nil,err
	}
	klog.V(3).Infof("KCE ASG %s projectId is %d.", AsgBuild.Name, projectIds)
	AsgBuild.ProjectId = projectIds[0]
	KceNodeGroup := &KceNodeGroup{
		kceManager: manager,
		Asg: AsgBuild,
	}
	return KceNodeGroup,nil
}

// add and register an asg to this cloud provider
func (kce *KceCloudProvider) addAsg(asg *KceNodeGroup) {
	kce.Asgs = append(kce.Asgs, asg)
}

func newKceCloudProvider(kceManager *KceManager,discoveryOpts cloudprovider.NodeGroupDiscoveryOptions, resourceLimiter *cloudprovider.ResourceLimiter) (cloudprovider.CloudProvider, error) {
	if discoveryOpts.StaticDiscoverySpecified() {
		return buildStaticallyDiscoveringProvider(kceManager, discoveryOpts.NodeGroupSpecs, resourceLimiter)
	}
	if discoveryOpts.AutoDiscoverySpecified() {
		return nil, fmt.Errorf("Only support static discovery scaling group in KCE cloud now. ")
	}

	return nil, fmt.Errorf("Failed to build KCE cloud provider: node group specs must be specified. ")
}

// BuildKceCloud returns new KceCloudProvider
func BuildKceCloud(opts config.AutoscalingOptions, do cloudprovider.NodeGroupDiscoveryOptions, rl *cloudprovider.ResourceLimiter) cloudprovider.CloudProvider {
	var kceManager *KceManager
	var kce_err error
	externalConfig, err := buildClusterConfig()
	if err != nil {
		klog.Fatalf("Failed to get config for external cluster: %v", err)
	}
	externalClient := kubeclient.NewForConfigOrDie(externalConfig)
	if opts.CloudConfig != "" {
		config, fileErr := os.Open(opts.CloudConfig)
		if fileErr != nil {
			klog.Fatalf("Couldn't open cloud provider configuration %s: %#v", opts.CloudConfig, fileErr)
		}
		defer config.Close()
		kceManager, kce_err = CreateKceManager(config,do, externalClient)
	} else {
		klog.V(0).Infof("Cloud config is null.")
		kceManager, kce_err = CreateKceManager(nil,do, externalClient)
	}
	if kce_err != nil {
		klog.Fatalf("Failed to create KCE Manager: %v", err)
	}
	klog.V(3).Info("Creating KCE Manager Complete.")

	provider, err := newKceCloudProvider(kceManager,do, rl)

	if err != nil {
		klog.Fatalf("Failed to create KCE cloud provider: %v.", err)
	}
	klog.V(3).Infof("KCE Recourse Limiter: %v .", rl)
	klog.V(3).Info("Creating KCE Cloud Complete.")

	return provider
}

func buildClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// Name returns name of the cloud provider.
func (k *KceCloudProvider) Name() string {
	return ProviderName
}

func (kce *KceCloudProvider) NodeGroups() []cloudprovider.NodeGroup {
	result := make([]cloudprovider.NodeGroup, 0, len(kce.Asgs))
	for _, asg := range kce.Asgs {
	result = append(result, asg)
	}
	return result
}

// NodeGroupForNode returns the node group for the given node.
func (k *KceCloudProvider) NodeGroupForNode(node *apiv1.Node) (cloudprovider.NodeGroup, error) {
	ngs := k.NodeGroups()
	for _, ng := range ngs {
		nodes, err := ng.Nodes()
		if err != nil {
			return nil, err
		}
		for _, n := range nodes {
			if node.Name == n.Id {
				return ng, nil
			}
		}
	}
	return nil, nil
}

// Pricing returns pricing model for this cloud provider or error if not available.
func (k *KceCloudProvider) Pricing() (cloudprovider.PricingModel, errors.AutoscalerError) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetAvailableMachineTypes get all machine types that can be requested from the cloud provider.
// Implementation optional.
func (k *KceCloudProvider) GetAvailableMachineTypes() ([]string, error) {
	return []string{}, cloudprovider.ErrNotImplemented
}

// NewNodeGroup builds a theoretical node group based on the node definition provided.
func (k *KceCloudProvider) NewNodeGroup(machineType string, labels map[string]string, systemLabels map[string]string,
	taints []apiv1.Taint,
	extraResources map[string]resource.Quantity) (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// GetResourceLimiter returns struct containing limits (max, min) for resources (cores, memory etc.).
func (k *KceCloudProvider) GetResourceLimiter() (*cloudprovider.ResourceLimiter, error) {
	return k.resourceLimiter, nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
// In particular the list of node groups returned by NodeGroups can change as a result of CloudProvider.Refresh().
func (k *KceCloudProvider) Refresh() error {
	return nil
}

// Cleanup cleans up all resources before the cloud provider is removed
func (k *KceCloudProvider) Cleanup() error {
	return nil
}

func (k *KceCloudProvider) GetAvailableGPUTypes() map[string]struct{} {
	return nil
}

func (k *KceCloudProvider) GPULabel() string {
	return ""
}

func (k *KceCloudProvider) Label() string {
	return ""
}
