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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

// Machine creates a new machine builder.
func Machine() MachineBuilder {
	return MachineBuilder{}
}

// MachineBuilder is used to build out a Machine object.
type MachineBuilder struct {
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
	bootstrap                      clusterv1.Bootstrap
	clusterName                    string
	failureDomain                  string
	infrastructureRef              clusterv1.ContractVersionedObjectReference
	minReadySeconds                *int32
	nodeDeletionTimeoutSeconds     *int32
	nodeDrainTimeoutSeconds        *int32
	nodeVolumeDetachTimeoutSeconds *int32
	providerID                     string
	readinessGates                 []clusterv1.MachineReadinessGate
	version                        string

	// Status fields.
	addresses                  clusterv1.MachineAddresses
	bootstrapDataSecretCreated *bool
	certificatesExpiryDate     metav1.Time
	conditions                 []metav1.Condition
	deletion                   *clusterv1.MachineDeletionStatus
	v1Beta1Conditions          clusterv1.Conditions
	v1Beta1FailureMessage      *string
	v1Beta1FailureReason       *capierrors.MachineStatusError
	infrastructureProvisioned  *bool
	lastUpdated                metav1.Time
	nodeInfo                   *corev1.NodeSystemInfo
	nodeRef                    clusterv1.MachineNodeReference
	observedGeneration         int64
	phase                      clusterv1.MachinePhase
}

// Build builds a new Machine based on the configuration provided.
func (m MachineBuilder) Build() *clusterv1.Machine {
	machine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       m.annotations,
			CreationTimestamp: m.creationTimestamp,
			DeletionTimestamp: m.deletionTimestamp,
			GenerateName:      m.generateName,
			Labels:            m.labels,
			Name:              m.name,
			Namespace:         m.namespace,
			OwnerReferences:   m.ownerReferences,
		},
		Spec: clusterv1.MachineSpec{
			Bootstrap:         m.bootstrap,
			ClusterName:       m.clusterName,
			FailureDomain:     m.failureDomain,
			InfrastructureRef: m.infrastructureRef,
			MinReadySeconds:   m.minReadySeconds,
			Deletion: clusterv1.MachineDeletionSpec{
				NodeDeletionTimeoutSeconds:     m.nodeDeletionTimeoutSeconds,
				NodeDrainTimeoutSeconds:        m.nodeDrainTimeoutSeconds,
				NodeVolumeDetachTimeoutSeconds: m.nodeVolumeDetachTimeoutSeconds,
			},
			ProviderID:     m.providerID,
			ReadinessGates: m.readinessGates,
			Version:        m.version,
		},
		Status: clusterv1.MachineStatus{
			Addresses:              m.addresses,
			CertificatesExpiryDate: m.certificatesExpiryDate,
			Conditions:             m.conditions,
			Deletion:               m.deletion,
			Deprecated: &clusterv1.MachineDeprecatedStatus{
				V1Beta1: &clusterv1.MachineV1Beta1DeprecatedStatus{
					Conditions:     m.v1Beta1Conditions,
					FailureMessage: m.v1Beta1FailureMessage,
					FailureReason:  m.v1Beta1FailureReason,
				},
			},
			Initialization: clusterv1.MachineInitializationStatus{
				BootstrapDataSecretCreated: m.bootstrapDataSecretCreated,
				InfrastructureProvisioned:  m.infrastructureProvisioned,
			},
			LastUpdated:        m.lastUpdated,
			NodeInfo:           m.nodeInfo,
			NodeRef:            m.nodeRef,
			ObservedGeneration: m.observedGeneration,
			Phase:              string(m.phase),
		},
	}

	return machine
}

// Object meta fields.

// WithAnnotations sets the Annotations for the machine builder.
func (m MachineBuilder) WithAnnotations(annotations map[string]string) MachineBuilder {
	m.annotations = annotations
	return m
}

// WithCreationTimestamp sets the creationTimestamp for the machine builder.
func (m MachineBuilder) WithCreationTimestamp(timestamp metav1.Time) MachineBuilder {
	m.creationTimestamp = timestamp
	return m
}

// WithDeletionTimestamp sets the deletionTimestamp for the machine builder.
func (m MachineBuilder) WithDeletionTimestamp(timestamp *metav1.Time) MachineBuilder {
	m.deletionTimestamp = timestamp
	return m
}

// WithGenerateName sets the generateName for the machine builder.
func (m MachineBuilder) WithGenerateName(generateName string) MachineBuilder {
	m.generateName = generateName
	return m
}

// WithLabels sets the Labels for the machine builder.
func (m MachineBuilder) WithLabels(labels map[string]string) MachineBuilder {
	m.labels = labels
	return m
}

// WithName sets the Name for the machine builder.
func (m MachineBuilder) WithName(name string) MachineBuilder {
	m.name = name
	return m
}

// WithNamespace sets the Namespace for the machine builder.
func (m MachineBuilder) WithNamespace(namespace string) MachineBuilder {
	m.namespace = namespace
	return m
}

// WithOwnerReferences sets the OwnerReferences for the machine builder.
func (m MachineBuilder) WithOwnerReferences(ownerRefs []metav1.OwnerReference) MachineBuilder {
	m.ownerReferences = ownerRefs
	return m
}

// Spec fields.

// WithBootstrap sets the Bootstrap for the machine builder.
func (m MachineBuilder) WithBootstrap(bootstrap clusterv1.Bootstrap) MachineBuilder {
	m.bootstrap = bootstrap
	return m
}

