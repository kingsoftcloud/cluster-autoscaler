package services

import (
	"fmt"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/klog/v2"
)

func (client *Client) DetachInstancesById(asg *kce_asg.KceAsg, instanceIDs []string) ([]byte, error) {
	query := fmt.Sprintf("Action=DetachInstance&Version=%s&ScalingGroupId=%s", "2016-03-04", client.AutoScalerGroupId(asg))
	for index, id := range instanceIDs {
		query = query + fmt.Sprintf("&ScalingInstanceId.%d=%s", index+1, id)
	}
	klog.Infof("\n####### Detach Instances in ASG %s ####### \n\tinstanceIDs: %v", client.AutoScalerGroupId(asg), instanceIDs)
	return DoRequest2016(client, query, "")
}


