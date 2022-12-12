package kce

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/config"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-cloud-client/services"
	//"os"
	"strings"
	"time"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)
const (
	AppengineInstanceUUIDKey = "appengine.sdns.ksyun.com/instance-uuid"
	refreshClientInterval   = 60 * time.Minute
	acsAutogenIncreaseRules = "acs-autogen-increase-rules"
	defaultAdjustmentType   = "TotalCapacity"
)

type autoScaling interface {
    GetProjectIdByAsg(asg * kce_asg.KceAsg) (data []byte, err error)
	ModifyScalingGroup(input *kce_asg.SetDesiredCapacityInput, asg *kce_asg.KceAsg) ([]byte, error)
	DescribeScalingInstance(id []string,projectIds []int64) ([]byte, error)
	CheckScaleDownProtection(nodes []*apiv1.Node) ([]byte, error)
	ListInstancesByAsg(asg *kce_asg.KceAsg) ([]byte, error)
	FindTemplateByAsg(asg * kce_asg.KceAsg) ([]byte, error)
   	ValidateAsg(asg * kce_asg.KceAsg) ([]byte, error)
	ListLabelsByAsg(asg * kce_asg.KceAsg) ([]byte, error)
	ListTaintsByAsg(asg * kce_asg.KceAsg) ([]byte, error)
	DetachInstancesById(asg * kce_asg.KceAsg, instanceIDs []string) ([]byte, error)
	CheckAutoScalerCanSell(asgs []* kce_asg.KceAsg) ([]byte, error)
	SetDesiredCapacity(input *kce_asg.SetDesiredCapacityInput, asg * kce_asg.KceAsg) ([]byte, error)
}

type CheckResponse struct {
	RequestId string `json:"RequestId"`
	Return    bool   `json:"Return"`
}

type ModifyResponse struct {
	RequestId string    `json:"RequestId"`
	ReturnSet ReturnSet `json:"ReturnSet"`
}

type ReturnSet struct {
	ScalingGroupId string `json:"ScalingGroupId"`
}

type LabelResponse struct {
	RequestId     string `json:"RequestId"`
	ReturnMessage string `json:"ReturnMessage"`
}

type TaintResponse struct {
	RequestId     string `json:"RequestId"`
	ReturnMessage string `json:"ReturnMessage"`
}

type TemplateResponse struct {
	RequestId string `json:"RequestId"`
	Return    string `json:"ReturnMessage"`
	//ScalingGroupId string `json:"ScalingGroupId"`
	ScalingConfigurationSet []*ScalingConfigurationSet `json:"ScalingConfigurationSet"`
}

//type CheckCanSellResponse struct {
//	RequestId     string           `json:"RequestId"`
//	CanSellAsgSet []CanSellAsgItem `json:"AutoScalingGroupCanSellSet"`
//}

type CheckCanSellResponse struct {
	RequestId     string           `json:"RequestId"`
	CanSellAsgSet []CanSellAsgItem `json:"ScalingGroupSet"`
}

type CanSellAsgItem struct {
	ScalingGroupId 	   string `json:"ScalingGroupId"`
	CanSell            string   `json:"Status"`
}

//type CanSellAsgItem struct {
//	AutoScalingGroupId string `json:"AutoScalingGroupId"`
//	CanSell            bool   `json:"CanSell"`
//}

type ScalingConfigurationSet struct {
	VCPU             int64  `json:"Cpu,string"`
	MemoryGb         int64  `json:"Mem,string"`
	GPU              int64  `json:"Gpu,string"`
	ContainerLabel   string `json:"ContainerLabel"`
	AvailabilityZone string `json:"availabilityZone"`
	ProjectId        int64  `json:"projectId"`
}

type InstanceResponse struct {
	RequestId     string           `json:"RequestId"`
	Instance      []*Instance      `json:"InstancesSet"`
}

type Instance struct {
	PrivateIpAddress string           `json:"PrivateIpAddress"`
	//InstanceName   string           `json:"InstanceName"`
	HostName         string           `json:"hostName"`
}

