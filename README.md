# kt-cloud-operator

This is an kubernetes operator used create a Kubernetes cluster on KT Cloud [here](https://gcloud.kt.com/console/). and its API documentation is found [here](https://cloud.kt.com/docs/open-api-guide/d/guide/how-to-use).

### **What works**

* Create cluster
* Delete cluster

### What Does not Work

* HA clusters
* Use LoadBalancers
* Auto-renew authentication token

### required configuration

#### Cloud-Setup

* Login into the cloud console
* Create your network tiers and virtual IPs
* Setup Public IP addresses
* Add Static routes for your tiers
* Create SSH Keypair for your VMs and download the key to your local computer
* Create a VM and with at least 50GB with Ubuntu 22.04 64bit and 2vcore 2GB
* After the VM gets ready, SSH into it and run the following commands on it

  ```
  sudo apt-get update
  sudo swapoff -a
  sudo sed -i '/\bswap\b/d' /etc/fstab
  sudo swapoff /swap.img
  sudo sysctl -w net.ipv4.ip_forward=1
  echo "net.ipv4.ip_forward = 1" | sudo tee -a /etc/sysctl.conf
  sudo sysctl -p
  echo -e "overlay\nbr_netfilter" | sudo tee /etc/modules-load.d/containerd.conf
  sudo modprobe overlay
  sudo modprobe br_netfilter
  sudo tee /etc/sysctl.d/kubernetes.conf <<EOF
  net.bridge.bridge-nf-call-ip6tables = 1
  net.bridge.bridge-nf-call-iptables = 1
  net.ipv4.ip_forward = 1
  EOF
  sudo apt install -y curl gnupg2 software-properties-common apt-transport-https ca-certificates
  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmour -o /etc/apt/trusted.gpg.d/docker.gpg
  sudo add-apt-repository -y "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
  sudo apt update
  sudo apt install -y containerd.io
  containerd config default | sudo tee /etc/containerd/config.toml >/dev/null 2>&1
  sudo sed -i 's/SystemdCgroup \= false/SystemdCgroup \= true/g' /etc/containerd/config.toml
  sudo systemctl restart containerd
  sudo systemctl enable containerd
  sudo apt-get update
  sudo apt-get install -y apt-transport-https ca-certificates curl gpg
  curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
  echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
  sudo apt-get update
  sudo apt-get install -y kubelet kubeadm kubectl
  sudo kubeadm config images pull
  sudo apt-mark hold kubelet kubeadm kubectl

  ```
* The scripts above installs kubernetes on the VM and holds the packages
* At this moment, take the VM snapshot to create a Kubernetes image from it which can be used to create a kubernetes node

#### Setup KRM CRs

* We need to obtain authentication subject-token manually and we have to curl the KT-Cloud APIs
* The guidance is found on [this](https://cloud.kt.com/docs/open-api-guide/d/guide/how-to-use) page
* After get the token, replace the ${SUBJECT_TOKEN} in /try-crds/infrastructure_v1beta1_ktsubjecttoken.yam file.
* '${CLUSTER_NAME}' has to be replaced in all files

##### Sample CRDs

* starting with try-crds/infrastructure_v1beta1_ktmachinetemplate.yaml, we have to modify the flavor, blockDeviceMapping, network tier and ssh key
  * For flavor, we have to use an API directly because it is not provided in the cloud console follow the guide on this [page](https://cloud.kt.com/docs/open-api-guide/d/computing/virtual-machine), on this API endpoint: https://api.ucloudbiz.olleh.com/gd1/server/flavors/detail
  * The blockingDeviceMapping data, can be taken from the console in servers/Image, click on the preferred image then information and get its ID. The other variables can be customized based on requirements
  * The networkTier.id is taken from this GET API: [https://api.ucloudbiz.olleh.com/gd1/nc/Network](https://api.ucloudbiz.olleh.com/gd1/nsm/v1/network)
  * The ssh keyname is the one which was created and downloaded earlier
* For the try-crds/infrastructure_v1beta1_machinedeployment.yaml, we have to modify the clusters, replicas failure domain matching the availability zone in KT cloud.
  * if the template if for the control-plane, put spec.type as control-plane otherwise worker
  * The failure domain name matches the zoneid of the associated account
* Leave the try-crds/infrastructure_v1beta1_cluster.yaml as is except the ${CLUSTER_NAME}
* Finally, change the try-crds/infrastructure_v1beta1_ktcluster.yaml by modifying on spec.controlPlaneExternalNetworkEnable putting boolean true or false. Not forgetting changing on ${CLUSTER_NAME}

### Others

* APIs can be tested using the postman exported JSON Collection in KTCloud.postman_collection.json, import it in POSTMAN and get startedi
* f you get the error below in the logs, don’t worry the reconciler is just checking the CP, it will reconcile to check if the API server is ready on port 8000

```
	"error": "Get \"http://${public_ip}:8000\": dial tcp 211.57.84.211:8000: connect: connection refused"}
```


```
kubectl get secret kt-cluster1-kubeconfig -o jsonpath='{.data.value}' | base64 -d>ktcluster.kubeconfig
# edit the kubeconfig and comment out the certificate-authority-data and add the "insecure-skip-tls-verify: true" key value
nano nano ktcluster.kubeconfig
#add this after commenting out the above
insecure-skip-tls-verify: true
# change the server to the assigned public IP
server: https://172.25.0.8:6443
# changed to
server: https://211.57.84.213:6443
# and we have the following
root@mgmt-control:~# nano ktcluster.kubeconfig
root@mgmt-control:~# kubectl get nodes --kubeconfig ktcluster.kubeconfig
NAME                                        STATUS     ROLES           AGE     VERSION
kt-cluster1-control-plane-62bodppn5-yammk   NotReady   control-plane   6m5s    v1.29.15
kt-cluster1-md-0-nspmm7rznc1g51w            NotReady   <none>          4m30s   v1.29.15

```

##### KubeConfig setting up
* Since we don't have direct communication setup to the cloud or cluster, we need to temporarily disable TLS verification (testing purposes only)
* Just delete the certificate-authority-data line in your kubeconfig and keep insecure-skip-tls-verify: true while testing:
* We want to access the cluster through the public assigned IP but internally, the certificates only allows private/local IP address
  * This means your kubeconfig file is using both:
    * a certificate authority file (certificate-authority, certificate-authority-data)
    * and insecure-skip-tls-verify: true
  * Kubernetes does not allow both at the same time — it's one or the other.
* Solution:
  * If you want to use insecure-skip-tls-verify: true (i.e., bypass TLS check):
  ```
  clusters:
  - cluster:
    server: https://211.57.84.213:6443
    insecure-skip-tls-verify: true
    # REMOVE this line if it exists:
    # certificate-authority-data: ...
    # OR
    # certificate-authority: ...
  ```
  * If you want to keep TLS verification:
    1. Remove insecure-skip-tls-verify: true
    2. Make sure the certificate-authority-data or certificate-authority is valid
    3. And the server URL matches one of the cert SANs (e.g., 172.25.0.118)
    But in your case, you said the public IP is not in the cert SAN — so this will fail unless you regenerate the certs (as explained earlier).

* Making docker image for the controller
  ```
    docker login
    make docker-build IMG=vitu1/kt-cloud-operator:v0.1
    make docker-push IMG=vitu1/kt-cloud-operator:v0.1
  ```