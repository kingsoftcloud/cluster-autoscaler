#!/bin/bash
set -e


GIT_SHA=`git rev-parse --short HEAD || echo "GitNotFound"`
version=$1
if [ -z $version ]; then
    version=`date "+%Y%m%d%H%M%S"`
    echo  "不指定版本号, 启用默认版本号: $version"
    echo "eg: build/build.sh $version"
    echo
fi
echo "version is : $version"
echo "git_sha is : $GIT_SHA"
echo "PWD is : $PWD"
# build
echo "go building cluster-autoscaler amd64 ..."
docker run --rm -e GOARCH=amd64   -e GO111MODULE=auto  -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler golang:1.18.3 go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/amd64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go


echo "go building cluster-autoscaler arm64 ..."
docker run --rm -e GOARCH=arm64  -e GO111MODULE=auto    -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler golang:1.18.3 go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/arm64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go

echo "docker building cluster-autoscaler amd64 image ..."
docker build -t hub.kce.ksyun.com/ksyun/cluster-autoscaler:"${version}"-amd64 ./build/docker/amd64
echo "docker building cluster-autoscaler arm64 image ..."
docker build -t hub.kce.ksyun.com/ksyun/cluster-autoscaler:"${version}"-arm64 ./build/docker/arm64

echo "docker pushing image ..."
docker push hub.kce.ksyun.com/ksyun/cluster-autoscaler:$version-amd64
docker push hub.kce.ksyun.com/ksyun/cluster-autoscaler:$version-arm64

echo "pushing mp image"
sed "s/VERSION/$version/g" ./build/docker/docker-manifest.yaml > ./build/docker/docker-manifest-"$version".yaml
manifest-tool  push  from-spec ./build/docker/docker-manifest-"$version".yaml
rm -rf ./build/docker/docker-manifest-"$version".yaml
