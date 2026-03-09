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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"

	//nolint:staticcheck // Ignore SA1019 (deprecation) until v1beta2.
	capierrors "sigs.k8s.io/cluster-api/errors"
)

// MachineSet creates a new MachineSet builder.
func MachineSet() MachineSetBuilder {
	return MachineSetBuilder{}
}

// MachineSetBuilder is used to build out a MachineSet object.
type MachineSetBuilder struct {
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
	clusterName           string
	deletePolicy          string
	machineNamingStrategy *clusterv1beta1.MachineNamingStrategy
	minReadySeconds       int32
	replicas              *int32
	selector              metav1.LabelSelector
	template              clusterv1beta1.MachineTemplateSpec

	// Status fields.
	availableReplicas    int32
	conditions           clusterv1beta1.Conditions
	failureMessage       *string
	failureReason        *capierrors.MachineSetStatusError
	fullyLabeledReplicas int32
	observedGeneration   int64
	readyReplicas        int32
	statusReplicas       int32
	statusSelector       string
	v1Beta2              *clusterv1beta1.MachineSetV1Beta2Status
}

// Build builds a new MachineSet based on the configuration provided.
func (m MachineSetBuilder) Build() *clusterv1beta1.MachineSet {
	machineSet := &clusterv1beta1.MachineSet{
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
		Spec: clusterv1beta1.MachineSetSpec{
			ClusterName:           m.clusterName,
			DeletePolicy:          m.deletePolicy,
			MachineNamingStrategy: m.machineNamingStrategy,
			MinReadySeconds:       m.minReadySeconds,
			Replicas:              m.replicas,
			Selector:              m.selector,
			Template:              m.template,
		},
		Status: clusterv1beta1.MachineSetStatus{
			AvailableReplicas:    m.availableReplicas,
			Conditions:           m.conditions,
			FailureMessage:       m.failureMessage,
			FailureReason:        m.failureReason,
			FullyLabeledReplicas: m.fullyLabeledReplicas,
			ObservedGeneration:   m.observedGeneration,
			ReadyReplicas:        m.readyReplicas,
			Replicas:             m.statusReplicas,
			Selector:             m.statusSelector,
			V1Beta2:              m.v1Beta2,
		},
	}

	return machineSet
}

// Object meta fields.

// WithAnnotations sets the annotations for the MachineSet builder.
func (m MachineSetBuilder) WithAnnotations(annotations map[string]string) MachineSetBuilder {
	m.annotations = annotations
	return m
}

// WithCreationTimestamp sets the creationTimestamp for the MachineSet builder.
func (m MachineSetBuilder) WithCreationTimestamp(timestamp metav1.Time) MachineSetBuilder {
	m.creationTimestamp = timestamp
	return m
}

// WithDeletionTimestamp sets the deletionTimestamp for the MachineSet builder.
func (m MachineSetBuilder) WithDeletionTimestamp(timestamp *metav1.Time) MachineSetBuilder {
	m.deletionTimestamp = timestamp
	return m
}

// WithGenerateName sets the generateName for the MachineSet builder.
func (m MachineSetBuilder) WithGenerateName(generateName string) MachineSetBuilder {
	m.generateName = generateName
	return m
}

// WithLabels sets the labels for the MachineSet builder.
func (m MachineSetBuilder) WithLabels(labels map[string]string) MachineSetBuilder {
	m.labels = labels
	return m
}

// WithName sets the name for the MachineSet builder.
func (m MachineSetBuilder) WithName(name string) MachineSetBuilder {
	m.name = name
	return m
}

// WithNamespace sets the namespace for the MachineSet builder.
func (m MachineSetBuilder) WithNamespace(namespace string) MachineSetBuilder {
	m.namespace = namespace
	return m
}

// WithOwnerReferences sets the OwnerReferences for the machine builder.
func (m MachineSetBuilder) WithOwnerReferences(ownerRefs []metav1.OwnerReference) MachineSetBuilder {
	m.ownerReferences = ownerRefs
	return m
}

