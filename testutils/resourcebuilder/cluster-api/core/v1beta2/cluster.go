/*
Copyright 2025 Red Hat, Inc.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

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
	availabilityGates    []clusterv1.ClusterAvailabilityGate
	clusterNetwork       clusterv1.ClusterNetwork
	controlPlaneEndpoint clusterv1.APIEndpoint
	controlPlaneRef      clusterv1.ContractVersionedObjectReference
	infrastructureRef    clusterv1.ContractVersionedObjectReference
	paused               *bool
	topology             clusterv1.Topology

	// Status fields.
	conditions                []metav1.Condition
	controlPlane              *clusterv1.ClusterControlPlaneStatus
	v1Beta1Conditions         clusterv1.Conditions
	controlPlaneInitialized   *bool
	failureDomains            []clusterv1.FailureDomain
	v1Beta1FailureMessage     *string
	v1Beta1FailureReason      *capierrors.ClusterStatusError
	infrastructureProvisioned *bool
	observedGeneration        int64
	phase                     string
	workers                   *clusterv1.WorkersStatus
}

// Build builds a new cluster based on the configuration provided.
func (c ClusterBuilder) Build() *clusterv1.Cluster {
	cluster := &clusterv1.Cluster{
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
		Spec: clusterv1.ClusterSpec{
			AvailabilityGates:    c.availabilityGates,
			ClusterNetwork:       c.clusterNetwork,
			ControlPlaneEndpoint: c.controlPlaneEndpoint,
			ControlPlaneRef:      c.controlPlaneRef,
			InfrastructureRef:    c.infrastructureRef,
			Paused:               c.paused,
			Topology:             c.topology,
		},
		Status: clusterv1.ClusterStatus{
			Conditions:   c.conditions,
			ControlPlane: c.controlPlane,
			Deprecated: &clusterv1.ClusterDeprecatedStatus{
				V1Beta1: &clusterv1.ClusterV1Beta1DeprecatedStatus{
					Conditions:     c.v1Beta1Conditions,
					FailureMessage: c.v1Beta1FailureMessage,
					FailureReason:  c.v1Beta1FailureReason,
				},
			},
			Initialization: clusterv1.ClusterInitializationStatus{
				InfrastructureProvisioned: c.infrastructureProvisioned,
				ControlPlaneInitialized:   c.controlPlaneInitialized,
			},
			FailureDomains:     c.failureDomains,
			ObservedGeneration: c.observedGeneration,
			Phase:              c.phase,
			Workers:            c.workers,
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

// WithAvailabilityGates sets the availability gates for the cluster builder.
func (c ClusterBuilder) WithAvailabilityGates(gates []clusterv1.ClusterAvailabilityGate) ClusterBuilder {
	c.availabilityGates = gates
	return c
}

// WithClusterNetwork sets the cluster network for the cluster builder.
func (c ClusterBuilder) WithClusterNetwork(network clusterv1.ClusterNetwork) ClusterBuilder {
	c.clusterNetwork = network
	return c
}

// WithControlPlaneEndpoint sets the control plane endpoint for the cluster builder.
func (c ClusterBuilder) WithControlPlaneEndpoint(endpoint clusterv1.APIEndpoint) ClusterBuilder {
	c.controlPlaneEndpoint = endpoint
	return c
}

// WithControlPlaneRef sets the control plane reference for the cluster builder.
func (c ClusterBuilder) WithControlPlaneRef(ref clusterv1.ContractVersionedObjectReference) ClusterBuilder {
	c.controlPlaneRef = ref
	return c
}

// WithInfrastructureRef sets the infrastructure reference for the cluster builder.
func (c ClusterBuilder) WithInfrastructureRef(ref clusterv1.ContractVersionedObjectReference) ClusterBuilder {
	c.infrastructureRef = ref
	return c
}

// WithPaused sets the paused state for the cluster builder.
func (c ClusterBuilder) WithPaused(paused bool) ClusterBuilder {
	c.paused = &paused
	return c
}

// WithTopology sets the topology for the cluster builder.
func (c ClusterBuilder) WithTopology(topology clusterv1.Topology) ClusterBuilder {
	c.topology = topology
	return c
}

// Status fields.

// WithConditions sets the conditions for the cluster builder.
func (c ClusterBuilder) WithConditions(conditions []metav1.Condition) ClusterBuilder {
	c.conditions = conditions
	return c
}

// WithV1Beta1Conditions sets the v1beta1 conditions for the cluster builder.
func (c ClusterBuilder) WithV1Beta1Conditions(conditions clusterv1.Conditions) ClusterBuilder {
	c.v1Beta1Conditions = conditions
	return c
}

// WithControlPlaneInitialized sets the control plane initialized state for the cluster builder.
func (c ClusterBuilder) WithControlPlaneInitialized(initialized bool) ClusterBuilder {
	c.controlPlaneInitialized = &initialized
	return c
}

// WithFailureDomains sets the failure domains for the cluster builder.
func (c ClusterBuilder) WithFailureDomains(failureDomains []clusterv1.FailureDomain) ClusterBuilder {
	c.failureDomains = failureDomains
	return c
}

// WithV1Beta1FailureMessage sets the v1beta1 failure message for the cluster builder.
func (c ClusterBuilder) WithV1Beta1FailureMessage(message string) ClusterBuilder {
	c.v1Beta1FailureMessage = &message
	return c
}

// WithV1Beta1FailureReason sets the v1beta1 failure reason for the cluster builder.
func (c ClusterBuilder) WithV1Beta1FailureReason(reason capierrors.ClusterStatusError) ClusterBuilder {
	c.v1Beta1FailureReason = &reason
	return c
}

// WithInfrastructureProvisioned sets the infrastructure ready state for the cluster builder.
func (c ClusterBuilder) WithInfrastructureProvisioned(provisioned bool) ClusterBuilder {
	c.infrastructureProvisioned = &provisioned
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

// WithControlPlaneStatus sets the control plane status for the cluster builder.
func (c ClusterBuilder) WithControlPlaneStatus(controlPlane *clusterv1.ClusterControlPlaneStatus) ClusterBuilder {
	c.controlPlane = controlPlane
	return c
}

// WithWorkersStatus sets the workers status for the cluster builder.
func (c ClusterBuilder) WithWorkersStatus(workers *clusterv1.WorkersStatus) ClusterBuilder {
	c.workers = workers
	return c
}
