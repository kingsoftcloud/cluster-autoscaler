package services

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
)

func(client *Client) GetProjectIdByAsg(asg * kce_asg.KceAsg) (data []byte, err error){
	templateData, err := client.FindTemplateByAsg(asg)
	if err!=nil{
		return nil,err
	}
	return templateData,nil
}