type autoScalingWrapper struct {
	oclient *OpenApiClient
	//todo Add kubeClient
	externalClient *kubernetes.Clientset
	autoScaling //client实现该接口
	cfg *config.CloudConfig
}

func newAutoScalingWrapper(cfg *config.CloudConfig,externalClient *kubernetes.Clientset) (*autoScalingWrapper, error) {
	if cfg.IsValid() == false {
		//Never reach here.
		return nil, fmt.Errorf("Your cloud config is not valid. ")
	}
	asw := &autoScalingWrapper{
		cfg: cfg,
	}
	openApiClient, err := NewOpenApiClient()
	if err != nil {
		return nil, err
	}
	klog.V(0).Info("KCE OpenApi Client Complete.")
	asw.oclient= openApiClient
	asw.externalClient= externalClient

	if cfg.STSEnabled == true {
		go func(asw *autoScalingWrapper, cfg *config.CloudConfig) {
			timer := time.NewTicker(refreshClientInterval)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
					client, err := getKceClient(cfg)
					if err == nil {
						asw.autoScaling = client
					}
				}
			}
		}(asw, cfg)
	}
	client, err := getKceClient(cfg)
	client.HttpClient = openApiClient.httpClient
	if err == nil {
		asw.autoScaling = client
	}
	return asw, err
}

func getKceClient(cfg *config.CloudConfig) (client *services.Client, err error) {
	region := cfg.RegionId
	client = &services.Client{}
	client.CloudConfig.AccessKeyID = cfg.AccessKeyID
	client.RegionId = region
	client.CloudConfig.AccessKeySecret = cfg.AccessKeySecret
	if err != nil {
		klog.Errorf("Failed to create ess kce-client with AccessKeyId and AccessKeySecret,Because of %s", err.Error())
	}
	return
}

func (a *autoScalingWrapper) SetDesiredCapacity (input *kce_asg.SetDesiredCapacityInput, asg * kce_asg.KceAsg) (bool, error) {
	info, err :=a.ModifyScalingGroup(input,asg)
	var resp CheckResponse
	err = json.Unmarshal(info, &resp)
	if err != nil {
		klog.Errorf("Invalid ASG %s, error: %v", asg.Name, err)
		return false,err
	}
	if (!resp.Return){
		klog.Errorf("Invalid ASG %s, error: asg can't sell", asg.Name)
		return false,err
	}
	return true,nil
}

func (a *autoScalingWrapper) SetDesiredCapacity2018(input *kce_asg.SetDesiredCapacityInput, asg * kce_asg.KceAsg) (bool, error) {
	var err error
	klog.V(0).Info("KCE CA set current size " + aws.StringValue(input.DesiredCapacity) + " by openapi")
	_, err = a.SetDesiredCapacity(input, asg)
	if err == nil {
		klog.V(3).Info("KCE CA set current size " + aws.StringValue(input.DesiredCapacity) + "" +
			" success, ASG ID :" + asg.Name)
		return true, nil
	}
	klog.V(0).Info("KCE CA set current size failed ,because :" + err.Error())
	return false, err
}

func (a *autoScalingWrapper) ScaleDownProtectionCheck(nodes []*apiv1.Node) ([]*apiv1.Node, error) {
	var err error
	var table = make(map[string]*apiv1.Node, len(nodes))
	for _, node := range nodes {
		SystemUUID:=node.Status.NodeInfo.SystemUUID
		klog.V(0).Infof("Get nodeinstanceuuid by node.Status.NodeInfo.SystemUUID : %s", SystemUUID)
		if SystemUUID!="" {
			table[SystemUUID] = node
		} else {
			klog.Warningf("ScaleDownProtectionCheck error: node %s without SystemUUID , skip ", node.Name)
		}
	}
	klog.V(0).Info("ScaleDownProtectionCheck: %v", table)
	var instanceList InstanceList
	vms, err := a.CheckScaleDownProtection(nodes)
	if err == nil {
		err = json.Unmarshal(vms, &instanceList)
		if err != nil {
			return nil, err
		}
	}
	var unprotected []*apiv1.Node
	for _, instance := range instanceList.Instances {
		if node, found := table[instance.ID]; found && !instance.ProtectedFromScaleDown {
			unprotected = append(unprotected, node)
		} else {
			klog.V(0).Info("ScaleDownProtectionCheck: node %s is under ScaleDownProtection." + instance.ID)
		}
	}
	return unprotected, err
}

