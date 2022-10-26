package services

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"io"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)
func buildOpenApiRequest(serviceName, region, body, securityToken string) (*http.Request, io.ReadSeeker) {
	reader := strings.NewReader(body)
	httpMethod := http.MethodGet
	if len(body) != 0 {
		reader = strings.NewReader(body)
		httpMethod = http.MethodPost
	}
	return buildOpenApiRequestWithBodyReader(serviceName, region, securityToken, reader, httpMethod)
}

func buildOpenApiRequestWithBodyReader(serviceName, region, securityToken string, body io.Reader, httpMethod string) (*http.Request, io.ReadSeeker) {
	var bodyLen int
	type lenner interface {
		Len() int
	}
	if lr, ok := body.(lenner); ok {
		bodyLen = lr.Len()
	}
	var endpoint string
	if os.Getenv("endpoint") == "" {
		endpoint = "https://kce.api.ksyun.com/?"
	} else {
		endpoint = os.Getenv("endpoint")
	}
	req, _ := http.NewRequest(httpMethod, endpoint, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if len(securityToken) > 0 {
		req.Header.Set("X-Ksc-Security-Token", securityToken)
	}

	if bodyLen > 0 {
		req.Header.Set("Content-Length", strconv.Itoa(bodyLen))
	}
	var seeker io.ReadSeeker
	if sr, ok := body.(io.ReadSeeker); ok {
		seeker = sr
	} else {
		seeker = nil
	}
	return req, seeker
}

func DoRequest(client *Client, query, postBody string) ([]byte, error) {
	s := v4.Signer{Credentials: credentials.NewStaticCredentials(client.CloudConfig.AccessKeyID, client.CloudConfig.AccessKeySecret, "")}
	req, body := buildOpenApiRequest("kce", client.RegionId, postBody, "client.CloudConfig.SecurityToken")
	if len(query) > 0 {
		req.URL.RawQuery = query
	}
	klog.V(3).Infof("Request url: %s", req.URL.String())
	_, err := s.Sign(req, body, "kce", client.RegionId, time.Now())
	if err != nil {
		klog.Errorf("Request Sign failed: %v", err)
		return nil, err
	}
	klog.V(5).Infof("Do HTTP Request: %v", req)
	resp, err := client.HttpClient.Do(req)
	if err != nil {
		klog.Errorf("HTTP Request failed: %v", err)
		return nil, err
	}
	statusCode := resp.StatusCode

	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		klog.Errorf("Get Response failed: %v", err)
		return nil, err
	}

	if statusCode >= 400 && statusCode <= 599 {
		klog.Errorf("Request url: %s , status code: %d , resp body: %s", req.RequestURI, statusCode, respBody)
		return respBody, fmt.Errorf("status code: %d, resp body: %s", statusCode, respBody)
	}

	return respBody, nil
}

func DoRequest2016(client *Client, query, postBody string) ([]byte, error) {
	s := v4.Signer{Credentials: credentials.NewStaticCredentials(client.CloudConfig.AccessKeyID, client.CloudConfig.AccessKeySecret, "")}
	req, body := buildOpenApiRequest("kec", client.RegionId, postBody, client.CloudConfig.SecurityToken)
	if len(query) > 0 {
		req.URL.RawQuery = query
	}
	klog.V(3).Infof("Request url: %s", req.URL.String())
	_, err := s.Sign(req, body, "kec", client.RegionId, time.Now())
	if err != nil {
		klog.Errorf("Request Sign failed: %v", err)
		return nil, err
	}
	klog.V(5).Infof("Do HTTP Request: %v", req)
	resp, err := client.HttpClient.Do(req)//err
	if err != nil {
		klog.Errorf("HTTP Request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	var statusCode = resp.StatusCode
	respBody, err := ioutil.ReadAll(resp.Body)//io.ReadCloser

	if err != nil {
		klog.Errorf("Get Response failed: %v", err)
		return nil, err
	}
	if statusCode >= 400 && statusCode <= 599 {
		klog.Errorf("Request url: %s , status code: %d , resp body: %s", req.RequestURI, statusCode, respBody)
		return respBody, fmt.Errorf("status code: %d, resp body: %s", statusCode, respBody)
	}

	return respBody, nil
}