// WithClusterName sets the ClusterName for the machine builder.
func (m MachineBuilder) WithClusterName(clusterName string) MachineBuilder {
	m.clusterName = clusterName
	return m
}

// WithFailureDomain sets the FailureDomain for the machine builder.
func (m MachineBuilder) WithFailureDomain(failureDomain string) MachineBuilder {
	m.failureDomain = failureDomain
	return m
}

// WithInfrastructureRef sets the InfrastructureRef for the machine builder.
func (m MachineBuilder) WithInfrastructureRef(infraRef clusterv1.ContractVersionedObjectReference) MachineBuilder {
	m.infrastructureRef = infraRef
	return m
}

// WithMinReadySeconds sets the MinReadySeconds for the machine builder.
func (m MachineBuilder) WithMinReadySeconds(seconds int32) MachineBuilder {
	m.minReadySeconds = &seconds
	return m
}

// WithNodeDeletionTimeout sets the NodeDeletionTimeout for the machine builder.
func (m MachineBuilder) WithNodeDeletionTimeoutSeconds(timeoutSeconds int32) MachineBuilder {
	m.nodeDeletionTimeoutSeconds = &timeoutSeconds
	return m
}

// WithNodeDrainTimeout sets the NodeDrainTimeout for the machine builder.
func (m MachineBuilder) WithNodeDrainTimeoutSeconds(timeoutSeconds int32) MachineBuilder {
	m.nodeDrainTimeoutSeconds = &timeoutSeconds
	return m
}

// WithNodeVolumeDetachTimeout sets the NodeVolumeDetachTimeout for the machine builder.
func (m MachineBuilder) WithNodeVolumeDetachTimeoutSeconds(timeoutSeconds int32) MachineBuilder {
	m.nodeVolumeDetachTimeoutSeconds = &timeoutSeconds
	return m
}

// WithNodeRef sets the NodeRef for the machine builder.
func (m MachineBuilder) WithNodeRef(nodeRef clusterv1.MachineNodeReference) MachineBuilder {
	m.nodeRef = nodeRef
	return m
}

// WithProviderID sets the ProviderID for the machine builder.
func (m MachineBuilder) WithProviderID(providerID string) MachineBuilder {
	m.providerID = providerID
	return m
}

// WithReadinessGates sets the ReadinessGates for the machine builder.
func (m MachineBuilder) WithReadinessGates(gates []clusterv1.MachineReadinessGate) MachineBuilder {
	m.readinessGates = gates
	return m
}

// WithVersion sets the Version for the machine builder.
func (m MachineBuilder) WithVersion(version string) MachineBuilder {
	m.version = version
	return m
}

// Status Fields.

// WithAddresses sets the Addresses for the machine builder.
func (m MachineBuilder) WithAddresses(addresses clusterv1.MachineAddresses) MachineBuilder {
	m.addresses = addresses
	return m
}

// WithBootstrapDataSecretCreated sets the BootstrapDataSecretCreated for the machine builder.
func (m MachineBuilder) WithBootstrapDataSecretCreated(created bool) MachineBuilder {
	m.bootstrapDataSecretCreated = &created
	return m
}

// WithCertificatesExpiryDate sets the CertificatesExpiryDate for the machine builder.
func (m MachineBuilder) WithCertificatesExpiryDate(expiryDate metav1.Time) MachineBuilder {
	m.certificatesExpiryDate = expiryDate
	return m
}

// WithConditions sets the Conditions for the machine builder.
func (m MachineBuilder) WithConditions(conditions []metav1.Condition) MachineBuilder {
	m.conditions = conditions
	return m
}

// WithV1Beta1Conditions sets the Conditions for the machine builder.
func (m MachineBuilder) WithV1Beta1Conditions(conditions clusterv1.Conditions) MachineBuilder {
	m.v1Beta1Conditions = conditions
	return m
}

// WithFailureMessage sets the FailureMessage for the machine builder.
func (m MachineBuilder) WithFailureMessage(message *string) MachineBuilder {
	m.v1Beta1FailureMessage = message
	return m
}

// WithFailureReason sets the FailureReason for the machine builder.
func (m MachineBuilder) WithFailureReason(reason *capierrors.MachineStatusError) MachineBuilder {
	m.v1Beta1FailureReason = reason
	return m
}

// WithInfrastructureProvisioned sets the InfrastructureProvisioned for the machine builder.
func (m MachineBuilder) WithInfrastructureProvisioned(provisioned bool) MachineBuilder {
	m.infrastructureProvisioned = &provisioned
	return m
}

// WithLastUpdated sets the LastUpdated for the machine builder.
func (m MachineBuilder) WithLastUpdated(lastUpdated metav1.Time) MachineBuilder {
	m.lastUpdated = lastUpdated
	return m
}

// WithNodeInfo sets the NodeInfo for the machine builder.
func (m MachineBuilder) WithNodeInfo(nodeInfo *corev1.NodeSystemInfo) MachineBuilder {
	m.nodeInfo = nodeInfo
	return m
}

// WithObservedGeneration sets the ObservedGeneration for the machine builder.
func (m MachineBuilder) WithObservedGeneration(generation int64) MachineBuilder {
	m.observedGeneration = generation
	return m
}

// WithPhase sets the Phase for the machine builder.
func (m MachineBuilder) WithPhase(phase clusterv1.MachinePhase) MachineBuilder {
	m.phase = phase
	return m
}

// WithDeletion sets the Deletion status for the machine builder.
func (m MachineBuilder) WithDeletion(deletion *clusterv1.MachineDeletionStatus) MachineBuilder {
	m.deletion = deletion
	return m
}