//get All Instances by kce autoscaling group name
func (a *autoScalingWrapper) InstancesByAsg2018(asg * kce_asg.KceAsg) (*InstanceList2018, error) {
	var err error
	klog.V(4).Info("List instances by ASGName: " + asg.Name)
	vms, err := a.ListInstancesByAsg(asg)
	if err == nil {
		var instances InstanceList2018
		err = json.Unmarshal(vms, &instances)
		if err != nil {
			return nil, err
		}
		_, zone := kce_asg.GetNodeGroupNameAndZone(asg)
		if zone != "" && len(instances.Instances) > 0 {
			var instancesZone []*InstancesSet2018
			for _, instance := range instances.Instances {
				if instance.AvailableZone == zone {
					instancesZone = append(instancesZone, instance)
				} else if instance.AvailableZone == "" {
					klog.V(5).Infof("ASG-%s: Filtering out instance - %s for az not provided.", asg.Name, instance.HostnameOverride)
				} else {
					klog.V(5).Infof("ASG-%s: Filtering out instance - %s@%s for not in az %s.", asg.Name, instance.HostnameOverride, instance.AvailableZone, zone)
				}
			}
			return &InstanceList2018{instances.DesiredCapacity, instances.RequestId, instancesZone}, nil
		}
		return &instances, err
	}
	klog.V(3).Infof("KCE CA list instances by ASGName: %s failed ,because: %v", asg.Name, err)
	return nil, err
}

func (a *autoScalingWrapper) getProjectIdByAsgId(asg * kce_asg.KceAsg) (projectId []int64, err error) {
	//根据AsgId获取启动模板,从而获取到ProjectID
	var ProjectIds []int64
	var ProjectIdsMap = make(map[int64]string)
	info, err := a.GetProjectIdByAsg(asg)
	if err == nil {
		var response TemplateResponse
		err = json.Unmarshal(info, &response)
		if err != nil {
			klog.Errorf("Json Unmarshal error: %v", err)
			return nil, err
		}
		if len(response.ScalingConfigurationSet) == 0 || response.ScalingConfigurationSet[0] == nil {
			klog.Errorf("ASG %s,  template ScalingConfigurationSet is nil.", asg.Name)
			return nil, fmt.Errorf("ASG %s template is nil. ", asg.Name)
		}
		//记录ProjectId
		ProjectId := response.ScalingConfigurationSet[0].ProjectId
		if _, ok := ProjectIdsMap[ProjectId]; ok != true {
			ProjectIds = append(ProjectIds, ProjectId)
			ProjectIdsMap[ProjectId] = ""
		}
		return ProjectIds,nil
	}
	return nil, err
}

func  (a *autoScalingWrapper)  GetHostNameById (ids []string,asg * kce_asg.KceAsg)([]string,error){
	data, err :=a.DescribeScalingInstance(ids,[]int64{asg.ProjectId})
	if(err != nil) {
		klog.Errorf("Invalid ASG %s, error: %v", err)
	}
	var resp InstanceResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		return nil ,err
	}
	var hostNames []string
	for _,item := range resp.Instance{
		hostNames = append(hostNames,item.HostName)
	}
	return hostNames,nil
}

func (a *autoScalingWrapper) InstancesByAsg(asg * kce_asg.KceAsg) (*InstanceList, error){
	var err error
	klog.V(4).Info("List instances by ASGName: " + asg.Name)
	vms, err := a.ListInstancesByAsg(asg)
	if err == nil {
		var instances InstanceList
		err = json.Unmarshal(vms, &instances)
		if err != nil {
			return nil, err
		}
		if len(instances.Instances) > 0 {
			var allInstances []*InstancesSet
			var instancesIds []string
			//获取所有的ASG下的instanceID
			for _, instance := range instances.Instances {
				instancesIds = append(instancesIds,instance.ID)
			}
			//获取所有的HostName
			instanceHostNames,getHostNameError := a.GetHostNameById(instancesIds,asg)
			if getHostNameError!=nil {
				return nil,getHostNameError
			}
			//渲染实例的HostName
			for index, instance := range instances.Instances {
				instance.HostnameOverride = instanceHostNames[index]
				allInstances = append(allInstances, instance)
			}
			return &InstanceList{instances.DesiredCapacity, instances.RequestId, allInstances}, nil
		}
		return &instances, err
	}
	klog.V(3).Infof("KCE CA list instances by ASGName: %s failed ,because: %v", asg.Name, err)
	return nil, err
}

