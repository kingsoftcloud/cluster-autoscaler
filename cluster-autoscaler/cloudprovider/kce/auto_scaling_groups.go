package kce

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/klog/v2"
	"strings"
	//schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/framework"
)

// KceNodeGroup implements NodeGroup interface.
type KceNodeGroup struct {
	Asg        *kce_asg.KceAsg
	kceManager *KceManager
}

type ScaleUpConfigMapData struct {
	Expander string            `json:"expander"`
	Nodes    []*kce_asg.KceAsg `json:"nodes"`
}

type NetInfo struct {
	AvailableZone string   `json:"available_zone"`
	SubNets       []string `json:"subnets"`
}

type InstanceList2018 struct {
	DesiredCapacity string          `json:"DesiredCapacity"`
	RequestId       string          `json:"RequestId"`
	Instances       []*InstancesSet2018 `json:"InstancesSet"`
}

type InstancesSet2018 struct {
	//todo aws providerID like
	Name                   string `json:"InstanceName"`
	IP                     string `json:"PrivateIpAddress"` // deprecated , use `HostnameOverride` instead
	ID                     string `json:"InstanceId"`
	AvailableZone          string `json:"AvailabilityZone"`
	//HostnameOverride     string `json:"HostnameOverride"`
	HostnameOverride       string `json:"Hostname"`
	ProtectedFromScaleDown bool   `json:"ProtectedFromScaleDown"`
}

type InstanceList struct {
	DesiredCapacity int             `json:"DesiredCapacity"`
	RequestId       string          `json:"RequestId"`
	Instances       []*InstancesSet `json:"ScalingInstanceSet"`
}

type InstancesSet struct{
	//todo aws providerID like
	Name                   string `json:"InstanceName"`
	//HostnameOverride       string `json:"PrivateIpAddress"` // deprecated , use `HostnameOverride` instead
	HostnameOverride       string `json:"Hostname"` // deprecated , use `HostnameOverride` instead
	ID                     string `json:"InstanceId"`
	HealthStatus           string `json:"HealthStatus"`
	ProtectedFromScaleIn   int   `json:"ProtectedFromScaleIn"`
	ProtectedFromScaleDown bool   `json:"ProtectedFromScaleDown"`
}

func (ng *KceNodeGroup) WholeAvailabilityZones() []string {
	return ng.Asg.AvailableZone
}

func (ng *KceNodeGroup) AvailabilityZone() string {
	_, zone := kce_asg.GetNodeGroupNameAndZone(ng.Asg) //utils.go-
	return zone
}

// MaxSize returns maximum size of the node group.
func (ng *KceNodeGroup) MaxSize() int {
	return ng.Asg.MaxSize
}

// MinSize returns minimum size of the node group.
func (ng *KceNodeGroup) MinSize() int {
	return ng.Asg.MinSize
}

func (ng *KceNodeGroup) Create() (cloudprovider.NodeGroup, error) {
	return nil, cloudprovider.ErrAlreadyExist
}

// TargetSize returns the current TARGET size of the node group. It is possible that the
// number is different from the number of nodes registered in Kubernetes.
func (ng *KceNodeGroup) TargetSize() (int, error) {
	size, err := ng.kceManager.GetAsgTargetSize(ng.Asg)
	if err != nil {
		klog.V(0).Infof("Kce node group get target size error: %v", err)
		return 0, err
	}
	klog.V(0).Infof("Kce node group %s get target size : %d", ng.Asg.Name, size)
	return int(size), nil
}

// IncreaseSize increases Asg size
func (ng *KceNodeGroup) IncreaseSize(delta int) error {
	klog.V(0).Infof("Kce node group scale up node size : %d", delta)
	if delta <= 0 {
		return fmt.Errorf("kce node group scale up node size <= 0")
	}
	size, err := ng.kceManager.GetAsgTargetSize(ng.Asg)
	if err != nil {
		return fmt.Errorf("get kce node group %s target size error : %v", ng.Asg.Name, err)
	}
	klog.V(0).Infof("Kce node group %s current node size : %d", ng.Asg.Name, size)
	if size+delta > ng.Asg.MaxSize {
		return fmt.Errorf("kce node group size increase too large - desired:%d max:%d", size+delta, ng.Asg.MaxSize)
	}
	return ng.kceManager.SetKcgSize(ng, size+delta)
}

