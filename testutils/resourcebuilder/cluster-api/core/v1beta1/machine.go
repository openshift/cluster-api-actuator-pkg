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
	bootstrap               capiv1.Bootstrap
	clusterName             string
	failureDomain           *string
	infrastructureRef       corev1.ObjectReference
	nodeDeletionTimeout     *metav1.Duration
	nodeDrainTimeout        *metav1.Duration
	nodeVolumeDetachTimeout *metav1.Duration
	providerID              *string
	version                 *string

	// Status fields.
	addresses              capiv1.MachineAddresses
	bootstrapReady         bool
	certificatesExpiryDate *metav1.Time
	conditions             capiv1.Conditions
	failureMessage         *string
	failureReason          *capierrors.MachineStatusError
	infrastructureReady    bool
	lastUpdated            *metav1.Time
	nodeInfo               *corev1.NodeSystemInfo
	nodeRef                *corev1.ObjectReference
	observedGeneration     int64
	phase                  capiv1.MachinePhase
}

// Build builds a new Machine based on the configuration provided.
func (m MachineBuilder) Build() *capiv1.Machine {
	machine := &capiv1.Machine{
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
		Spec: capiv1.MachineSpec{
			Bootstrap:               m.bootstrap,
			ClusterName:             m.clusterName,
			FailureDomain:           m.failureDomain,
			InfrastructureRef:       m.infrastructureRef,
			NodeDeletionTimeout:     m.nodeDeletionTimeout,
			NodeDrainTimeout:        m.nodeDrainTimeout,
			NodeVolumeDetachTimeout: m.nodeVolumeDetachTimeout,
			ProviderID:              m.providerID,
			Version:                 m.version,
		},
		Status: capiv1.MachineStatus{
			Addresses:              m.addresses,
			BootstrapReady:         m.bootstrapReady,
			CertificatesExpiryDate: m.certificatesExpiryDate,
			Conditions:             m.conditions,
			FailureMessage:         m.failureMessage,
			FailureReason:          m.failureReason,
			InfrastructureReady:    m.infrastructureReady,
			LastUpdated:            m.lastUpdated,
			NodeInfo:               m.nodeInfo,
			NodeRef:                m.nodeRef,
			ObservedGeneration:     m.observedGeneration,
			Phase:                  string(m.phase),
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
func (m MachineBuilder) WithBootstrap(bootstrap capiv1.Bootstrap) MachineBuilder {
	m.bootstrap = bootstrap
	return m
}

// WithClusterName sets the ClusterName for the machine builder.
func (m MachineBuilder) WithClusterName(clusterName string) MachineBuilder {
	m.clusterName = clusterName
	return m
}

// WithFailureDomain sets the FailureDomain for the machine builder.
func (m MachineBuilder) WithFailureDomain(failureDomain *string) MachineBuilder {
	m.failureDomain = failureDomain
	return m
}

// WithInfrastructureRef sets the InfrastructureRef for the machine builder.
func (m MachineBuilder) WithInfrastructureRef(infraRef corev1.ObjectReference) MachineBuilder {
	m.infrastructureRef = infraRef
	return m
}

// WithNodeDeletionTimeout sets the NodeDeletionTimeout for the machine builder.
func (m MachineBuilder) WithNodeDeletionTimeout(timeout *metav1.Duration) MachineBuilder {
	m.nodeDeletionTimeout = timeout
	return m
}

// WithNodeDrainTimeout sets the NodeDrainTimeout for the machine builder.
func (m MachineBuilder) WithNodeDrainTimeout(timeout *metav1.Duration) MachineBuilder {
	m.nodeDrainTimeout = timeout
	return m
}

// WithNodeVolumeDetachTimeout sets the NodeVolumeDetachTimeout for the machine builder.
func (m MachineBuilder) WithNodeVolumeDetachTimeout(timeout *metav1.Duration) MachineBuilder {
	m.nodeVolumeDetachTimeout = timeout
	return m
}

// WithNodeRef sets the NodeRef for the machine builder.
func (m MachineBuilder) WithNodeRef(nodeRef *corev1.ObjectReference) MachineBuilder {
	m.nodeRef = nodeRef
	return m
}

// WithProviderID sets the ProviderID for the machine builder.
func (m MachineBuilder) WithProviderID(providerID *string) MachineBuilder {
	m.providerID = providerID
	return m
}

// WithVersion sets the Version for the machine builder.
func (m MachineBuilder) WithVersion(version *string) MachineBuilder {
	m.version = version
	return m
}

// Status Fields.

// WithAddresses sets the Addresses for the machine builder.
func (m MachineBuilder) WithAddresses(addresses capiv1.MachineAddresses) MachineBuilder {
	m.addresses = addresses
	return m
}

// WithBootstrapReady sets the BootstrapReady for the machine builder.
func (m MachineBuilder) WithBootstrapReady(ready bool) MachineBuilder {
	m.bootstrapReady = ready
	return m
}

// WithCertificatesExpiryDate sets the CertificatesExpiryDate for the machine builder.
func (m MachineBuilder) WithCertificatesExpiryDate(expiryDate *metav1.Time) MachineBuilder {
	m.certificatesExpiryDate = expiryDate
	return m
}

// WithConditions sets the Conditions for the machine builder.
func (m MachineBuilder) WithConditions(conditions capiv1.Conditions) MachineBuilder {
	m.conditions = conditions
	return m
}

// WithFailureMessage sets the FailureMessage for the machine builder.
func (m MachineBuilder) WithFailureMessage(message *string) MachineBuilder {
	m.failureMessage = message
	return m
}

// WithFailureReason sets the FailureReason for the machine builder.
func (m MachineBuilder) WithFailureReason(reason *capierrors.MachineStatusError) MachineBuilder {
	m.failureReason = reason
	return m
}

// WithInfrastructureReady sets the InfrastructureReady for the machine builder.
func (m MachineBuilder) WithInfrastructureReady(ready bool) MachineBuilder {
	m.infrastructureReady = ready
	return m
}

// WithLastUpdated sets the LastUpdated for the machine builder.
func (m MachineBuilder) WithLastUpdated(lastUpdated *metav1.Time) MachineBuilder {
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
func (m MachineBuilder) WithPhase(phase capiv1.MachinePhase) MachineBuilder {
	m.phase = phase
	return m
}
