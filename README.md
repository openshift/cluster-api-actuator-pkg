# cluster-api-actuator-pkg

Shared packages for Cluster API actuators.

## Runing the cluster autoscaler e2e tests locally using minikube

You can run the autoscaler e2e tests locally using minikube with the
following setup:

### Start minikube

```console
$ minikube start --vm-driver kvm2 --kubernetes-version v1.13.4 --v 5 --memory=25165
* minikube v1.2.0 on linux (amd64)
* Creating kvm2 VM (CPUs=2, Memory=25165MB, Disk=20000MB) ...
Running pre-create checks...
Creating machine...
(minikube) Creating machine...
(minikube) Creating network...
(minikube) Setting up minikube home directory...
(minikube) Building disk image...
(minikube) Downloading /home/aim/.minikube/cache/boot2docker.iso from file:///home/aim/.minikube/cache/iso/minikube-v1.2.0.iso...
(minikube) Creating domain...
(minikube) Creating network...
(minikube) Ensuring networks are active...
(minikube) Ensuring network default is active
(minikube) Ensuring network minikube-net is active
(minikube) Getting domain xml...
(minikube) Creating domain...
(minikube) Waiting to get IP...
(minikube) Found IP for machine: 192.168.39.74
(minikube) Waiting for SSH to be available...
Waiting for machine to be running, this may take a few minutes...
Detecting operating system of created instance...
Waiting for SSH to be available...
Detecting the provisioner...
Provisioning with buildroot...
Setting Docker configuration on the remote daemon...
Checking connection to Docker...
Docker is up and running!
* Configuring environment for Kubernetes v1.13.4 on Docker 18.09.6
* Pulling images ...
* Launching Kubernetes ... 
* Verifying: apiserver proxy etcd scheduler controller dns
* Done! kubectl is now configured to use "minikube"

$ kubectl get nodes
NAME       STATUS   ROLES    AGE   VERSION
minikube   Ready    master   11m   v1.13.4
```

### Deploy kubemark stack

```console
$ (cd ~/go-projects/cluster-api-provider-kubemark/src/github.com/openshift/cluster-api-provider-kubemark/; cd config && kustomize build | kubectl apply --validate=false -f -)
namespace/kubemark-actuator created
customresourcedefinition.apiextensions.k8s.io/clusters.cluster.k8s.io created
customresourcedefinition.apiextensions.k8s.io/machinedeployments.machine.openshift.io created
customresourcedefinition.apiextensions.k8s.io/machines.machine.openshift.io created
customresourcedefinition.apiextensions.k8s.io/machinesets.machine.openshift.io created
serviceaccount/kubemark created
clusterrole.rbac.authorization.k8s.io/cluster-api-manager-role created
clusterrole.rbac.authorization.k8s.io/kubemark-actuator-role created
clusterrolebinding.rbac.authorization.k8s.io/cluster-api-manager-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/kubemark-actuator-rolebinding created
configmap/deleteunreadynodes created
deployment.apps/clusterapi-manager-controllers created
deployment.apps/machineapi-kubemark-controllers created

$ kubectl get deployments --all-namespaces
NAMESPACE     NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
default       clusterapi-manager-controllers    1/1     1            1           2m32s
kube-system   coredns                           2/2     2            2           14m
kube-system   machineapi-kubemark-controllers   1/1     1            1           2m32s

$ kubectl get pods --all-namespaces
NAMESPACE     NAME                                              READY   STATUS    RESTARTS   AGE
default       clusterapi-manager-controllers-6c46d758f-5v2vf    3/3     Running   0          2m54s
kube-system   coredns-86c58d9df4-jhhh6                          1/1     Running   0          14m
kube-system   coredns-86c58d9df4-tnpxs                          1/1     Running   0          14m
kube-system   etcd-minikube                                     1/1     Running   0          13m
kube-system   kube-addon-manager-minikube                       1/1     Running   0          13m
kube-system   kube-apiserver-minikube                           1/1     Running   0          13m
kube-system   kube-controller-manager-minikube                  1/1     Running   2          13m
kube-system   kube-proxy-swrz7                                  1/1     Running   0          14m
kube-system   kube-scheduler-minikube                           1/1     Running   2          13m
kube-system   machineapi-kubemark-controllers-d4c489595-lfbfc   1/1     Running   0          2m54s
kube-system   storage-provisioner                               1/1     Running   0          14m
```

