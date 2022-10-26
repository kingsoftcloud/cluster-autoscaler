package kce_asg

import "strings"
const (
	// ProviderName is the cloud provider name for Kingsoft
	ProviderName             = "kce"
	NodeGroupIdZoneSeparator = "@"
	NodeGroupIdTemplate      = "%s" + NodeGroupIdZoneSeparator + "%s"
)

type KceAsg struct {
	Name          string   `json:"name"`
	MinSize       int      `json:"min_size"`
	MaxSize       int      `json:"max_size"`
	AvailableZone []string `json:"available_zones"`
}

func GetNodeGroupNameAndZone(asg *KceAsg) (string, string) {
	if strings.Contains(asg.Name, NodeGroupIdZoneSeparator) {
		parts := strings.Split(asg.Name, NodeGroupIdZoneSeparator)
		return parts[0], parts[1]
	} else {
		return asg.Name, ""
	}
}


type SetDesiredCapacityInput struct {
	_ struct{} `type:"structure"`

	// The name of the Auto Scaling group.
	//
	// AutoScalingGroupName is a required field
	AutoScalingGroupName *string `min:"1" type:"string" required:"true"`

	// The number of EC2 instances that should be running in the Auto Scaling group.
	//
	// DesiredCapacity is a required field
	DesiredCapacity *string `type:"string" required:"true"`
}
