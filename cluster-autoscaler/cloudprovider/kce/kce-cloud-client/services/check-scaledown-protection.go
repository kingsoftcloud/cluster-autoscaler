package services

import (
	"fmt"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"strings"
)

func (client *Client) CheckScaleDownProtection(nodes []*apiv1.Node) ([]byte, error) {
	var kecInstances []string
	var queryParam strings.Builder
	//kubeadm注释内容
	for _, node := range nodes {
		//kubeadm集群没有"appengine.sdns.ksyun.com/instance-uuid"，改为使用SystemUUID
		SystemUUID:=node.Status.NodeInfo.SystemUUID
		klog.V(0).Infof("Get nodeinstanceuuid by node.Status.NodeInfo.SystemUUID : %s", SystemUUID)
		if SystemUUID!="" {
			kecInstances = append(kecInstances, SystemUUID)
		} else {
			klog.Warningf("ScaleDownProtectionCheck error: node %s without SystemUUID , skip ", node.Name)
		}
	}

	if len(kecInstances) == 0 {
		return []byte("[]"), nil
	}

	for i, val := range kecInstances {
		queryParam.WriteString(fmt.Sprintf("InstanceId.%d=%s", i, val))
		if i < len(kecInstances)-1 {
			queryParam.WriteString("&")
		}
	}
	query := "Action=CheckVmProtectedFromScaleDown&Version=" + openApiVersion + "&ClusterId=" + client.ClusterId + "&" + queryParam.String()
	return DoRequest(client, query, "")
}