### 

### Deploy CRDs for cluster-autoscaler-operator

```console
$ (cd ~/go-projects/cluster-autoscaler-operator/src/github.com/openshift/cluster-autoscaler-operator/; kustomize build | kubectl apply --validate=false -f -)
customresourcedefinition.apiextensions.k8s.io/clusterautoscalers.autoscaling.openshift.io created
customresourcedefinition.apiextensions.k8s.io/machineautoscalers.autoscaling.openshift.io created
serviceaccount/cluster-autoscaler-operator created
serviceaccount/cluster-autoscaler created
role.rbac.authorization.k8s.io/cluster-autoscaler created
role.rbac.authorization.k8s.io/prometheus-k8s-cluster-autoscaler-operator created
role.rbac.authorization.k8s.io/cluster-autoscaler-operator created
clusterrole.rbac.authorization.k8s.io/cluster-autoscaler-operator created
clusterrole.rbac.authorization.k8s.io/cluster-autoscaler created
rolebinding.rbac.authorization.k8s.io/cluster-autoscaler created
rolebinding.rbac.authorization.k8s.io/prometheus-k8s-cluster-autoscaler-operator created
rolebinding.rbac.authorization.k8s.io/cluster-autoscaler-operator created
clusterrolebinding.rbac.authorization.k8s.io/cluster-autoscaler-operator created
clusterrolebinding.rbac.authorization.k8s.io/cluster-autoscaler created
configmap/cluster-autoscaler-operator-ca created
secret/cluster-autoscaler-operator-cert created
service/cluster-autoscaler-operator created
deployment.apps/cluster-autoscaler-operator created

$ kubectl get deployments --all-namespaces
NAMESPACE     NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
default       clusterapi-manager-controllers    1/1     1            1           5m15s
kube-system   cluster-autoscaler-operator       0/1     1            0           39s
kube-system   coredns                           2/2     2            2           16m
kube-system   machineapi-kubemark-controllers   1/1     1            1           5m15s

$ kubectl get pods --all-namespaces
NAMESPACE     NAME                                              READY   STATUS             RESTARTS   AGE
default       clusterapi-manager-controllers-6c46d758f-5v2vf    3/3     Running            0          5m40s
kube-system   cluster-autoscaler-operator-7f6d66f5c6-4glb5      0/1     CrashLoopBackOff   3          64s
kube-system   coredns-86c58d9df4-jhhh6                          1/1     Running            0          17m
kube-system   coredns-86c58d9df4-tnpxs                          1/1     Running            0          17m
kube-system   etcd-minikube                                     1/1     Running            0          16m
kube-system   kube-addon-manager-minikube                       1/1     Running            0          16m
kube-system   kube-apiserver-minikube                           1/1     Running            0          16m
kube-system   kube-controller-manager-minikube                  1/1     Running            2          16m
kube-system   kube-proxy-swrz7                                  1/1     Running            0          17m
kube-system   kube-scheduler-minikube                           1/1     Running            2          16m
kube-system   machineapi-kubemark-controllers-d4c489595-lfbfc   1/1     Running            0          5m40s
kube-system   storage-provisioner                               1/1     Running            0          17m
```

At this point I set the cluster-autoscaler-operator image to something
explicit:

```console
$ kubectl set image deployment/cluster-autoscaler-operator cluster-autoscaler-operator=frobware/origin-cluster-autoscaler-operator:latest -n kube-system
deployment.extensions/cluster-autoscaler-operator image updated

$ kubectl get pods --all-namespaces
NAMESPACE     NAME                                              READY   STATUS    RESTARTS   AGE
default       clusterapi-manager-controllers-6c46d758f-5v2vf    3/3     Running   0          8m45s
kube-system   cluster-autoscaler-operator-7787659984-2brpt      1/1     Running   0          100s
kube-system   coredns-86c58d9df4-jhhh6                          1/1     Running   0          20m
kube-system   coredns-86c58d9df4-tnpxs                          1/1     Running   0          20m
kube-system   etcd-minikube                                     1/1     Running   0          19m
kube-system   kube-addon-manager-minikube                       1/1     Running   0          19m
kube-system   kube-apiserver-minikube                           1/1     Running   0          19m
kube-system   kube-controller-manager-minikube                  1/1     Running   2          19m
kube-system   kube-proxy-swrz7                                  1/1     Running   0          20m
kube-system   kube-scheduler-minikube                           1/1     Running   2          19m
kube-system   machineapi-kubemark-controllers-d4c489595-lfbfc   1/1     Running   0          8m45s
kube-system   storage-provisioner                               1/1     Running   0          20m
```

