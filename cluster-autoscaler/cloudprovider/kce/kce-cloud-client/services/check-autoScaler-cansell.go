package services
//
import (
	"fmt"
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/kce-asg"
	"net/url"
)

func (client *Client) CheckAutoScalerCanSell(asgs []*kce_asg.KceAsg) ([]byte, error) {
	//params := url.Values{}
	//params.Add("Action", "CheckAutoScalerCanSell")
	//params.Add("Version", openApiVersion)
	//for index, asg := range asgs {
	//	params.Add(fmt.Sprintf("AutoScalingGroupId.%d", index+1), client.AutoScalerGroupId(asg))
	//}
	//return DoRequest(client, params.Encode(), "")

	params := url.Values{}
	params.Add("Action", "DescribeScalingGroup")
	params.Add("Version", openApiVersion2016)
	for index, asg := range asgs {
		params.Add(fmt.Sprintf("AutoScalingGroupId.%d", index+1), client.AutoScalerGroupId(asg))
	}
	return DoRequest2016(client, params.Encode(), "")
}
