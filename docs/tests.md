<!-- TOC -->
* [Overview](#overview)
* [Tests](#tests)
  * [Operators](#operators)
    * [Machine API](#machine-api)
      * [Machine API operator deployment should be available](#machine-api-operator-deployment-should-be-available)
      * [Machine API operator deployment should reconcile controllers deployment](#machine-api-operator-deployment-should-reconcile-controllers-deployment)
      * [Machine API operator deployment should maintains deployment spec](#machine-api-operator-deployment-should-maintains-deployment-spec)
      * [Machine API operator deployment should reconcile mutating webhook configuration](#machine-api-operator-deployment-should-reconcile-mutating-webhook-configuration)
      * [Machine API operator deployment should reconcile validating webhook configuration](#machine-api-operator-deployment-should-reconcile-validating-webhook-configuration)
      * [Machine API operator deployment should recover after validating webhook configuration deletion](#machine-api-operator-deployment-should-recover-after-validating-webhook-configuration-deletion)
      * [Machine API operator deployment should recover after mutating webhook configuration deletion](#machine-api-operator-deployment-should-recover-after-mutating-webhook-configuration-deletion)
      * [Machine API operator deployment should maintains spec after mutating webhook configuration change and preserve caBundle](#machine-api-operator-deployment-should-maintains-spec-after-mutating-webhook-configuration-change-and-preserve-cabundle)
      * [Machine API operator deployment should maintains spec after validating webhook configuration change and preserve caBundle](#machine-api-operator-deployment-should-maintains-spec-after-validating-webhook-configuration-change-and-preserve-cabundle)
      * [Machine API cluster operator status should be available](#machine-api-cluster-operator-status-should-be-available)
      * [When cluster-wide proxy is configured, Machine API cluster operator should  create machines when configured behind a proxy](#when-cluster-wide-proxy-is-configured-machine-api-cluster-operator-should--create-machines-when-configured-behind-a-proxy)
    * [Cluster Machine Approver](#cluster-machine-approver)
      * [Cluster Machine Approver deployment should be available](#cluster-machine-approver-deployment-should-be-available)
      * [Cluster Machine Approver Cluster Operator Status should be available](#cluster-machine-approver-cluster-operator-status-should-be-available)
    * [Cluster Autoscaler](#cluster-autoscaler)
      * [Cluster autoscaler operator should reject invalid ClusterAutoscaler resources early via webhook](#cluster-autoscaler-operator-should-reject-invalid-clusterautoscaler-resources-early-via-webhook)
      * [Cluster autoscaler operator should reject invalid MachineAutoscaler resources early via webhook](#cluster-autoscaler-operator-should-reject-invalid-machineautoscaler-resources-early-via-webhook)
      * [Cluster autoscaler operator deployment should be available](#cluster-autoscaler-operator-deployment-should-be-available)
      * [Cluster autoscaler cluster operator status should be available](#cluster-autoscaler-cluster-operator-status-should-be-available)
  * [Machines](#machines)
    * [AWS](#aws)
      * [MetadataServiceOptions should not allow to create machineset with incorrect metadataServiceOptions.authentication](#metadataserviceoptions-should-not-allow-to-create-machineset-with-incorrect-metadataserviceoptionsauthentication)
      * [MetadataServiceOptions should enforce auth on metadata service if metadataServiceOptions.authentication set to Required](#metadataserviceoptions-should-enforce-auth-on-metadata-service-if-metadataserviceoptionsauthentication-set-to-required)
      * [MetadataServiceOptions should allow unauthorized requests to metadata service if metadataServiceOptions.authentication is Optional](#metadataserviceoptions-should-allow-unauthorized-requests-to-metadata-service-if-metadataserviceoptionsauthentication-is-optional)
    * [Managed Cluster (common machinesets tests)](#managed-cluster--common-machinesets-tests-)
      * [Managed cluster should drain node before removing machine resource](#managed-cluster-should-drain-node-before-removing-machine-resource)
      * [Managed cluster should recover from deleted worker machines](#managed-cluster-should-recover-from-deleted-worker-machines)
      * [Managed cluster should grow and decrease when scaling different machineSets simultaneously](#managed-cluster-should-grow-and-decrease-when-scaling-different-machinesets-simultaneously)
      * [Managed cluster should reject invalid machinesets](#managed-cluster-should-reject-invalid-machinesets)
      * [Managed cluster should have ability to additively reconcile taints from machine to nodes](#managed-cluster-should-have-ability-to-additively-reconcile-taints-from-machine-to-nodes)
    * [Webhooks](#webhooks)
      * [Webhooks should be able to create a machine from a minimal providerSpec](#webhooks-should-be-able-to-create-a-machine-from-a-minimal-providerspec)
      * [Webhooks should be able to create machines from a machineset with a minimal providerSpec](#webhooks-should-be-able-to-create-machines-from-a-machineset-with-a-minimal-providerspec)
      * [Webhooks should return an error when removing required fields from the Machine providerSpec](#webhooks-should-return-an-error-when-removing-required-fields-from-the-machine-providerspec)
      * [Webhooks should return an error when removing required fields from the MachineSet providerSpec](#webhooks-should-return-an-error-when-removing-required-fields-from-the-machineset-providerspec)
    * [Lifecycle Hooks](#lifecycle-hooks)
      * [Lifecycle Hooks should pause lifecycle actions when present](#lifecycle-hooks-should-pause-lifecycle-actions-when-present)
  * [MachineHealthCheck](#machinehealthcheck)
      * [MachineHealthCheck should remediate unhealthy nodes](#machinehealthcheck-should-remediate-unhealthy-nodes)
      * [MachineHealthCheck should not remediate larger number of unhealthy machines then maxUnhealthy](#machinehealthcheck-should-not-remediate-larger-number-of-unhealthy-machines-then-maxunhealthy)
    * [Autoscaler](#autoscaler)
      * [Autoscaler should use a ClusterAutoscaler that has 100 maximum total nodes count It scales from/to zero](#autoscaler-should-use-a-clusterautoscaler-that-has-100-maximum-total-nodes-count-it-scales-fromto-zero)
      * [Autoscaler should use a ClusterAutoscaler that has 100 maximum total nodes count cleanup deletion information after scale down [Slow]](#autoscaler-should-use-a-clusterautoscaler-that-has-100-maximum-total-nodes-count-cleanup-deletion-information-after-scale-down-slow)
      * [Autoscaler should use a ClusterAutoscaler that has 12 maximum total nodes count and balance similar nodes enabled scales up and down while respecting MaxNodesTotal [Slow][Serial]](#autoscaler-should-use-a-clusterautoscaler-that-has-12-maximum-total-nodes-count-and-balance-similar-nodes-enabled-scales-up-and-down-while-respecting-maxnodestotal-slowserial)
      * [Autoscaler should use a ClusterAutoscaler that has 12 maximum total nodes count and balance similar nodes enabled places nodes evenly across node groups [Slow]](#autoscaler-should-use-a-clusterautoscaler-that-has-12-maximum-total-nodes-count-and-balance-similar-nodes-enabled-places-nodes-evenly-across-node-groups-slow)
<!-- TOC -->

# Overview

This document describes tests presented in this repository as of the start of December 2022.
Some tests are grouped if they are similar. Each heading represents a single test and reflects its full
name as it's programmed and shown in the JUnit report.

# Tests

## Operators

### Machine API

#### Machine API operator deployment should be available
* Parallel: No (sits in disruptive suite)
* ExecTime: ~5s on AWS
* Recommendation: 
  * Might be extracted and run in parallel with the other part of the suite.
  * Or, moved into `BeforeEach` block and executed before another MachineAPI Operator tests, instead of having this separately

This test checks status of the `openshift-machine-api/machine-api-operator` deployment.
Passes if deployment reports `AvailableReplicas` > 0. If no deployment available, will fail in 15m due to timeout in
the [framework check function](https://github.com/openshift/cluster-api-actuator-pkg/blob/06ef5d16ea5ee64a008c0711af92bad671db0372/pkg/framework/deployment.go#L63)

#### Machine API operator deployment should reconcile controllers deployment
* Parallel: No
* ExecTime: ~10s on AWS (but might affect another tests due to controller might not be started due to leaderelection waiting)
* Recommendation:
  * Extend this test for wait and check that deleted controller was fully running, since seems not all controller has `releaseOnCancel` parameter set up. 

This test query `openshift-machine-api/machine-api-controllers` deployment, saves manifest and deletes deployment.
After, tests wait for the deployment to be restored and be equal to the initial manifest.
This test checks operator behaviour to manage and maintain MAPI components deployment state.

#### Machine API operator deployment should maintains deployment spec
* Parallel: No
* ExecTime: ~5s on AWS (but might affect another tests due to controller might not be started due to leaderelection waiting)
* Recommendation:
  * Extend this test for wait and check that deleted controller was fully running
  * OR: Extract this into separate serial suite with the previous test, run one after another and check that controllers runs successfully after tests was passed.

This test query `openshift-machine-api/machine-api-controllers` deployment, saves manifest and scales deployment to zero replicas.
After, tests wait for the deployment to be restored and available (available replicas > 0).
This test checks operator behaviour to manage and maintain MAPI components deployment state.

#### Machine API operator deployment should reconcile mutating webhook configuration
#### Machine API operator deployment should reconcile validating webhook configuration
* Parallel: No (each)
* ExecTime: ~5s on AWS (each)
* Recommendation:
  - Extract this test, make it parallel
  - break dependency to the vendored MAPO if it's possible
  - this two tests might be generalized in a way and merged into one
  - this tests might be executed as pre-step for [should recover after mutating webhook configuration deletion](#machine-api-operator-deployment-should-recover-after-mutating-webhook-configuration-deletion) [should maintains spec after validating webhook configuration change and preserve caBundle](#machine-api-operator-deployment-should-maintains-spec-after-validating-webhook-configuration-change-and-preserve-cabundle)

Two pretty much the same tests. Checks that webhooks are configured in a cluster. Tests using webhook definition (golang structure) imported from the MAPO
repository, which is not good and might cause issues if webhook settings will need to be changed (these tests will fail, till MAPO revendoring in cluster-actuator-pkg-repo).

This test checks operator behaviour to manage and maintain MAPI webhooks state.

#### Machine API operator deployment should recover after validating webhook configuration deletion
#### Machine API operator deployment should recover after mutating webhook configuration deletion
* Parallel: No (each)
* ExecTime: ~5s on AWS (each)
* Recommendation:
    - break dependency to the vendored MAPO if it's possible
    - this two tests might be generalized in a way and merged into one, the only difference is a resource type

This test checks that webhook configurations are presented in cluster then delete it. Waits for configurations to be restored and 
being equal to what is encoded within the vendored MAPO repo.

This test checks operator behaviour to manage and maintain MAPI webhooks state.

#### Machine API operator deployment should maintains spec after mutating webhook configuration change and preserve caBundle
#### Machine API operator deployment should maintains spec after validating webhook configuration change and preserve caBundle
* Parallel: No (each)
* ExecTime: ~5s on AWS (each)
* Recommendation:
    - break dependency to the vendored MAPO if it's possible
    - this two tests might be generalized in a way and merged into one, the only difference is a resource type

This test checks that webhook configurations are presented in cluster then changes it. Waits for configurations to be restored and
being equal to what is encoded within the vendored MAPO repo.

This test checks operator behaviour to manage and maintain MAPI webhooks state.

#### Machine API cluster operator status should be available
* Parallel: Yes
* ExecTime: ~2s on AWS
* Recommendation:
  * Might be executed in `BeforeEach` block for other Machine API related tests. This separate case might be removed
Checks cluster operator status to be available, not progressing and not degraded.

#### When cluster-wide proxy is configured, Machine API cluster operator should  create machines when configured behind a proxy
* Parallel: No
* ExecTime: ~30m on AWS
* Recommendation:
  - ~make this test periodic (done already)~
  - extend this test - add check for the CCCMO also

This test consists of several stages and exercises MAPO ability to pick up and properly handle cluster-wide proxy settings as well as
machine-controllers ability to operate behind a proxy server:

1) Deploying HTTP proxy server as a DeamonSet
2) Waits for machine-api controllers will be redeployed with updated proxy settings
3) Creates machineset, waits when machine will be created and hit the Running phase
4) Removes proxy
5) Waits for machine-api controllers will be redeployed without proxy settings

Since configuring proxy causes deployment updates for KCM, MAO and a bunch of other system critical components, this test takes quite long time.
Also, this test is quite disruptive and should be running in Serial mode, due to it touches cluster-wide settings which affect not only MAO parts.

### Cluster Machine Approver

Recommendation: This suite needs to be extended with some sort of disruptive suite (change/delete CMA deployment, ensure that it's preserved).
Another variant is to remove this test at all since a cluster will not finish installation if the operator will be in the degraded state.

#### Cluster Machine Approver deployment should be available
* Parallel: Yes
* ExecTime: ~2s on AWS

This test checks status of the `openshift-cluster-machine-approver/machine-approver` deployment.
Passes if deployment reports `AvailableReplicas` > 0. If no deployment available, will fail in 15m due to timeout in
the [framework check function](https://github.com/openshift/cluster-api-actuator-pkg/blob/06ef5d16ea5ee64a008c0711af92bad671db0372/pkg/framework/deployment.go#L63)

#### Cluster Machine Approver Cluster Operator Status should be available
* Parallel: Yes
* ExecTime: ~2s on AWS
* Recommendation:
  * Might be executed in `BeforeEach` block for other CMAO related tests. This separate case might be removed

Checks cluster operator status to be available, not progressing and not degraded.

### Cluster Autoscaler

Recommendation: This suite need to be extended with some sort of disruptive suite (change/delete autoscaler deployment, ensure that it's preserved by the operator)

#### Cluster autoscaler operator should reject invalid ClusterAutoscaler resources early via webhook
* Parallel: Yes
* ExecTime: ~5s on AWS
* Recommendation:
  - Remove from this suite, move this test to the cluster autoscaler repo
  - Or review and extend, but better remove

This test just checks that creation of ClusterAutoscaler resource with name other than 'default' is rejected

#### Cluster autoscaler operator should reject invalid MachineAutoscaler resources early via webhook
* Parallel: Yes
* ExecTime: ~5s on AWS
* Recommendation:
  - Remove, and implement this test to the cluster autoscaler repo
  - Alternative - move this test closer to another webhooks test.

This test just checks that creation of MachineAutoscaler resource with MinReplicas > MaxReplicas is rejected

#### Cluster autoscaler operator deployment should be available
* Parallel: Yes
* ExecTime: ~5s on AWS

This test checks status of the `openshift-machine-api/cluster-autoscaler-operator` deployment.
Passes if deployment reports `AvailableReplicas` > 0. If no deployment available, will fail in 15m due to timeout in
the [framework check function](https://github.com/openshift/cluster-api-actuator-pkg/blob/06ef5d16ea5ee64a008c0711af92bad671db0372/pkg/framework/deployment.go#L63)

#### Cluster autoscaler cluster operator status should be available
* Parallel: Yes
* ExecTime: ~2s on AWS
* Recommendation:
  * Might be executed in `BeforeEach` block for other CAO related tests. This separate case might be removed

Checks cluster operator status to be available, not progressing and not degraded.

## Machines

### AWS
#### MetadataServiceOptions should not allow to create machineset with incorrect metadataServiceOptions.authentication
* Parallel: Yes
* ExecTime: ~5s on AWS
* Notes: AWS only
* Recommendation:
  * Might be removed. Such functionality might be covered with webhook tests in MAPO

This test excersising validation webhook behaviour for `metadataServiceOptions.authentication` field. Checks that machineset can not be created
if `metadataServiceOptions.authentication` is not equal `Required` or `Optional`

#### MetadataServiceOptions should enforce auth on metadata service if metadataServiceOptions.authentication set to Required
#### MetadataServiceOptions should allow unauthorized requests to metadata service if metadataServiceOptions.authentication is Optional
* Parallel: Yes (each)
* ExecTime: 4-5m on AWS (each)
* Notes: AWS only

Each of these tests creates a machineset with a respective value (`Required` or `Optional`) in `metadataServiceOptions.authentication`.
Then, after a machine from the created machineset hits "Running" phase,  pod with simple curl command starts on the respective node then.
Test waits for the curl output and checks if it contains expected response (401 for required case, 200 for optional).

### Managed Cluster (common machinesets tests)
Every single test in this section creates a manineset with 2 machines, because of that, this tests
takes a significant time (>5 m).

All of the test listed below shares common `BeforeEach` and `AfterEach` blocks.
In `BeforeEach` machineset creation and wait happen (this machineset has 2 desired replicas configured, so 2 machines will be created for each test here).
In `AfterEach` created machineset is being deleted.

#### Managed cluster should drain node before removing machine resource
* Parallel: Yes
* ExecTime: >12m on AWS
* Recommendations:
  * Update resource creation helper functions, and add parameters to specify workload names, replicas count, selectors, etc. Now it's almost impossible to say what is going on without looking into these functions.
  * Add more meaningful comments inside the test code, now it's not easy to figure out what is going on there.
  * Fix logging here, it is quite noisy, and do flood reports.
  * Reduce amount of pods created
  * Check where pods was actually scheduled

This test checks nodes are properly draining during machine deletion and this process also respects PDBs (pod disruption budgets).
During this test - machineset with two desired replicas and a replication controller with 20 desired replicas are creating, then PDB creating for this workload.
Machines from the machinesets marking with labels to be sure that pods created by the replication controller will be scheduled there.
After the replication controller is ready, one of the machines from the machineset marking for deletion. Along with the deletion process test checking that pods amount on the node
is going down to 1 and RC has at most one non-ready replica.

Test considered failed in case if more than 1 replica within RC detected as not ready during machineset deletion. That would mean that PDB constraints were violated.

#### Managed cluster should recover from deleted worker machines
* Parallel: Yes
* ExecTime: >11m on AWS
* Recommendations:
  * Reduce created machines number to 1 for this test, currently there are 2
  * This test might be possibly removed since it exercises a machineset ability to recreate deleted machines and implemented with envtest

This test takes machines from the machineset which was created in `BeforeEach` block and marks them for deletion,
then waits for machines to be deleted and recreated by the machineset. Takes quite a while, because 2 machines should be deleted and recreated then.

#### Managed cluster should grow and decrease when scaling different machineSets simultaneously
* Parallel: Yes
* ExecTime: >10m on AWS
* Recommendations:
  * Reduce created machines number to 1 for this test, currently there are 2
  * Remove this test, similiar one included into origin suite. Aside of that we running this and other tests which creates and scales MSes in parallel, so we are excercising this functionality

This test creates another machineset with 0 desired replicas, then it scales it up to 3 desired replicas.
At the same time machineset which was created in `BeforeEach` block scales down to 0.
This test considering passed when both machinesets reach desired replicas (0 for initial machineset, 3 for newly created machineset) before the timeout (30m currently).

#### Managed cluster should reject invalid machinesets
* Parallel: Yes
* ExecTime: >6m on AWS
* Recommendations:
  * Get this out of this suite! No machines need to be created here.
  * Move this to another webhook tests

This test just tries to create machineset with empty provider config, but due to machineset is creating in `BeforeEach` block, it takes a while.
We are creating and waiting for 2 machines here for nothing...

#### Managed cluster should have ability to additively reconcile taints from machine to nodes
* Parallel: Yes
* ExecTime: >6m on AWS
* Recommendation:
  * Add check also for labels, or create new one which would exercise labels and annotations as well
  * Extend this test with another cases
    * What happen if taint will be removed from the spec? What are we expecting to happen? Some comments need to be added there, at least.
    * Same for labels ^

This test checks nodelink controller ability to reconcile taints from a machine spec and add it to respective node.
It picks a machine from the previously created machine spec and extends its spec with extra taint, then checks
that taint will appear on the node within 3 min. Also, within the test extra, 'not-from-machine' taint adds, seem to check
that the nodelink controller does not touch user-specified taints.

### Webhooks
Tests listed below share common `BeforeEach` and `AfterEach` blocks.
In `BeforeEach` platform-dependant 'minimal provider spec' for a machine is creating.
In `AfterEach` created resources (machines, machinesets) resources cleanup happens.

Recommendation: 
* It might be wise to collect all webhook-related tests within this suite, i.e. other webhook-related tests might be
extracted and put into this suite. For example, [Managed cluster should reject invalid machinesets](#managed-cluster-should-reject-invalid-machinesets) might fit here.
* Add test which checks that all webhooks infrastructure is configured properly for MAPI, CAO and CAPI perhaps (?). 

#### Webhooks should be able to create a machine from a minimal providerSpec
* Parallel: Yes
* ExecTime: ~5m on AWS

This test mainly checks machine-related mutating webhooks ability to set up defaults.
Webhook is expected to populate platform-dependant default values (like providerSpec.CredentialsSecret for vsphere machines).
Within the test, 'minimal provider spec' created in `BeforeEach` wraps into a machine manifest and sends it to the api server.
The test is considered passed if the machine hit the `Running` stage within 15m.

#### Webhooks should be able to create machines from a machineset with a minimal providerSpec
* Parallel: Yes
* ExecTime: ~5m on AWS

This test mainly checks machine-related mutating webhooks ability to set up defaults.
Webhook is expected to populate platform-dependant default values (like providerSpec.CredentialsSecret for vsphere machines).
Within the test, 'minimal provider spec' created in `BeforeEach` wraps into a machineset manifest and sends to the api server.
The test is considered passed if the machineset became fully ready within 30m.

#### Webhooks should return an error when removing required fields from the Machine providerSpec
#### Webhooks should return an error when removing required fields from the MachineSet providerSpec
* Parallel: Yes
* ExecTime: ~10s on AWS

These tests try to apply 'minimal provider spec' on an already existing machine/machineset. Since there are fields which expected to be presented
in resource (for example providerSpec.CredentialsSecret for vsphere machines), the request should be denied by the validation webhook.

### Lifecycle Hooks
#### Lifecycle Hooks should pause lifecycle actions when present
* Parallel: Yes
* ExecTime: >6m on AWS

This test checks pre-delete and pre-drain lifecycle hooks functionality.
The presence of pre-drain and pre-delete hooks should block respective (drain, delete) procedures till hooks would not be removed.
The test itself is quite straightforward and split into well-defined stages and checks:
1) Creates machineset, waits for the machine to be running, creates workload job on the machine
2) Sets up LifecycleHook on the machine (update machine spec basically)
3) Deletes the machine (scale down the machineset)
4) Checks pre-drain condition is false and workload is not evicted
5) Removes the pre-drain hook
6) Checks that the workload pod was evicted, but the machine is still present
7) Removes the pre-delete hook
8) Checks the machine is deleted

## MachineHealthCheck
These tests check MachineHealthCheck functionality. Tests listed below share common `BeforeEach` and `AfterEach` blocks.
Within `BeforeEach` machineset with 4 desired replicas creating. In `AfterEach` resources created during a test are attempted to be cleaned up.

Recommendation: Refactor these tests to run them in order for and use the same machinesets. Given that we are setting up conditions on machines, we could perform 
`maxUnhelthy` test first which does not suppose machine remediation. Test for actual remediation might be done on the same machineset. 

#### MachineHealthCheck should remediate unhealthy nodes
* Parallel: Yes
* ExecTime: >6m on AWS

Within this test, several machines (for now `maxUnhelthy` - 1, which is currently 1) marking as unhealthy with a specific condition.
Then MHC creating, such MHC is configured to observe this condition.
After this test waits for unhealthy machines to be deleted and checks that healthy machines stay intact.

This test exercises MHC remediation functionality.

#### MachineHealthCheck should not remediate larger number of unhealthy machines then maxUnhealthy
* Parallel: Yes
* ExecTime: >9m on AWS

Within this test, a number of machines exceeding `maxUnhelthy` marking as unhealthy with a specific condition.
Then MHC creating, such MHC is configured to observe this condition.
Test waits for a `RemediationRestricted` event on the machineset and checks that no machines were deleted.

This test exercises MHC behaviour that prevents remediation if a number of machines marked as unhealthy more than a number in `maxUnhelthy` parameter.

### Autoscaler

#### Autoscaler should use a ClusterAutoscaler that has 100 maximum total nodes count It scales from/to zero
* Parallel: No
* ExecTime: >7m on AWS
* Recommendation:
  * amount of desired machines might be reduced from 3 to 2

Within this test machineset with the initial desired 0 replicas and with a specific set of labels is created.
After, a Job with 3 desired replicas is attempting to be scheduled on nodes with labels configured for previously created machineset.

The test is expecting machineset to be scaled up to 3 machines and accept the payload. Then, created Job is removing and the machineset is expected to be scaled down back to 0 replicas. 

Test considered failed if scale up or down did not happen within 15 min timeout.

#### Autoscaler should use a ClusterAutoscaler that has 100 maximum total nodes count cleanup deletion information after scale down [Slow]
* Parallel: No
* ExecTime: ~13m on AWS
* Recommendation:
  * This test is a bit cryptic at first glance and requires at least some in-code explanation about the meaning of checks for taints and annotations presence. Having some documentation references will be beneficial here.
  * amount of desired machines might be reduced from 3 to 2 for each machineset

Within this test machineset with 1 desired replica and with a specific set of labels is created.
After, a Job with 6 desired replicas is attempting to be scheduled on nodes with labels configured for previously created machinesets.

The test is expecting both machinesets to be scaled up to 3 machines and accept the payload. Then, created Job is removing and the machineset is expected to be scaled down back to 1 replica.

After machinesets will be scaled back to 1 replica test checks if no cluster-autoscaler specific annotations are presented on machines and no autoscaler specific taints are presented on respective nodes from these machinesets.

Test considered failed if scale up or down did not happen, or after scaling down there are taints/annotations leftovers within a 3 minutes timeout.

#### Autoscaler should use a ClusterAutoscaler that has 12 maximum total nodes count and balance similar nodes enabled scales up and down while respecting MaxNodesTotal [Slow][Serial]
* Parallel: No
* ExecTime: ~13m on AWS
* Recommendation:
  * Since this test does not leverage or checking `balance similar nodes` feature, this might be extracted to separate group with reducing `MaxNodesTotal` number to ~8. This should speed up this test a bit and reduce resource consumption.
  * This test requires revise and extension of checks. Workload destiny need to be checked too. As i understand this test at the moment of writing, part of the workload should not be scheduled due to `MaxNodesTotal` limit.
  * Gather and report a number of nodes when the test starts
  * Might be merged with from/to zero scaling test ([Autoscaler should use a ClusterAutoscaler that has 100 maximum total nodes count It scales from/to zero](#autoscaler-should-use-a-clusterautoscaler-that-has-100-maximum-total-nodes-count-it-scales-fromto-zero)).

This test excersising `MaxNodesTotal` limit for the autoscaler. Intially it creates machineset with 1 desired replica and a Job
with `MaxNodesTotal` + 1 desired pods in it. Then waits cluster to be scaled up till `MaxNodesTotal` nodes number and checks that this constraint was not violated.

Then workload deletes and test waits for the machineset to be scaled back to 1 replica.

#### Autoscaler should use a ClusterAutoscaler that has 12 maximum total nodes count and balance similar nodes enabled places nodes evenly across node groups [Slow]
* Parallel: No
* ExecTime: ~10m on AWS
* Recommendation:
  * Check kubernetes-autoscaler deployment for the `balance-similar-node-groups` flag presence first
  * Currently max machineset size for machine autoscaler resource is 3 within this test. I suggest to increase it to 4 or 6 for being more sure that `balance-similar-node-groups` does its thing and that the test pass is not a coincidence.

This test excersises [balance-similar-node-groups](https://github.com/openshift/kubernetes-autoscaler/blob/master/cluster-autoscaler/FAQ.md#im-running-cluster-with-nodes-in-multiple-zones-for-ha-purposes-is-that-supported-by-cluster-autoscaler)
feature of the kubernetes-autoscaler and, implicitly, the ability of the `cluster-autoscaler-operator` to pipe down this flag to the
kubernetes-autoscaler deployment.

Within this test, two machinesets with an initial 1 desired replica are created. Then by creating a Job with 4 desired replicas
targeting created machinesets related nodes test tries to trigger cluster expansion by 2 nodes (2 machines/nodes were created initially during machinesets creation, should be 4 at the end).
Then test checks that at the end both machinesets will be scaled up by 1 machine and each machineset has 2 replicas.
Expected that cluster autoscaler will place workload pods on each of the 2 machinesets created.
