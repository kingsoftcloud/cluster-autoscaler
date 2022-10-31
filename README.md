# 一、kubeadm集群搭建

## 1.1 master节点搭建

在云服务器（Kingsoft Cloud Elastic Compute，简称KEC）控制台新建一台云主机，以我创建的IP为[10.5.34.85](http://10.5.34.85:6443)的云主机为例。

## 1.2 环境配置

在IP为[10.5.34.85](http://10.5.34.85:6443)的云主机上进行如下环境配置

## 安装docker并登录金山云镜像仓库

```Go
# 设置存储库
# 安装yum-utils包（提供yum-config-manager 实用程序）并设置稳定存储库。
$ sudo yum install -y yum-utils
$ sudo yum-config-manager \
    --add-repo \
    https://download.docker.com/linux/centos/docker-ce.repo

# 安装最新版本的 Docker Engine 和 containerd
$ sudo yum install docker-ce docker-ce-cli containerd.io

# 查看docker-ce支持版本
$ sudo yum list docker-ce --showduplicates|sort -r
#查看docker-ce-cli版本
yum list docker-ce-cli --showduplicates|sort -r
# 指定版本号安装
$ sudo yum install -y docker-ce-19.03.13 docker-ce-cli-19.03.13 containerd.io

#启动docker
$ sudo systemctl start docker
#通过运行hello-world 映像验证 Docker Engine 是否已正确安装。
$ sudo docker run hello-world
#设置开机定时启动docker
$ sudo systemctl enable docker
```

登录金山云镜像仓库：

docker login -u {username} -p {password} hub.kce.ksyun.com

如果登录失败，请检查hosts配置:

ping  hub.kce.ksyun.com

 /etc/hosts

## K8s安装包准备

yum install kubelet-1.20.0 kubeadm-1.20.0 kubectl-1.20.0 --disableexcludes=kubernetes

## 初始化K8s集群

 kubeadm init  --kubernetes-version=v1.20.0  --pod-network-cidr=10.244.0.0/16 --service-cidr=10.96.0.0/12 --apiserver-advertise-address=10.5.34.85（这是服务器的IP,需要替换为你的服务器IP）

假如初始化成功，会有提示如下：

```Go
Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 10.5.34.85:6443 --token mfxeqg.027w04r1p1b72xwt \
    --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10 
```

按照该提示，执行以下命令：

```Go
  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

## 生成永久token和hash值

执行以下两行命令

```Go
#生成token值
kubeadm token create --ttl 0
#生成hash值
openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'
```

这样，以后在其他服务器上执行以下命令，就可以将该服务器加入集群了。

```Go
kubeadm join 10.5.34.85:6443 --token 上面生成的tokan值 --discovery-token-ca-cert-hash sha256:上面生成的hash值
```

例如：

```Go
kubeadm join 10.5.34.85:6443 --token mfxeqg.027w04r1p1b72xwt --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10
```

## 安装网络插件

wget  <https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml> 

kubectl apply -f kube-flannel.yml

## 1.3 制作镜像

在KEC控制台，首先点击IP为[10.5.34.85](http://10.5.34.85:6443)的云主机一栏的 更多按钮->制作镜像按钮，然后填写镜像名称等，开始制作镜像。我创建的镜像名称为kube-scheduler。

![img](https://kingsoft-cloud.feishu.cn/space/api/box/stream/download/asynccode/?code=NGM3MzEzMTkwMmFlNTg2ZjMyZWQ3ZjllYzk2ZTRjMjVfMDNsM3U3S3hXTVJ0U1M4bmM5cjM4M3VCalBHMlE0ZkZfVG9rZW46Ym94Y25FWlJ3U1c2ZDNFN09SMUJ2SWRqS3NnXzE2NjcxOTM1MDQ6MTY2NzE5NzEwNF9WNA)

制作镜像的目的是为了让云主机自创建成功以后，就包含docker kubeadm等环境，省去了以上环境配置的步骤。后通过结合启动配置中的UserData（提供给实例启动时使用的用户自定义数据）一起使用，可以实现ca组件扩容的主机一经创建，就可以自动加入集群，打标签等功能。

# 二、创建CA镜像

下载代码

```Go
git clone -b cluster-autoscaler-kce-1.20.3 https://github.com/kingsoftcloud/cluster-autoscaler.git
```

对于amd64架构：切换到 cluster-autoscaler/cluster-autoscaler文件夹下，执行以下命令

```Go
docker run --rm -e GOARCH=amd64  -e GO111MODULE=auto -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler  golang:1.18.3  go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/amd64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go

docker build -t kse/cluster-autoscaler:vv31 ./build/docker/amd64

docker tag kse/cluster-autoscaler:vv31  hub.kce.ksyun.com/golang/ca-new:vv31

docker push hub.kce.ksyun.com/golang/ca-new:vv31
```

命令解释：

```Go
1. Build the cluster-autoscaler binary:
docker run --rm -e GOARCH=amd64  -e GO111MODULE=auto -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler  golang:1.18.3  go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/amd64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go
2. Build the docker image:
docker build -t kse/cluster-autoscaler:version ./build/docker/amd64
3. tag
docker tag kse/cluster-autoscaler:version  hub.kce.ksyun.com/golang/ca-new:version
4. Push the docker image to KCR:
docker push hub.kce.ksyun.com/golang/ca-new:version
```

对于arm64架构:切换到 cluster-autoscaler/cluster-autoscaler文件夹下，执行以下命令

```Go
docker run --rm -e GOARCH=arm64  -e GO111MODULE=auto -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler  golang:1.18.3  go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/arm64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go
docker build -t kse/cluster-autoscaler:version ./build/docker/arm64
docker tag kse/cluster-autoscaler:version  hub.kce.ksyun.com/golang/ca-new:version
docker push hub.kce.ksyun.com/golang/ca-new:version
```

# 三、创建伸缩组

在AS控制台(https://kec.console.ksyun.com/as/#/create/group/)创建伸缩组，包含以下两个步骤：

## 3.1 创建启动配置

选择机型和镜像,其中镜像选择自定义镜像，选择名称为kube-scheduler的镜像。

用户自定义数据(提供给实例启动时使用的用户自定义数据),支持的最大数据大小为 16KB。

![img](https://kingsoft-cloud.feishu.cn/space/api/box/stream/download/asynccode/?code=NzZkYjdmYjgyZjY4NDZlMjY3ODhjN2RkZWZmODYwZGVfWjhBU0tjRmRVTjFRMHJtUEN0d0FKNkcxMGtaSXhFbmJfVG9rZW46Ym94Y25iYWx4bUVJdkFCd0VPM1h6c3gzRE1oXzE2NjcxOTM1MDQ6MTY2NzE5NzEwNF9WNA)

```Go
#!/bin/sh
cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1
kubeadm reset -f                   
rm -rf $HOME/.kube
kubeadm join 10.5.34.85:6443 --token mfxeqg.027w04r1p1b72xwt     --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10
echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile
source /etc/profile
kubectl label node $HOSTNAME label=label1
```

详细解释：

```Go
#!/bin/sh
#备份/etc/kubernetes/admin.conf文件
cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1
#执行kubeadm join前执行kubeadm reset -f 指令
kubeadm reset -f                   
rm -rf $HOME/.kube
#将该云主机加入集群
kubeadm join 10.5.34.85:6443 --token mfxeqg.027w04r1p1b72xwt     --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10
# 将kubeconfig的路径保存到环境变量里
echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile
# 使环境变量生效
source /etc/profile
# 如果您的ASG（伸缩组）需要带一个或多个标签，请执行以下命令
kubectl label node $HOSTNAME label=label
kubectl label node $HOSTNAME label=label1
```

## 3.2 创建伸缩组

![img](https://kingsoft-cloud.feishu.cn/space/api/box/stream/download/asynccode/?code=ZGI4OTRiMGUyMzY3Yzk3Mjg1YjlhYTY2ZjgwNmM5M2RfajF5THhEbG5tSFk4Rm1iUERqbjZiUG1QVDhHSm5LWHVfVG9rZW46Ym94Y25nZnpQeGQ2MUk4bk5iTjVvVng0cjJiXzE2NjcxOTM1MDQ6MTY2NzE5NzEwNF9WNA)

在控制台可以看到我创建的伸缩组的ID为784216551637458944

## 3.3 验证伸缩组有效性

查看ID为784216551637458944的ASG中的这一台云主机是否成功加入集群

![img](https://kingsoft-cloud.feishu.cn/space/api/box/stream/download/asynccode/?code=NTRiOWRmOWFhNGQyMjFiNTI1ZjBlZWI2ZThmMGQ4ZDhfcFFRZ3RON2h0cUF5Z1kxdHFTNHpuSnh0eFhuUEk3QmhfVG9rZW46Ym94Y25scjFMeEFaaDZDUEx4WGFxZm4zU3VoXzE2NjcxOTM1MDQ6MTY2NzE5NzEwNF9WNA)

假如失败了，请检查上述操作规范性。也可以登录到该云主机上，通过查看/var/log/cloud-init.log的文件内容，分析UserData没有生效的原因。需要注意的是：请不要直接对启动配置中的UserData数据进行修改，因为这不会生效。

# 四、创建CA Deployment

## 4.1 创建Secret

创建AK SK的方法参见https://docs.ksyun.com/documents/40311

地域（region）代码：

   ![img](https://kingsoft-cloud.feishu.cn/space/api/box/stream/download/asynccode/?code=YjFjZjM2NTVlZGYyNGQ3YTM3M2QwZDczYTQyNTIyNTBfQ0lnNWtqd3dpYUFLZDFsblZPZU9YcE5kZEpuQnEyU0JfVG9rZW46Ym94Y25VbnJMSUpYcG5yc3J3WEZ1RWRMbFdjXzE2NjcxOTM1MDQ6MTY2NzE5NzEwNF9WNA)

```Go
    apiVersion: v1
    kind: Secret
    metadata:
      name: cloud-config
      namespace: kube-system
    data:
      # insert your base64 encoded Kcecloud access id and key here, ensure there's no trailing newline:
      # such as:  echo -n "your_access_key_id" | base64
      access-key-id: "<BASE64_ACCESS_KEY_ID>"
      access-key-secret: "<BASE64_ACCESS_KEY_SECRET>"
      region-id: "<BASE64_REGION_ID>"
```
执行命令：
    kubectl apply -f cloud-config.yml

   ## 4.2 创建ServiceAccount和Role等

```Go
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      labels:
        k8s-addon: cluster-autoscaler.addons.k8s.io
        k8s-app: cluster-autoscaler
      name: cluster-autoscaler
      namespace: kube-system
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      labels:
        k8s-addon: cluster-autoscaler.addons.k8s.io
        k8s-app: cluster-autoscaler
      name: cluster-autoscaler
    rules:
      - apiGroups:
          - coordination.k8s.io
        resources:
          - leases
        verbs:
          - create
      - apiGroups:
          - coordination.k8s.io
        resourceNames:
          - cluster-autoscaler
        resources:
          - leases
        verbs:
          - get
          - update
          - patch
          - delete
      - apiGroups:
          - ""
        resources:
          - events
          - endpoints
        verbs:
          - create
          - patch
          - update
      - apiGroups:
          - ""
        resources:
          - pods/eviction
        verbs:
          - create
      - apiGroups:
          - ""
        resources:
          - pods/status
        verbs:
          - update
      - apiGroups:
          - ""
        resourceNames:
          - cluster-autoscaler
        resources:
          - endpoints
        verbs:
          - get
          - update
          - patch
          - delete
      - apiGroups:
          - ""
        resources:
          - nodes
        verbs:
          - watch
          - list
          - get
          - update
          - patch
      - apiGroups:
          - ""
        resources:
          - configmaps
        verbs:
          - watch
          - list
          - get
          - update
      - apiGroups:
          - ""
        resources:
          - pods
          - services
          - replicationcontrollers
          - persistentvolumeclaims
          - persistentvolumes
        verbs:
          - watch
          - list
          - get
      - apiGroups:
          - extensions
        resources:
          - replicasets
          - daemonsets
        verbs:
          - watch
          - list
          - get
      - apiGroups:
          - policy
        resources:
          - poddisruptionbudgets
        verbs:
          - watch
          - list
      - apiGroups:
          - apps
        resources:
          - statefulsets
          - replicasets
          - daemonsets
        verbs:
          - watch
          - list
          - get
      - apiGroups:
          - storage.k8s.io
        resources:
          - storageclasses
        verbs:
          - get
          - list
          - watch
      - apiGroups:
          - batch
        resources:
          - jobs
        verbs:
          - get
          - list
          - watch
    
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      labels:
        k8s-addon: cluster-autoscaler.addons.k8s.io
        k8s-app: cluster-autoscaler
      name: cluster-autoscaler
      namespace: kube-system
    rules:
      - apiGroups:
          - ""
        resources:
          - configmaps
        verbs:
          - create
          - list
          - watch
      - apiGroups:
          - ""
        resourceNames:
          - cluster-autoscaler-status
          - cluster-autoscaler-priority-expander
        resources:
          - configmaps
        verbs:
          - delete
          - get
          - update
          - watch
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      labels:
        k8s-addon: cluster-autoscaler.addons.k8s.io
        k8s-app: cluster-autoscaler
      name: cluster-autoscaler
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: cluster-autoscaler
    subjects:
      - kind: ServiceAccount
        name: cluster-autoscaler
        namespace: kube-system
    
    
    ---
    apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      labels:
        k8s-addon: cluster-autoscaler.addons.k8s.io
        k8s-app: cluster-autoscaler
      name: cluster-autoscaler
      namespace: kube-system
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: cluster-autoscaler
    subjects:
      - kind: ServiceAccount
        name: cluster-autoscaler
        namespace: kube-system
```
执行以下命令：
   kubectl apply -f role.yml

   ## 4.3 创建Deployment

```Go
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      labels:
        app: cluster-autoscaler
      name: cluster-autoscaler
      namespace: kube-system
    spec:
      replicas: 1
      revisionHistoryLimit: 10
      selector:
        matchLabels:
          app: cluster-autoscaler
      strategy:
        rollingUpdate:
          maxSurge: 25%
          maxUnavailable: 25%
        type: RollingUpdate
      template:
        metadata:
          labels:
            app: cluster-autoscaler
        spec:
          containers:
            - command:
                - ./cluster-autoscaler
                - --v=5
                - --logtostderr=true
                - --cloud-provider=kce
                - --expander=random
                - --scale-down-enabled=true
                - --skip-nodes-with-local-storage=false
                - --stderrthreshold=info
                - --nodes=MinSize:MaxSize:ASG_ID@label=value,label1=value1
                - --nodes=MinSize:MaxSize:ASG_ID
              env:
                - name: endpoint
                  value: http://internal.api.ksyun.com/
                - name: ACCESS_KEY_ID
                  valueFrom:
                    secretKeyRef:
                      key: access-key-id
                      name: cloud-config
                - name: ACCESS_KEY_SECRET
                  valueFrom:
                    secretKeyRef:
                      key: access-key-secret
                      name: cloud-config
                - name: REGION_ID
                  valueFrom:
                    secretKeyRef:
                      key: region-id
                      name: cloud-config
              image: hub.kce.ksyun.com/golang/ca-new:git8
              imagePullPolicy: Always
              name: cluster-autoscaler
              resources: {}
              terminationMessagePath: /dev/termination-log
              terminationMessagePolicy: File
          dnsPolicy: ClusterFirst
          #nodeSelector:
            #kubernetes.io/role: node1
          restartPolicy: Always
          schedulerName: default-scheduler
          securityContext: {}
          serviceAccount: cluster-autoscaler
          serviceAccountName: cluster-autoscaler
          terminationGracePeriodSeconds: 30
          tolerations:
            - operator: Exists
```


在执行kubectl apply -f ca.yml之前，请根据您创建的ASG的ID等信息更新nodes=MinSize:MaxSize:ASG_ID@label=value,label1=value1

|                         | nodes                                                        | UserData                                                     |
| ----------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| 设置1个ASG              | - --nodes=0:10:4343545432                                    | #!/bin/sh cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1 kubeadm reset -f                    rm -rf $HOME/.kube kubeadm join [10.5.34.85:6443](http://10.5.34.85:6443) --token mfxeqg.027w04r1p1b72xwt     --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10 echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile source /etc/profile |
| 设置多个ASG时           | - --nodes=0:10:4343545432 - --nodes=0:10:4396783032          | 同上                                                         |
| 设置带有一个标签的ASG时 | - --nodes=0:10:4343545432@label=value                        | #!/bin/sh cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1 kubeadm reset -f                    rm -rf $HOME/.kube kubeadm join [10.5.34.85:6443](http://10.5.34.85:6443) --token mfxeqg.027w04r1p1b72xwt     --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10 echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile source /etc/profile kubectl label node $HOSTNAME label=value |
| 设置带有多个标签的ASG时 | - --nodes=0:10:4343545432@label=value,label1=value1多个标签之间使用逗号分割 | #!/bin/sh cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1 kubeadm reset -f                    rm -rf $HOME/.kube kubeadm join [10.5.34.85:6443](http://10.5.34.85:6443) --token mfxeqg.027w04r1p1b72xwt     --discovery-token-ca-cert-hash sha256:9480fe409b09c917c2d435e2543cf9f4ccfb2a5cb2c5da8a4e8dd5e9aab3df10 echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile source /etc/profile kubectl label node $HOSTNAME label=valuekubectl label node $HOSTNAME label1=value1 |

其他参数：

​     在节点上运行的所有 pod 的 cpu、memory的总和 < 节点可分配总额的 50%时自动缩容,其中一个Node从检查出空闲，持续10min时间内依然空闲，才会被真正移除（所有参数都可定制）

| --max-empty-bulk-delete=10            | 最大缩容并发数，可以同时缩容的最大空节点数，如果存在pod，每次缩容最多一个节点 |
| ------------------------------------- | ------------------------------------------------------------ |
| --scale-down-delay-after-add=10m0s    | 集群扩容10分钟后，开始判断缩容条件                           |
| --scale-down-enabled=true             | 开启缩容                                                     |
| --scale-down-unneeded-time=10m0s      | 节点满足缩容条件10分钟后，开始缩容                           |
| --scale-down-utilization-threshold=50 | 节点已经分配的资源占可分配资源<50%时，开始判断缩容条件       |
| --max-total-unready-percentage=45     | Maximum percentage of unready nodes in the cluster.  After this is exceeded, CA halts operations被ca组件标记为unready的node数量超过总node数量的45%，ca组件停止工作执行kubectl get cm cluster-autoscaler-status -n kube-system -o yaml命令，能够看到被ca组件标记为unready的node数量 |
| --ok-total-unready-count=3            | Number of allowed unready nodes, irrespective of max-total-unready-percentage允许的unready节点数，不考虑max-total-unready-percentage |

## 4.4 验证

 kubectl get deployment -n kube-system

 kubectl get pod -l app=cluster-autoscaler -n kube-system

查看日志

 kubectl logs -f cluster-autoscaler-pod-name -n kube-system

修改deployment

kubectl edit deploy cluster-autoscaler -n kube-system

缩容相关日志

kubectl logs -f cluster-autoscaler-5df855f68b-2qwkx -n kube-system|grep "scale down" 扩容相关日志

kubectl logs -f cluster-autoscaler-5df855f68b-2qwkx -n kube-system|grep "scale_up.go

## 4.5 测试

```Go
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mylabel-pod
  namespace: test
  labels:
    app: ratings
    version: v1
spec:
  replicas: 40
  selector:
    matchLabels:
      app: ratings
      version: v1
  template:
    metadata:
      labels:
        app: ratings
        version: v1
    spec:
      containers:
      - name: ratings
        image: hub.kce.ksyun.com/kasm-public/examples-bookinfo-ratings-v1:1.16.2
        imagePullPolicy: IfNotPresent
        resources: # 资源配额
         #limits: # 限制资源（上限）
            #cpu: "1" # CPU限制，单位是core数
            #memory: "1Gi" # 内存限制
          requests: # 请求资源（下限）
            #cpu: "1" # CPU限制，单位是core数
            memory: "600Mi" #~
```

kubectl apply -f books.yml

集群中出现pendding pod,将触发ca组件扩容出来一定数量的node。

kubectl delete -f books.yml

一段时间后（根据对应的缩容时间设置），新扩容出来的node将会被ca组件移出集群。

# 五、注意事项：

手动在AS控制台(而不是ca组件缩容)移除ASG里的node以后，该node会被ca组件标记为unready状态，因此手动在AS控制台移除ASG里的node以后，请在集群中执行kubectl delete node node-name删除该 node。通过kubectl get cm cluster-autoscaler-status -n kube-system -o yaml命令可以观测到被ca组件标记为unready的node的数量。unready的node数量超过ca组件设置的限制，ca组件将会停止工作。详见https://github.com/kingsoftcloud/cluster-autoscaler/blob/main/cluster-autoscaler/FAQ.md。
