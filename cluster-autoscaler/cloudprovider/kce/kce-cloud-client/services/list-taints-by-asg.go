package services

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
)

func (client *Client) ListTaintsByAsgs(asg *kce_asg.KceAsg) ([]byte, error) {
	query := "Action=DescribeAutoScalerTaint&Version=" + openApiVersion + "&AutoScalerGroupId=" + client.AutoScalerGroupId(asg)
	return DoRequest(client, query, "")
}