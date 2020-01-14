# cluster-api-actuator-pkg

Shared packages for Cluster API actuators.

## Running the cluster autoscaler operator e2e tests against an OpenShift cluster

This test suite is designed to run against a full OpenShift 4 cluster.
The test suite is agnostic of the hosting environment and the choice of cloud provider is up to the reader.

These instructions are written for the cluster autoscaler operator though should work for any project using the cluster-api-actuator-pkg.

### Create a cluster

The easiest way to get a cluster to test against is to use an installer that supports Installer-Provisioned Infrastructure (IPI).

Instructions for creating an IPI cluster are available for the following cloud providers:
- [AWS](https://cloud.redhat.com/openshift/install/aws/installer-provisioned)
- [Azure](https://cloud.redhat.com/openshift/install/azure/installer-provisioned)
- [GCP](https://cloud.redhat.com/openshift/install/gcp/installer-provisioned)
- [Red Hat OpenStack](https://cloud.redhat.com/openshift/install/openstack/installer-provisioned)

### Deploy the code to test

Before making any changes to the cluster components you wish to test, you must disable the Cluster-Version Operator (CVO).
If you do not disable the CVO, when you try to deploy your test code, the CVO will revert the component back to the original version.

To disable the CVO, scale its deployment to 0 replicas:

```console
$ oc scale --replicas 0 -n openshift-cluster-version deployments/cluster-version-operator
deployment.apps/cluster-version-operator scaled
```

Now deploy the code either directly into the cluster, or by running it locally.

### Deploy the code to test within the cluster

To deploy your test code within the cluster, you must first build and push a container image to a repository.
Once pushed, override the image within the deployment to deploy your code for testing:

```console
$ oc set image -n machine-api-operator deployment/cluster-autoscaler-operator cluster-autoscaler-operator=<YOUR CONTAINER IMAGE>
deployment.apps/cluster-autoscaler-operator image updated
```

### Deploy the code to test locally

To deploy your test code locally, you must first disable the existing operator running within the OpenShift cluster:

```console
$ oc scale --replicas 0 -n openshift-machine-api deployments/cluster-autoscaler-operator
deployment.apps/cluster-autoscaler-operator scaled
```

Once the operator has been disabled, build your code to test locally and run it on your machine,
pointing it to the cluster by passing the appropriate `--kubeconfig` flag:

```console
$ make build
  ...
$ ./bin/cluster-autoscaler-operator --kubeconfig=<PATH/TO/YOUR/CLUSTERS/KUBECONFIG> ...
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
