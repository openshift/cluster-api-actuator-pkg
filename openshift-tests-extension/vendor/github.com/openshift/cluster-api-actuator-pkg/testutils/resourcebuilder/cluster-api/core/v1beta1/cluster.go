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

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

// Cluster creates a new cluster builder.
func Cluster() ClusterBuilder {
	return ClusterBuilder{}
}

// ClusterBuilder is used to build out a Cluster object.
type ClusterBuilder struct {
	// Object meta fields.
	annotations       map[string]string
	creationTimestamp metav1.Time
	deletionTimestamp *metav1.Time
	generateName      string
	labels            map[string]string
	name              string
	namespace         string
	ownerReferences   []metav1.OwnerReference

	// Spec fields.
	clusterNetwork       *capiv1.ClusterNetwork
	controlPlaneEndpoint capiv1.APIEndpoint
	controlPlaneRef      *corev1.ObjectReference
	infrastructureRef    *corev1.ObjectReference
	paused               bool
	topology             *capiv1.Topology

	// Status fields.
	conditions          capiv1.Conditions
	controlPlaneReady   bool
	failureDomains      capiv1.FailureDomains
	failureMessage      *string
	failureReason       *capierrors.ClusterStatusError
	infrastructureReady bool
	observedGeneration  int64
	phase               string
}

// Build builds a new cluster based on the configuration provided.
func (c ClusterBuilder) Build() *capiv1.Cluster {
	cluster := &capiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       c.annotations,
			CreationTimestamp: c.creationTimestamp,
			DeletionTimestamp: c.deletionTimestamp,
			GenerateName:      c.generateName,
			Labels:            c.labels,
			Name:              c.name,
			Namespace:         c.namespace,
			OwnerReferences:   c.ownerReferences,
		},
		Spec: capiv1.ClusterSpec{
			ClusterNetwork:       c.clusterNetwork,
			ControlPlaneEndpoint: c.controlPlaneEndpoint,
			ControlPlaneRef:      c.controlPlaneRef,
			InfrastructureRef:    c.infrastructureRef,
			Paused:               c.paused,
			Topology:             c.topology,
		},
		Status: capiv1.ClusterStatus{
			Conditions:          c.conditions,
			ControlPlaneReady:   c.controlPlaneReady,
			FailureDomains:      c.failureDomains,
			FailureMessage:      c.failureMessage,
			FailureReason:       c.failureReason,
			InfrastructureReady: c.infrastructureReady,
			ObservedGeneration:  c.observedGeneration,
			Phase:               c.phase,
		},
	}

	return cluster
}

// Object meta field.

// WithAnnotations sets the annotations for the cluster builder.
func (c ClusterBuilder) WithAnnotations(annotations map[string]string) ClusterBuilder {
	c.annotations = annotations
	return c
}

// WithGenerateName sets the generateName for the cluster builder.
func (c ClusterBuilder) WithGenerateName(generateName string) ClusterBuilder {
	c.generateName = generateName
	return c
}

// WithCreationTimestamp sets the creationTimestamp for the cluster builder.
func (c ClusterBuilder) WithCreationTimestamp(timestamp metav1.Time) ClusterBuilder {
	c.creationTimestamp = timestamp
	return c
}

// WithDeletionTimestamp sets the deletionTimestamp for the cluster builder.
func (c ClusterBuilder) WithDeletionTimestamp(timestamp *metav1.Time) ClusterBuilder {
	c.deletionTimestamp = timestamp
	return c
}

// WithLabels sets the labels for the cluster builder.
func (c ClusterBuilder) WithLabels(labels map[string]string) ClusterBuilder {
	c.labels = labels
	return c
}

// WithName sets the name for the cluster builder.
func (c ClusterBuilder) WithName(name string) ClusterBuilder {
	c.name = name
	return c
}

// WithNamespace sets the namespace for the cluster builder.
func (c ClusterBuilder) WithNamespace(namespace string) ClusterBuilder {
	c.namespace = namespace
	return c
}

// WithOwnerReferences sets the OwnerReferences for the cluster builder.
func (c ClusterBuilder) WithOwnerReferences(ownerRefs []metav1.OwnerReference) ClusterBuilder {
	c.ownerReferences = ownerRefs
	return c
}

// Spec fields.

// WithClusterNetwork sets the cluster network for the cluster builder.
func (c ClusterBuilder) WithClusterNetwork(network *capiv1.ClusterNetwork) ClusterBuilder {
	c.clusterNetwork = network
	return c
}

// WithControlPlaneEndpoint sets the control plane endpoint for the cluster builder.
func (c ClusterBuilder) WithControlPlaneEndpoint(endpoint capiv1.APIEndpoint) ClusterBuilder {
	c.controlPlaneEndpoint = endpoint
	return c
}

// WithControlPlaneRef sets the control plane reference for the cluster builder.
func (c ClusterBuilder) WithControlPlaneRef(ref *corev1.ObjectReference) ClusterBuilder {
	c.controlPlaneRef = ref
	return c
}

// WithInfrastructureRef sets the infrastructure reference for the cluster builder.
func (c ClusterBuilder) WithInfrastructureRef(ref *corev1.ObjectReference) ClusterBuilder {
	c.infrastructureRef = ref
	return c
}

// WithPaused sets the paused state for the cluster builder.
func (c ClusterBuilder) WithPaused(paused bool) ClusterBuilder {
	c.paused = paused
	return c
}

// WithTopology sets the topology for the cluster builder.
func (c ClusterBuilder) WithTopology(topology *capiv1.Topology) ClusterBuilder {
	c.topology = topology
	return c
}

// Status fields.

// WithConditions sets the conditions for the cluster builder.
func (c ClusterBuilder) WithConditions(conditions capiv1.Conditions) ClusterBuilder {
	c.conditions = conditions
	return c
}

// WithControlPlaneReady sets the control plane ready state for the cluster builder.
func (c ClusterBuilder) WithControlPlaneReady(ready bool) ClusterBuilder {
	c.controlPlaneReady = ready
	return c
}

// WithFailureDomains sets the failure domains for the cluster builder.
func (c ClusterBuilder) WithFailureDomains(failureDomains capiv1.FailureDomains) ClusterBuilder {
	c.failureDomains = failureDomains
	return c
}

// WithFailureMessage sets the failure message for the cluster builder.
func (c ClusterBuilder) WithFailureMessage(message string) ClusterBuilder {
	c.failureMessage = &message
	return c
}

// WithFailureReason sets the failure reason for the cluster builder.
func (c ClusterBuilder) WithFailureReason(reason capierrors.ClusterStatusError) ClusterBuilder {
	c.failureReason = &reason
	return c
}

// WithInfrastructureReady sets the infrastructure ready state for the cluster builder.
func (c ClusterBuilder) WithInfrastructureReady(ready bool) ClusterBuilder {
	c.infrastructureReady = ready
	return c
}

// WithObservedGeneration sets the observed generation for the cluster builder.
func (c ClusterBuilder) WithObservedGeneration(generation int64) ClusterBuilder {
	c.observedGeneration = generation
	return c
}

// WithPhase sets the phase for the cluster builder.
func (c ClusterBuilder) WithPhase(phase string) ClusterBuilder {
	c.phase = phase
	return c
}