// get start config settings by autoscale group name
func (a *autoScalingWrapper) GetInstanceTemplate(asg * kce_asg.KceAsg) (*KceTemplate, error) {
	info, err := a.FindTemplateByAsg(asg)
	if err == nil {
		var response TemplateResponse
		err = json.Unmarshal(info, &response)
		if err != nil {
			klog.Errorf("Json Unmarshal error: %v", err)
			return nil, err
		}
		if len(response.ScalingConfigurationSet) == 0 || response.ScalingConfigurationSet[0] == nil {
			klog.Errorf("ASG %s, template ScalingConfigurationSet is nil.", asg.Name)
			return nil, fmt.Errorf("Asg %s template is nil. ", asg.Name)
		}
		klog.V(3).Infof("ASG %s template CPU : %d , Memory GB: %d , GPU : %d ", asg.Name, response.ScalingConfigurationSet[0].VCPU,
			response.ScalingConfigurationSet[0].MemoryGb, response.ScalingConfigurationSet[0].GPU)
		return &KceTemplate{
			InstanceType: &KceInstance{
							VCPU:     response.ScalingConfigurationSet[0].VCPU,
							GPU:      response.ScalingConfigurationSet[0].GPU,
							MemoryMb: response.ScalingConfigurationSet[0].MemoryGb * 1024,
						   },
			Labels: a.FindLabelsByAsg(asg),
			Tags: a.FindTaintsByAsg(asg),
		}, nil
	}
	klog.Error("Get instance template by ASG: " + asg.Name + " failed , because :" + err.Error())
	return nil, err

}

func (a *autoScalingWrapper) ValidateAsgById(asg * kce_asg.KceAsg) bool {
	var err error
	b, err := a.ValidateAsg(asg)
	if err != nil {
		klog.Errorf("Invalid ASG %s, error: %v.", asg.Name, err)
		return false
	}
	var response CheckResponse
	err = json.Unmarshal(b, &response)
	if err != nil {
		klog.Errorf("Invalid ASG %s, error: %v.", asg.Name, err)
		return false
	}
	if !response.Return {
		klog.Errorf("Invalid ASG %s, error: ASG not exist.", asg.Name)
		return false
	}

	data, err := a.CheckAutoScalerCanSell([] * kce_asg.KceAsg{asg})
	if err != nil {
		klog.Errorf("Invalid ASG %s, error: %v.", asg.Name, err)
		return false
	}
	var resp CheckCanSellResponse
	err = json.Unmarshal(data, &resp)
	if err != nil {
		klog.Errorf("Invalid ASG %s, error: %v.", asg.Name, err)
		return false
	}
	//if len(resp.CanSellAsgSet) == 0 || !resp.CanSellAsgSet[0].CanSell{
	if len(resp.CanSellAsgSet) == 0 || resp.CanSellAsgSet[0].CanSell!="Active" {
		klog.Errorf("Invalid ASG %s, error: ASG can't sell.", asg.Name)
		return false
	}
	return true
}

func (a *autoScalingWrapper) FindLabelsByAsg(asg *kce_asg.KceAsg) map[string]string {
	labels :=  AutoScalerGroupLabel(asg)
	if len(labels)!=0{
		for label:=range labels{
			klog.V(3).Infof("ASG label %s",label)
			klog.V(3).Infof(label,labels[label])
		}
	}
	
	return  labels
}

