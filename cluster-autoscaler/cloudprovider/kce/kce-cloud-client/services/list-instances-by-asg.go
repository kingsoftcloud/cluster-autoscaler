package services

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
)
func (client *Client) ListInstancesByAsgs2018(asg *kce_asg.KceAsg) ([]byte, error) {
	query := "Action=DescribeAutoScalerAllVm&Version=" + openApiVersion + "&AutoScalerGroupId=" + client.AutoScalerGroupId(asg) + "&ClusterId=" + client.ClusterId
	return DoRequest(client, query, "")
}

func (client *Client) ListInstancesByAsgs(asg *kce_asg.KceAsg) ([]byte, error) {
	query := "Action=DescribeScalingInstance&Version=" + openApiVersion2016 + "&ScalingGroupId=" + client.AutoScalerGroupId(asg)
	return DoRequest2016(client, query, "")
}

func (client *Client) DescribeScalingInstance(id string) ([]byte, error) {
	query := "Action=DescribeInstances&Version=" + openApiVersion2016 + "&InstanceId.1=" + id
	return DoRequest2016(client, query, "")
}