### Create some MachineSet's for autoscaling

```console
$ kubectl get machinesets --all-namespaces
No resources found.

$ kubectl create -f ~/go-projects/cluster-api-provider-kubemark/src/github.com/openshift/cluster-api-provider-kubemark/examples/machine-set-2a.yaml
$ kubectl create -f ~/go-projects/cluster-api-provider-kubemark/src/github.com/openshift/cluster-api-provider-kubemark/examples/machine-set-2b.yaml
$ kubectl create -f ~/go-projects/cluster-api-provider-kubemark/src/github.com/openshift/cluster-api-provider-kubemark/examples/machine-set-2c.yaml
```

The MachineSet's by default have a replica count of 0 but we can set
those to have 1 replica each which is what the e2e tests are
expecting:

```console
function reset_machinesets {
  oc patch -n kube-system machineset/kubemark-scale-group-2a -p '{"spec":{"replicas":1}}' --type=merge;
  oc patch -n kube-system machineset/kubemark-scale-group-2b -p '{"spec":{"replicas":1}}' --type=merge;
  oc patch -n kube-system machineset/kubemark-scale-group-2c -p '{"spec":{"replicas":1}}' --type=merge;
}

$ reset_machinesets

$ kubectl get machinesets --all-namespaces
NAMESPACE     NAME                      DESIRED   CURRENT   READY   AVAILABLE   AGE
kube-system   kubemark-scale-group-2a   1         1                             23s
kube-system   kubemark-scale-group-2b   1         1                             16s
kube-system   kubemark-scale-group-2c   1         1                             13s
```

### Build the e2e tests

```console
$ make build-e2e 
go test -c -o bin/e2e github.com/openshift/cluster-api-actuator-pkg/pkg/e2e
```

### Run the autoscaler e2e tests

```console
$ NAMESPACE=kube-system ./hack/ci-integration.sh -ginkgo.focus "Autoscaler should" -ginkgo.v -ginkgo.dryRun
=== RUN   TestE2E
Running Suite: Machine Suite
============================
Random Seed: 1562320813
Will run 1 of 15 specs

[Feature:Machines][Serial] Autoscaler should 
  scale up and down
  /home/aim/go-projects/cluster-api-actuator-pkg/src/github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/autoscaler/autoscaler.go:229
•SSSSSSSSSSSSSS
Ran 1 of 15 Specs in 0.000 seconds
SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 14 Skipped
--- PASS: TestE2E (0.00s)
PASS
ok      github.com/openshift/cluster-api-actuator-pkg/pkg/e2e   0.037s
```

Adjust `-ginkgo.focus` as appropriate.

### Lather, rinse, repeat...

The tests (during development) will / can fail and you can use the
following function to clean up should that occur before running them
again:

```console
function reset_testenv {
  kubectl delete machineautoscalers --all;
  kubectl delete clusterautoscalers --all;
  kubectl delete jobs -n default --all;
  oc patch -n kube-system machineset/kubemark-scale-group-2a -p '{"spec":{"replicas":1}}' --type=merge;
  oc patch -n kube-system machineset/kubemark-scale-group-2b -p '{"spec":{"replicas":1}}' --type=merge;
  oc patch -n kube-system machineset/kubemark-scale-group-2c -p '{"spec":{"replicas":1}}' --type=merge;
}

$ reset_testenv
$ NAMESPACE=kube-system ./hack/ci-integration.sh -ginkgo.focus "Autoscaler should" -ginkgo.v
```