func  AutoScalerGroupLabel(asg *kce_asg.KceAsg) map[string]string  {
	_, label := kce_asg.GetNodeGroupNameAndZone(asg)
	if len(label)==0{
		return nil
	}
	labels := strings.Split(label, ",")
	labelMap := make(map[string]string)
	for _,s:=range labels{
		label_key_value := strings.Split(s, "=")
		labelMap[label_key_value[0]]=label_key_value[1]
	}
	return  labelMap
}

func (a *autoScalingWrapper) FindTaintsByAsg(asg * kce_asg.KceAsg) []*autoscaling.TagDescription {
	var err error
	info, err := a.ListTaintsByAsg(asg)
	if err == nil {
		var response TaintResponse

		err = json.Unmarshal(info, &response)
		if err != nil {
			klog.V(3).Infof("Find taints info error: %v", err)
			return nil
		}
		taints := strings.Split(response.ReturnMessage, ",")
		nodeTaints := make([]*autoscaling.TagDescription, len(taints))
		for _, s := range taints {
			tmp := strings.Split(s, "=")
			if len(tmp) != 2 {
				klog.V(3).Infof("ASG - %s with invalid Taint: %s \n", s)
				continue
			}
			nodeTaints = append(nodeTaints, &autoscaling.TagDescription{Key: &tmp[0], Value: &tmp[1]})
			klog.V(3).Infof("ASG - %s with Taint: %s=%s \n", asg.Name, tmp[0], tmp[1])
		}
		return nodeTaints
	}
	return nil
}

//func (a *autoScalingWrapper) DetachInstances(asg *kce_asg.KceAsg, instanceIds []string, hostNames []string) error {
//	var DeleteInstancesIds []string
//	//vms, err := a.ListInstancesByAsg(asg)
//	vms, err := a.DescribeScalingInstance(instanceIds,[]int64{asg.ProjectId})
//	if err == nil {
//		var instances InstanceList
//		err = json.Unmarshal(vms, &instances)
//		if err != nil {
//			return err
//		}
//		if len(instances.Instances) > 0 {
//			for _, instance := range instances.Instances {
//				if instance.ProtectedFromScaleIn == 0{
//					DeleteInstancesIds = append(DeleteInstancesIds,instance.ID)
//				}
//			}
//		}
//		if len(instances.Instances) <= 0 {
//			return fmt.Errorf("Invalide instanceId. ")
//		}
//	}else{
//		klog.V(3).Infof("Describe instances by ASGName: %s failed ,because: %v", asg.Name, err)
//		return  err
//	}
//	_, err = a.DetachInstancesById(asg, DeleteInstancesIds)
//	if err == nil {
//		for _, hostName := range hostNames {
//			err := a.externalClient.CoreV1().Nodes().Delete(context.TODO(),hostName, metav1.DeleteOptions{})
//			if(err != nil){
//				klog.Errorf("Delete node %s from cluster error: %v", hostName, err)
//				return err
//			}
//		}
//	}
//	return err
//}

func (a *autoScalingWrapper) DetachInstances(asg * kce_asg.KceAsg, instanceName []string, hostNames []string) error {
	var DeleteInstances []string
	vms, err := a.ListInstancesByAsg(asg)
	if err == nil {
		var instances InstanceList
		err = json.Unmarshal(vms, &instances)
		if err != nil {
			return err
		}
		count := instances.DesiredCapacity
		set := make(map[string]*InstancesSet, count)
		if len(instances.Instances) > 0 {
			for _, instance := range instances.Instances {
				if instance.ProtectedFromScaleIn==0{
					set[instance.ID] = instance
				}
			}
		}
		for _,ins := range instanceName{
			_, ok := set[ins]
			if ok{
				DeleteInstances = append(DeleteInstances,ins)
			}
		}
	}else{
		klog.V(3).Infof("KCE CA list instances by ASGName: %s failed ,because: %v", asg.Name, err)
	}
	_, err = a.DetachInstancesById(asg, DeleteInstances)
	if err == nil {
		for _, hostName := range hostNames {
			err := a.externalClient.CoreV1().Nodes().Delete(context.TODO(),hostName, metav1.DeleteOptions{})
			if(err != nil){
				klog.Errorf("Delete node %s from cluster error: %v", hostName, err)
				return err
			}
		}
	}
	return err
}
