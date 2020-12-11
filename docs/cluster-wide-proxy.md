# Overview

Confirms the behavior of the machine-api-controller when a [cluster-wide proxy](https://docs.openshift.com/container-platform/4.6/networking/enable-cluster-wide-proxy.html) is configured and unconfigured.  

In order to allow the cluster-wide proxy to be configured, a proxy is deployed as a daemonset in the cluster and is 
accessed by nodes and pods via the service network.  This was done due to variances in security configuration between 
cloud providers. Originally, this test deployed a separate standalone node which hosted a proxy via the host network.  
This worked for every cloud provider except for AWS.

This test aims to confirm the following:

- Confirm that a reencrypting man in the middle proxy exposing a custom signer is usuable with the machine-api-controller.
  i.e. Confirm that the `machine-api-controller` consumes and uses a custom PKI. 
- Confirm the `machine-api-controller` deployment can respond to changes in proxy configuration
- Confirm that a machine set can be created and destroyed

