/*
Copyright 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// AWSCluster creates a new AWSClusterBuilder.
func AWSCluster() AWSClusterBuilder {
	return AWSClusterBuilder{}
}

// AWSClusterBuilder is used to build out an AWSCluster object.
type AWSClusterBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string

	// Spec fields.
	additionalTags                    capav1.Tags
	bastion                           capav1.Bastion
	controlPlaneEndpoint              clusterv1.APIEndpoint
	controlPlaneLoadBalancer          *capav1.AWSLoadBalancerSpec
	identityRef                       *capav1.AWSIdentityReference
	imageLookupBaseOS                 string
	imageLookupFormat                 string
	imageLookupOrg                    string
	networkSpec                       capav1.NetworkSpec
	partition                         string
	region                            string
	s3Bucket                          *capav1.S3Bucket
	secondaryControlPlaneLoadBalancer *capav1.AWSLoadBalancerSpec
	sshKeyName                        *string

	// Status fields.
	bastionInstance *capav1.Instance
	conditions      clusterv1.Conditions
	failureDomains  clusterv1.FailureDomains
	network         capav1.NetworkStatus
	ready           bool
}

// Build builds a new AWSCluster based on the configuration provided.
func (a AWSClusterBuilder) Build() *capav1.AWSCluster {
	awsCluster := &capav1.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       a.annotations,
			CreationTimestamp: a.creationTimestamp,
			DeletionTimestamp: a.deletionTimestamp,
			GenerateName:      a.generateName,
			Labels:            a.labels,
			Name:              a.name,
			Namespace:         a.namespace,
		},
		Spec: capav1.AWSClusterSpec{
			AdditionalTags:                    a.additionalTags,
			Bastion:                           a.bastion,
			ControlPlaneEndpoint:              a.controlPlaneEndpoint,
			ControlPlaneLoadBalancer:          a.controlPlaneLoadBalancer,
			IdentityRef:                       a.identityRef,
			ImageLookupBaseOS:                 a.imageLookupBaseOS,
			ImageLookupFormat:                 a.imageLookupFormat,
			ImageLookupOrg:                    a.imageLookupOrg,
			NetworkSpec:                       a.networkSpec,
			Partition:                         a.partition,
			Region:                            a.region,
			S3Bucket:                          a.s3Bucket,
			SSHKeyName:                        a.sshKeyName,
			SecondaryControlPlaneLoadBalancer: a.secondaryControlPlaneLoadBalancer,
		},
		Status: capav1.AWSClusterStatus{
			Bastion:        a.bastionInstance,
			Conditions:     a.conditions,
			FailureDomains: a.failureDomains,
			Network:        a.network,
			Ready:          a.ready,
		},
	}

	return awsCluster
}

// Object meta fields.

// WithAnnotations sets the annotations for the AWSCluster builder.
func (a AWSClusterBuilder) WithAnnotations(annotations map[string]string) AWSClusterBuilder {
	a.annotations = annotations
	return a
}

// WithCreationTimestamp sets the creationTimestamp for the AWSCluster builder.
func (a AWSClusterBuilder) WithCreationTimestamp(timestamp metav1.Time) AWSClusterBuilder {
	a.creationTimestamp = timestamp
	return a
}

// WithDeletionTimestamp sets the deletionTimestamp for the AWSCluster builder.
func (a AWSClusterBuilder) WithDeletionTimestamp(timestamp *metav1.Time) AWSClusterBuilder {
	a.deletionTimestamp = timestamp
	return a
}

// WithGenerateName sets the generateName for the AWSCluster builder.
func (a AWSClusterBuilder) WithGenerateName(generateName string) AWSClusterBuilder {
	a.generateName = generateName
	return a
}

// WithLabels sets the labels for the AWSCluster builder.
func (a AWSClusterBuilder) WithLabels(labels map[string]string) AWSClusterBuilder {
	a.labels = labels
	return a
}

// WithName sets the name for the AWSCluster builder.
func (a AWSClusterBuilder) WithName(name string) AWSClusterBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the AWSCluster builder.
func (a AWSClusterBuilder) WithNamespace(namespace string) AWSClusterBuilder {
	a.namespace = namespace
	return a
}

// Spec fields.

// WithAdditionalTags sets the additionalTags for the AWSCluster builder.
func (a AWSClusterBuilder) WithAdditionalTags(tags capav1.Tags) AWSClusterBuilder {
	a.additionalTags = tags
	return a
}

// WithBastion sets the bastion for the AWSCluster builder.
func (a AWSClusterBuilder) WithBastion(bastion capav1.Bastion) AWSClusterBuilder {
	a.bastion = bastion
	return a
}

// WithControlPlaneEndpoint sets the controlPlaneEndpoint for the AWSCluster builder.
func (a AWSClusterBuilder) WithControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) AWSClusterBuilder {
	a.controlPlaneEndpoint = endpoint
	return a
}

// WithControlPlaneLoadBalancer sets the controlPlaneLoadBalancer for the AWSCluster builder.
func (a AWSClusterBuilder) WithControlPlaneLoadBalancer(lb *capav1.AWSLoadBalancerSpec) AWSClusterBuilder {
	a.controlPlaneLoadBalancer = lb
	return a
}

// WithIdentityRef sets the identityRef for the AWSCluster builder.
func (a AWSClusterBuilder) WithIdentityRef(identityRef *capav1.AWSIdentityReference) AWSClusterBuilder {
	a.identityRef = identityRef
	return a
}

// WithImageLookupFormat sets the imageLookupFormat for the AWSCluster builder.
func (a AWSClusterBuilder) WithImageLookupFormat(format string) AWSClusterBuilder {
	a.imageLookupFormat = format
	return a
}

// WithImageLookupOrg sets the imageLookupOrg for the AWSCluster builder.
func (a AWSClusterBuilder) WithImageLookupOrg(org string) AWSClusterBuilder {
	a.imageLookupOrg = org
	return a
}

// WithImageLookupBaseOS sets the imageLookupBaseOS for the AWSCluster builder.
func (a AWSClusterBuilder) WithImageLookupBaseOS(baseOS string) AWSClusterBuilder {
	a.imageLookupBaseOS = baseOS
	return a
}

// WithNetworkSpec sets the networkSpec for the AWSCluster builder.
func (a AWSClusterBuilder) WithNetworkSpec(networkSpec capav1.NetworkSpec) AWSClusterBuilder {
	a.networkSpec = networkSpec
	return a
}

// WithPartition sets the partition for the AWSCluster builder.
func (a AWSClusterBuilder) WithPartition(partition string) AWSClusterBuilder {
	a.partition = partition
	return a
}

// WithRegion sets the region for the AWSCluster builder.
func (a AWSClusterBuilder) WithRegion(region string) AWSClusterBuilder {
	a.region = region
	return a
}

// WithSecondaryControlPlaneLoadBalancer sets the secondaryControlPlaneLoadBalancer for the AWSCluster builder.
func (a AWSClusterBuilder) WithSecondaryControlPlaneLoadBalancer(lb *capav1.AWSLoadBalancerSpec) AWSClusterBuilder {
	a.secondaryControlPlaneLoadBalancer = lb
	return a
}

// WithSSHKeyName sets the sshKeyName for the AWSCluster builder.
func (a AWSClusterBuilder) WithSSHKeyName(sshKeyName string) AWSClusterBuilder {
	a.sshKeyName = &sshKeyName
	return a
}

// WithS3Bucket sets the s3Bucket for the AWSCluster builder.
func (a AWSClusterBuilder) WithS3Bucket(s3Bucket *capav1.S3Bucket) AWSClusterBuilder {
	a.s3Bucket = s3Bucket
	return a
}

// Status fields.

// WithBastionStatus sets the bastion status for the AWSCluster builder.
func (a AWSClusterBuilder) WithBastionStatus(bastionInstance *capav1.Instance) AWSClusterBuilder {
	a.bastionInstance = bastionInstance
	return a
}

// WithConditions sets the conditions for the AWSCluster builder.
func (a AWSClusterBuilder) WithConditions(conditions clusterv1.Conditions) AWSClusterBuilder {
	a.conditions = conditions
	return a
}

// WithFailureDomains sets the failureDomains for the AWSCluster builder.
func (a AWSClusterBuilder) WithFailureDomains(failureDomains clusterv1.FailureDomains) AWSClusterBuilder {
	a.failureDomains = failureDomains
	return a
}

// WithNetwork sets the network status for the AWSCluster builder.
func (a AWSClusterBuilder) WithNetwork(network capav1.NetworkStatus) AWSClusterBuilder {
	a.network = network
	return a
}

// WithReady sets the ready status for the AWSCluster builder.
func (a AWSClusterBuilder) WithReady(ready bool) AWSClusterBuilder {
	a.ready = ready
	return a
}
