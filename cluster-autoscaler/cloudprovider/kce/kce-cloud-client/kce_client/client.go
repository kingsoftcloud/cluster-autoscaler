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

package kce_client

import (
	"k8s.io/autoscaler/cluster-autoscaler/cloudprovider/kce/config"
	"net/http"
)

// Version value will be replaced while build: -ldflags="-X sdk.version=x.x.x"
var Version = "0.0.1"

// Client is common SDK client
type Client struct {
	RegionId       string
	CloudConfig    config.CloudConfig
	HttpClient     *http.Client
	ClusterId string
}