// Spec fields.

// WithClusterName sets the clusterName for the MachineSet builder.
func (m MachineSetBuilder) WithClusterName(clusterName string) MachineSetBuilder {
	m.clusterName = clusterName
	return m
}

// WithDeletePolicy sets the deletePolicy for the MachineSet builder.
func (m MachineSetBuilder) WithDeletePolicy(deletePolicy string) MachineSetBuilder {
	m.deletePolicy = deletePolicy
	return m
}

// WithMachineNamingStrategy sets the machineNamingStrategy for the MachineSet builder.
func (m MachineSetBuilder) WithMachineNamingStrategy(strategy *clusterv1beta1.MachineNamingStrategy) MachineSetBuilder {
	m.machineNamingStrategy = strategy
	return m
}

// WithMinReadySeconds sets the minReadySeconds for the MachineSet builder.
func (m MachineSetBuilder) WithMinReadySeconds(minReadySeconds int32) MachineSetBuilder {
	m.minReadySeconds = minReadySeconds
	return m
}

// WithReplicas sets the replicas for the MachineSet builder.
func (m MachineSetBuilder) WithReplicas(replicas int32) MachineSetBuilder {
	m.replicas = &replicas
	return m
}

// WithSelector sets the selector for the MachineSet builder.
func (m MachineSetBuilder) WithSelector(selector metav1.LabelSelector) MachineSetBuilder {
	m.selector = selector
	return m
}

// WithTemplate sets the template for the MachineSet builder.
func (m MachineSetBuilder) WithTemplate(template clusterv1beta1.MachineTemplateSpec) MachineSetBuilder {
	m.template = template
	return m
}

// Status.

// WithStatusAvailableReplicas sets the status availableReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusAvailableReplicas(availableReplicas int32) MachineSetBuilder {
	m.availableReplicas = availableReplicas
	return m
}

// WithStatusConditions sets the status conditions for the MachineSet builder.
func (m MachineSetBuilder) WithStatusConditions(conditions clusterv1beta1.Conditions) MachineSetBuilder {
	m.conditions = conditions
	return m
}

// WithStatusFailureMessage sets the status failureMessage for the MachineSet builder.
func (m MachineSetBuilder) WithStatusFailureMessage(failureMessage string) MachineSetBuilder {
	m.failureMessage = &failureMessage
	return m
}

// WithStatusFailureReason sets the status failureReason for the MachineSet builder.
func (m MachineSetBuilder) WithStatusFailureReason(failureReason capierrors.MachineSetStatusError) MachineSetBuilder {
	m.failureReason = &failureReason
	return m
}

// WithStatusFullyLabeledReplicas sets the status fullyLabeledReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusFullyLabeledReplicas(fullyLabeledReplicas int32) MachineSetBuilder {
	m.fullyLabeledReplicas = fullyLabeledReplicas
	return m
}

// WithStatusObservedGeneration sets the status observedGeneration for the MachineSet builder.
func (m MachineSetBuilder) WithStatusObservedGeneration(observedGeneration int64) MachineSetBuilder {
	m.observedGeneration = observedGeneration
	return m
}

// WithStatusReadyReplicas sets the status readyReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusReadyReplicas(readyReplicas int32) MachineSetBuilder {
	m.readyReplicas = readyReplicas
	return m
}

// WithStatusReplicas sets the status replicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusReplicas(replicas int32) MachineSetBuilder {
	m.statusReplicas = replicas
	return m
}

// WithStatusSelector sets the status selector for the MachineSet builder.
func (m MachineSetBuilder) WithStatusSelector(selector string) MachineSetBuilder {
	m.statusSelector = selector
	return m
}

// WithV1Beta2Status sets the v1beta2 status for the MachineSet builder.
func (m MachineSetBuilder) WithV1Beta2Status(v1Beta2 *clusterv1beta1.MachineSetV1Beta2Status) MachineSetBuilder {
	m.v1Beta2 = v1Beta2
	return m
}