// DeleteNodes deletes nodes from this node group. Error is returned either on
// failure or if the given node doesn't belong to this node group. This function
// should wait until node group size is updated. Implementation required.
func (ng *KceNodeGroup) DeleteNodes(nodes []*apiv1.Node) error {
	refs := make([]string, 0)
	hostNames := make([]string, 0)
	for _, node := range nodes {
		instances, err := ng.kceManager.GetKceNodes(ng.Asg)
		if err != nil {
			return err
		}
		flag := false
		for _, instance := range instances {
			if node.Name == instance.HostnameOverride {
				flag = true
				refs = append(refs, instance.ID)
				hostNames = append(hostNames, node.Name)
				break
			}
		}
		if !flag {
			return fmt.Errorf("%s belongs to a different Asg than %s", node.Name, ng.Asg.Name)
		}
	}
	klog.Infof("Start to delete instances Id: %v", refs)
	klog.Infof("Start to delete instances hostNames: %v", hostNames)
	return ng.kceManager.DeleteInstances(ng.Asg, refs, hostNames)
}

// DecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the
// request for new nodes that have not been yet fulfilled. Delta should be negative.
// It is assumed that cloud provider will not delete the existing nodes if the size
// when there is an option to just decrease the target.
func (ng *KceNodeGroup) DecreaseTargetSize(delta int) error {
	return nil
}

// Id returns an unique identifier of the node group.
func (ng *KceNodeGroup) Id() string {
	return ng.Asg.Name
}

// Debug returns a string containing all information regarding this node group.
func (ng *KceNodeGroup) Debug() string {
	return fmt.Sprintf("%s (%d:%d)", ng.Asg.Name, ng.MinSize(), ng.MaxSize())
}

// Nodes returns a list of all nodes that belong to this node group.
func (ng *KceNodeGroup) Nodes() ([]cloudprovider.Instance, error) {
	cacheKey := fmt.Sprintf("%s/%s/%s", "KceNodeGroup", "Nodes", ng.Asg.Name)
	if obj, found := ng.kceManager.cache.Get(cacheKey); found {
		return obj.([]cloudprovider.Instance), nil
	}
	//m.service.InstancesByAsg(asg)
	asgNodes, err := ng.kceManager.GetKceNodes(ng.Asg)
	if err != nil {
		return nil, err
	}
	nodes := make([]cloudprovider.Instance, len(asgNodes))
	hostNames := make([]string, 0)
	for i, n := range asgNodes {
		hostNames = append(hostNames, n.HostnameOverride)
		nodes[i] = cloudprovider.Instance{
			Id:     n.HostnameOverride,
			Status: nil,
		}
	}
	ng.kceManager.cache.Set(cacheKey, nodes, 0)
	klog.V(0).Infof("KCE node group %s : vm node - %s  ", ng.Asg.Name, strings.Join(hostNames, ","))
	return nodes, nil
}

// TemplateNodeInfo returns a schedulercache.NodeInfo structure of an empty
// (as if just started) node. This will be used in scale-up simulations to
// predict what would a new node look like if a node group was expanded. The returned
// NodeInfo is expected to have a fully populated Node object, with all of the labels,
// capacity and allocatable information as well as all pods that are started on
// the node by default, using manifest (most likely only kube-proxy). Implementation optional.
func (ng *KceNodeGroup) TemplateNodeInfo() (*schedulernodeinfo.NodeInfo, error) {
	template, err := ng.kceManager.getKceTemplate(ng.Asg)
	if err != nil {
		return nil, err
	}
	klog.V(0).Infof("\n####### KCE ASG %s template #######"+
		"\n\tInstanceFlavor: %dC%dG\n\tContainer Labels: %s\n\tContainer Taints: %s",
		ng.Asg.Name, template.InstanceType.VCPU, (template.InstanceType.MemoryMb)/1024, template.Labels, template.Tags)

	node, err := ng.kceManager.buildNodeFromTemplate(ng.Asg, template)

	if err != nil {
		return nil, err
	}

	nodeInfo := schedulernodeinfo.NewNodeInfo(cloudprovider.BuildKubeProxy(ng.Asg.Name))
	nodeInfo.SetNode(node)

	return nodeInfo, nil
}

// Exist checks if the node group really exists on the cloud provider side. Allows to tell the
// theoretical node group from the real one.
func (ng *KceNodeGroup) Exist() bool {
	return ng.kceManager.ValidateASG(ng.Asg)
}

// Delete deletes the node group on the cloud provider side.
// This will be executed only for autoprovisioned node groups, once their size drops to 0.
func (ng *KceNodeGroup) Delete() error {
	return cloudprovider.ErrNotImplemented
}

// Autoprovisioned returns true if the node group is autoprovisioned.
func (ng *KceNodeGroup) Autoprovisioned() bool {
	return false
}
