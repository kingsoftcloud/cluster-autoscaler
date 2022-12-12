package services

import (
	"fmt"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/klog/v2"
)
func (client *Client) ListInstancesByAsgs2018(asg *kce_asg.KceAsg) ([]byte, error) {
	query := "Action=DescribeAutoScalerAllVm&Version=" + openApiVersion + "&AutoScalerGroupId=" + client.AutoScalerGroupId(asg) + "&ClusterId=" + client.ClusterId
	return DoRequest(client, query, "")
}

func (client *Client) ListInstancesByAsg(asg *kce_asg.KceAsg) ([]byte, error) {
	query := "Action=DescribeScalingInstance&Version=" + openApiVersion2016 + "&ScalingGroupId=" + client.AutoScalerGroupId(asg)
	return DoRequest2016(client, query, "")
}

func (client *Client) DescribeScalingInstance(InstanceIds []string,projectIds []int64) ([]byte, error) {
	//query := "Action=DescribeInstances&Version=" + openApiVersion2016 + "&InstanceId.1=" + id
	query := "Action=DescribeInstances&Version=" + openApiVersion2016
	if len(InstanceIds)==0{
		klog.V(5).Infof("Please check the instanceId",InstanceIds)
		return nil,nil
	}
	for index, id := range InstanceIds {
		query = query + fmt.Sprintf("&InstanceId.%d=%s", index+1, id)
	}
	for index, id := range projectIds {
		query = query + fmt.Sprintf("&ProjectId.%d=%d", index+1, id)
	}
	klog.V(5).Infof("query is %s",query)
	return DoRequest2016(client, query, "")
}
