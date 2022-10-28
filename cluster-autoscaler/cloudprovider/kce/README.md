# Cluster Autoscaler on Kingsoft Cloud 

## Overview
| Kubernetes Version | Cluster Autoscaler Version |
| ------------------ | ---------- |
| 1.20             | 1.20.3     |

The Cluster Autoscaler Version 1.20.X works with self-built Kubernetes cluster on [KEC](https://kec.console.ksyun.com/v2/#/kec) and
specified [Kingsoft Cloud Auto Scaling](https://kec.console.ksyun.com/as/#/group). It runs as a Deployment on a worker node in the cluster. This README will go over some of the necessary steps required to get the cluster autoscaler up and running.

## Deployment Steps
### Build cluster-autoscaler Image
#### Environment
1. Download Project

    Get the `cluster-autoscaler` project and download it. 
    
3. Docker environment

    Make sure you have Docker installed in the above machine.
    
#### Build and push the image
To get the cluster-autoscaler image, please execute the `./build.sh`  commands in the directory of `cluster-autoscaler/cluster-autoscaler` of the cluster-autoscaler project downloaded previously. More specifically,

1. Build the `cluster-autoscaler` binary:
    ```
    docker run --rm -e GOARCH=amd64(arm64)  -e GO111MODULE=auto -v $PWD:/go/src/k8s.io/autoscaler/cluster-autoscaler  golang:1.18.3  go build -o /go/src/k8s.io/autoscaler/cluster-autoscaler/build/docker/amd64/cluster-autoscaler /go/src/k8s.io/autoscaler/cluster-autoscaler/main.go
    ```
2. Build the docker image:
    ```
   docker build -t {Image repository address}/{Organization name}/{Image name:tag} ./build/docker/amd64(arm64)
   ```
   
3. Login to KCR:
    ```
    docker login -u {Encoded username} -p {Encoded password} {KCR endpoint}
    ```
    
4. Push the docker image to KCR:
    ```
    docker push {Image repository address}/{Organization name}/{Image name:tag}
    ```


The above steps use Kingsoft Cloud Container Registry  [KCR](https://kce.console.ksyun.com/#/image/repository) as an example registry.

## Build Kubernetes Cluster on [KEC](https://kec.console.ksyun.com/v2/#/kec) 

- Launch a new KEC instance as the master node.

### 1. Install kubelet, kubeadm and kubectl   

Please see installation [here](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)

### 2. Install Docker
Please see installation [here](https://docs.docker.com/engine/install/)

### 3. Initialize Cluster
```bash
kubeadm init

mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

### 4. Install Flannel Network
```bash 
kubectl apply -f https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
```
### 5. Generate Token

Generate a token that never expires. Remember this token since it will be used later.

```bash
kubeadm token create -ttl 0
```
Get hash key. Remember the key since it will be used later.
```
openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //'
```

### 6. Create OS Image with K8S Tools
- Launch a new KEC instance and it into the k8s cluster.

    ```bash
    kubeadm join --token $TOKEN $API_Server_EndPoint --discovery-token-ca-cert-hash $HASHKEY
    ```
- Copy `/etc/kubernetes/admin.conf` from master node to this KEC `/etc/kubernetes/admin.conf` to setup kubectl on this instance.

- Go to [KEC](https://kec.console.ksyun.com/v2/#/kec) service and select your KEC instance as source to create an OS image with K8S tools.


## Create AS Group
 Go to [Kingsoft Cloud Auto Scaling](https://kec.console.ksyun.com/as/#/group) service to create an AS Group.

- While creating the `AS Configuration`, please select private image which we just created and add the following script into `UserData`.
    ```bash
    #!/bin/sh
    cp  /etc/kubernetes/admin.conf /etc/kubernetes/admin.conf1
    kubeadm reset -f                   
    rm -rf $HOME/.kube
    kubeadm join --token $TOKEN $API_Server_EndPoint --discovery-token-ca-cert-hash $HASHKEY
    echo "export KUBECONFIG=/etc/kubernetes/admin.conf1" >> /etc/profile
    source /etc/profile
    #add labels if you need
    kubectl label node $HOSTNAME label=value
    kubectl label node $HOSTNAME label1=value1
    ```
    
     The script help to join the new instance into the k8s cluster automatically.
    
 - Bind the AS Group with this AS Configuration.

## Deploy Cluster Autoscaler Deployment
### 1. Prepare Identity authentication

​	 Use access-key-id and access-key-secret

```
apiVersion: v1
kind: Secret
metadata:
  name: cloud-config
  namespace: kube-system
data:
  # insert your base64 encoded Kcecloud access id and key here
  # such as:  echo -n "your_access_key_id" | base64
  access-key-id: "<BASE64_ACCESS_KEY_ID>"
  access-key-secret: "<BASE64_ACCESS_KEY_SECRET>"
  region-id: "<BASE64_REGION_ID>"
```

### 2. Configure cluster-autoscaler deployment

```
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
            - --nodes=[min]:[max]:[ASG_ID@label=value,label1=value1]
            - --nodes=[min]:[max]:[ASG_ID]
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
          image: {Image repository address}/{Organization name}/{Image name:tag}
          imagePullPolicy: Always
          name: cluster-autoscaler
          resources: {}
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
      dnsPolicy: ClusterFirst
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext: {}
      serviceAccount: cluster-autoscaler
      serviceAccountName: cluster-autoscaler
      terminationGracePeriodSeconds: 30
      tolerations:
        - operator: Exists

```


​     Change the image to the `cluster-autoscaler image` you just pushed. 

​     The `--nodes` parameter should match the parameters of the AS Group you created.

   ```
   {Minimum number of nodes}:{Maximum number of nodes}:{AS Group ID}
   ```

```
 {Minimum number of nodes}:{Maximum number of nodes}:{AS Group ID@label=value,label1=value1}
```

​    For AS Group with labels, please use the above format.

​    More configuration options can be added to the cluster autoscaler, such as `scale-down-delay-after-add`, `scale-down-unneeded-time`, etc. See available configuration options [here](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/FAQ.md#what-are-the-parameters-to-ca).

​    An example deployment file is provided at `kce/examples/cluster-autoscaler-standard.yaml`. 

### 3. Deploy cluster autoscaler on the cluster
Login to the master node and run the following command:

```
kubectl create -f cluster-autoscaler-standard.yaml
```
## Notes
If the `cluster-autoscaler` component not scale-up or scale-down for a long time, execute the [kubectl get cm cluster-autoscaler-status  -n kube-system -o yaml ] command to check whether the cluster status is healthy. When the number of unready nodes in the cluster reaches a certain level, the cluster will be unhealthy and the `cluster-autoscaler` component will stop work. Please refer to [How does CA deal with unready nodes?]( https://github.com/kingsoftcloud/cluster-autoscaler/blob/main/cluster-autoscaler/FAQ.md)

If you click the remove button on the Auto Scaling service console to remove the node from the scaling group, the node will be marked as unready by the `cluster-autoscaler` component. To avoid this risk, please execute the [kubectl delete node node_name] command in the cluster after the above removal operation.

If the UserData set in the AS configuration does not take effect, log in to the instance under the corresponding AS Group and check whether there are errors in the /var/log/cloud-init.log file. Also, please do not edit the UserData in the AS configuration on the Auto Scaling service console, because the changes will not take effect.

## Support & Contact Info

Interested in Cluster Autoscaler on KingSoft Cloud? Want to talk? Have questions, concerns or great ideas? Please reach out to us at `GUOTIANDI@kingsoft.com`.
