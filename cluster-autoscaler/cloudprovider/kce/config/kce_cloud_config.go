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

func (config *CloudConfig) GetFromEnv(key string) string {
	val,ok := os.LookupEnv(key)
	if !ok || val == ""{
		return ""
	}
	klog.V(3).Infof("get value from env,key:%s,value:%s",key,val)
	val = strings.Replace(val, "\\n", "", -1)
	return strings.Replace(val, "\n", "", -1)
}

func (config *CloudConfig) IsValid() bool {
	if config.AccessKeyID == "" {
		config.AccessKeyID = config.GetFromEnv(accessKeyId)
	}

	if config.AccessKeySecret == "" {
		config.AccessKeyID = config.GetFromEnv(accessKeySecret)
	}

	if config.RegionId == "" {
		config.AccessKeyID = config.GetFromEnv(regionId)
	}

	if config.AccessKeyID == "" || config.AccessKeySecret == "" {
		klog.V(5).Infof("Failed to get AccessKeyId:%s,AccessKeySecret:%s from CloudConfig and Env\n", config.AccessKeyID, config.AccessKeySecret)
		return false
	}

	return true
}