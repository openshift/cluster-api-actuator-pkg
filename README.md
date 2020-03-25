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

### Run the autoscaler e2e tests

Before running the autoscaler e2e tests, ensure that you have exported the
`KUBECONFIG` environment variable, or that you have a valid config in
`$HOME/.kube/config`. By default, these tests will run in the current namespace.

These tests use the [Ginkgo](https://onsi.github.io/ginkgo/) framework, you
can pass command line options directly through the `ci-integration.sh` script.
For example, use `-focus` to restrict the tests you are running.

```console
$ ./hack/ci-integration.sh -focus "Autoscaler should" -v -dryRun
[2] Running Suite: Machine Suite
[2] ============================
[2] Random Seed: 1585144111
[2] Parallel test node 2/4.
[2]
[4] Running Suite: Machine Suite
[4] ============================
[4] Random Seed: 1585144111
[4] Parallel test node 4/4.
[4]
[2] SSSSS
[2] ------------------------------
[2] [Feature:Machines] Autoscaler should
[2]   scale up and down
[2]   /home/mike/workspace/go/src/github.com/openshift/cluster-api-actuator-pkg/pkg/autoscaler/autoscaler.go:209
[4] SSSS
[4] Ran 0 of 4 Specs in 0.002 seconds
[4] SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 4 Skipped
[4] PASS
[2] •SSSS
[2] Ran 1 of 10 Specs in 0.007 seconds
[2] SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 9 Skipped
[2] PASS
[1] Running Suite: Machine Suite
[1] ============================
[1] Random Seed: 1585144111
[1] Parallel test node 1/4.
[1]
[1]
[1] Ran 0 of 0 Specs in 0.001 seconds
[1] SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 0 Skipped
[1] PASS
[3] Running Suite: Machine Suite
[3] ============================
[3] Random Seed: 1585144111
[3] Parallel test node 3/4.
[3]
[3]
[3] Ran 0 of 0 Specs in 0.000 seconds
[3] SUCCESS! -- 0 Passed | 0 Failed | 0 Pending | 0 Skipped
[3] PASS

Ginkgo ran 1 suite in 4.609743815s
Test Suite Passed
```
