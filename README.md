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
  * For flavor, we have to use an API directly because it is not provided in the cloud console follow the guide on this [page](https://cloud.kt.com/docs/open-api-guide/d/computing/virtual-machine), on this API endpoint: https://api.ucloudbiz.olleh.com/d1/server/flavors/detail
  * The blockingDeviceMapping data, can be taken from the console in servers/Image, click on the preferred image then information and get its ID. The other variables can be customized based on requirements
  * The networkTier.id is taken from this GET API: [https://api.ucloudbiz.olleh.com/gd1/nc/Network](https://api.ucloudbiz.olleh.com/gd1/nc/Network)
  * The ssh keyname is the one which was created and downloaded earlier
* For the try-crds/infrastructure_v1beta1_machinedeployment.yaml, we have to modify the clusters, replicas failure domain matching the availability zone in KT cloud.
  * if the template if for the control-plane, put spec.type as control-plane otherwise worker
  * The failure domain name matches the zoneid of the associated account
* Leave the try-crds/infrastructure_v1beta1_cluster.yaml as is except the ${CLUSTER_NAME}
* Finally, change the try-crds/infrastructure_v1beta1_ktcluster.yaml by modifying on spec.controlPlaneExternalNetworkEnable putting boolean true or false. Not forgetting changing on ${CLUSTER_NAME}

### Others

* if you get the error below in the logs, don’t worry the reconciler is just checking the CP, it will reconcile to check if the API server is ready on port 8000

  ```
  "error": "Get \"http://${public_ip}:8000\": dial tcp 211.57.84.211:8000: connect: connection refused"}
  ```
