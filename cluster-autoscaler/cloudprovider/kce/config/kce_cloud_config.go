/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	klog "k8s.io/klog/v2"
	"os"
	"strings"
)

const (
	accessKeyId     = "ACCESS_KEY_ID"
	accessKeySecret = "ACCESS_KEY_SECRET"
	regionId        = "REGION_ID"
)

type CloudConfig struct {
	RegionId        string
	AccessKeyID     string
	AccessKeySecret string
	STSEnabled      bool
	SecurityToken   string
}

func (config *CloudConfig) IsValid() bool {
	if config.AccessKeyID == "" {
		klog.V(3).Infof("os.Getenv(accessKeyId)", os.Getenv(accessKeyId))
		var str = strings.Replace(os.Getenv(accessKeyId), "\\n", "", -1)
		str = strings.Replace(str, "\n", "", -1)
		config.AccessKeyID = str
		klog.V(3).Infof("config.AccessKeyID", config.AccessKeyID)
	}

	if config.AccessKeySecret == "" {
		klog.V(3).Infof("os.Getenv(accessKeySecret)", os.Getenv(accessKeySecret))
		var str = strings.Replace(os.Getenv(accessKeySecret), "\\n", "", -1)
		str = strings.Replace(str, "\n", "", -1)
		config.AccessKeySecret = str
		klog.V(3).Infof("config.AccessKeySecret", config.AccessKeySecret)
	}

	if config.RegionId == "" {
		klog.V(3).Infof("os.Getenv(regionId)", os.Getenv(regionId))
		var str = strings.Replace(os.Getenv(regionId), "\\n", "", -1)
		str = strings.Replace(str, "\n", "", -1)
		config.RegionId = str
		klog.V(3).Infof("config.RegionId ", config.RegionId )
	}

	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		klog.V(5).Infof("Failed to get AccessKeyId:%s,AccessKeySecret:%s from CloudConfig and Env\n", config.AccessKeyID, config.AccessKeySecret)
		return false
	}

	return true
}

func (config *CloudConfig) GetRegion() string {
	if config.RegionId != "" {
		return config.RegionId
	}

	return ""
}
