package services

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-cloud-client/kce_client"
	"k8s.io/klog/v2"
	"net/url"
)
const (
	openApiVersion string = "2018-03-14"
	openApiVersion2016 string = "2016-03-04"
)

type DesiredCapacityBody struct {
	AvailabilityZone   string
	AutoScalingGroupId string
	DesiredCapacity    string
	ClusterId          string
}


func (client *Client) AutoScalerGroupId(asg *kce_asg.KceAsg) string {
	id, _ := kce_asg.GetNodeGroupNameAndZone(asg)
	return id
}
func (client *Client) AutoScalerGroupZone(asg *kce_asg.KceAsg) string {
	_, zone := kce_asg.GetNodeGroupNameAndZone(asg)
	return zone
}
func (client *Client) SetDesiredCapacity(input *kce_asg.SetDesiredCapacityInput, asg *kce_asg.KceAsg) ([]byte, error){
	query := "Action=ModifyClusterAsg&Version=" + openApiVersion  //"2018-03-14"-
	klog.Info()
	postBody := DesiredCapacityBody{
		AutoScalingGroupId: client.AutoScalerGroupId(asg), //id, _ := GetNodeGroupNameAndZone(asg) 根据ASG的name获取NodeGroupID
		DesiredCapacity:    kce_client.StringValue(input.DesiredCapacity),
		ClusterId:          client.ClusterId,
	}
	body := url.Values{}
	//参数设置
	body.Set("ClusterId", postBody.ClusterId)
	body.Set("AutoScalingGroupId", client.AutoScalerGroupId(asg))
	body.Set("DesiredCapacity", postBody.DesiredCapacity)
	body.Set("AvailabilityZone", client.AutoScalerGroupZone(asg))
	klog.Infof("\n####### Modify DesiredCapacity in ASG %s ####### \n\tDesiredCapacity: %s \n\tAvailabilityZone: %s", client.AutoScalerGroupId(asg), postBody.DesiredCapacity, client.AutoScalerGroupZone(asg))
	//返回response结果
	return DoRequest(client, query, body.Encode())

}


func (client *Client) ModifyScalingGroup(input *kce_asg.SetDesiredCapacityInput, asg *kce_asg.KceAsg) ([]byte, error){
	query := "Action=ModifyScalingGroup&Version=" + openApiVersion2016 + "&ScalingGroupId=" + client.AutoScalerGroupId(asg)+"&DesiredCapacity=" + *(input.DesiredCapacity)
	return DoRequest2016(client, query, "")
}




