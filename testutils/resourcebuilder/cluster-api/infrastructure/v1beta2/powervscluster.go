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
	capibmv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

// PowerVSCluster creates a new PowerVSClusterBuilder.
func PowerVSCluster() PowerVSClusterBuilder {
	return PowerVSClusterBuilder{}
}

// PowerVSClusterBuilder is used to build out an PowerVSCluster object.
type PowerVSClusterBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string

	// Spec fields.
	controlPlaneEndpoint clusterv1.APIEndpoint
	loadBalancers        []capibmv1.VPCLoadBalancerSpec
	network              capibmv1.IBMPowerVSResourceReference
	resourceGroup        *capibmv1.IBMPowerVSResourceReference
	serviceInstance      *capibmv1.IBMPowerVSResourceReference
	zone                 *string

	// Status fields.
	conditions clusterv1.Conditions
	ready      bool
}

// Build builds a new IBMPowerVSCluster based on the configuration provided.
func (p PowerVSClusterBuilder) Build() *capibmv1.IBMPowerVSCluster {
	return &capibmv1.IBMPowerVSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IBMPowerVSCluster",
			APIVersion: capibmv1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              p.name,
			Namespace:         p.namespace,
			CreationTimestamp: p.creationTimestamp,
			DeletionTimestamp: p.deletionTimestamp,
			Labels:            p.labels,
			Annotations:       p.annotations,
			GenerateName:      p.generateName,
		},
		Spec: capibmv1.IBMPowerVSClusterSpec{
			Network:              p.network,
			ControlPlaneEndpoint: p.controlPlaneEndpoint,
			ServiceInstance:      p.serviceInstance,
			Zone:                 p.zone,
			ResourceGroup:        p.resourceGroup,
			LoadBalancers:        p.loadBalancers,
		},
		Status: capibmv1.IBMPowerVSClusterStatus{
			Ready:      p.ready,
			Conditions: p.conditions,
		},
	}
}

// Object meta fields.

// WithAnnotations sets the annotations for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithAnnotations(annotations map[string]string) PowerVSClusterBuilder {
	p.annotations = annotations
	return p
}

// WithCreationTimestamp sets the creationTimestamp for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithCreationTimestamp(timestamp metav1.Time) PowerVSClusterBuilder {
	p.creationTimestamp = timestamp
	return p
}

// WithDeletionTimestamp sets the deletionTimestamp for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithDeletionTimestamp(timestamp *metav1.Time) PowerVSClusterBuilder {
	p.deletionTimestamp = timestamp
	return p
}

// WithGenerateName sets the generateName for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithGenerateName(generateName string) PowerVSClusterBuilder {
	p.generateName = generateName
	return p
}

// WithLabels sets the labels for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithLabels(labels map[string]string) PowerVSClusterBuilder {
	p.labels = labels
	return p
}

// WithName sets the name for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithName(name string) PowerVSClusterBuilder {
	p.name = name
	return p
}

// WithNamespace sets the namespace for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithNamespace(namespace string) PowerVSClusterBuilder {
	p.namespace = namespace
	return p
}

// Spec fields.

// WithControlPlaneEndpoint sets the controlPlaneEndpoint for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) PowerVSClusterBuilder {
	p.controlPlaneEndpoint = endpoint
	return p
}

// WithLoadBalancer sets the loadBalancers for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithLoadBalancer(loadBalancers []capibmv1.VPCLoadBalancerSpec) PowerVSClusterBuilder {
	p.loadBalancers = loadBalancers
	return p
}

// WithNetwork sets the network for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithNetwork(network capibmv1.IBMPowerVSResourceReference) PowerVSClusterBuilder {
	p.network = network
	return p
}

// WithResourceGroup sets the resourceGroup for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithResourceGroup(resourceGroup *capibmv1.IBMPowerVSResourceReference) PowerVSClusterBuilder {
	p.resourceGroup = resourceGroup
	return p
}

// WithServiceInstance sets the serviceInstance for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithServiceInstance(serviceInstance *capibmv1.IBMPowerVSResourceReference) PowerVSClusterBuilder {
	p.serviceInstance = serviceInstance
	return p
}

// WithZone sets the zone for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithZone(zone *string) PowerVSClusterBuilder {
	p.zone = zone
	return p
}

// Status fields.

// WithConditions sets the conditions for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithConditions(conditions clusterv1.Conditions) PowerVSClusterBuilder {
	p.conditions = conditions
	return p
}

// WithReady sets the ready status for the IBMPowerVSCluster builder.
func (p PowerVSClusterBuilder) WithReady(ready bool) PowerVSClusterBuilder {
	p.ready = ready
	return p
}
