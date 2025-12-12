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
	clusterName     string
	deleteionOrder  clusterv1.MachineSetDeletionOrder
	machineNaming   clusterv1.MachineNamingSpec
	minReadySeconds *int32
	replicas        *int32
	selector        metav1.LabelSelector
	template        clusterv1.MachineTemplateSpec

	// Status fields.
	availableReplicas           *int32
	conditions                  []metav1.Condition
	v1Beta1AvailableReplicas    int32
	v1Beta1Conditions           clusterv1.Conditions
	v1Beta1FailureMessage       *string
	v1Beta1FailureReason        *capierrors.MachineSetStatusError
	v1Beta1FullyLabeledReplicas int32
	observedGeneration          int64
	readyReplicas               *int32
	v1Beta1ReadyReplicas        int32
	statusReplicas              *int32
	statusSelector              string
	upToDateReplicas            *int32
}

// Build builds a new MachineSet based on the configuration provided.
func (m MachineSetBuilder) Build() *clusterv1.MachineSet {
	if m.minReadySeconds != nil {
		m.template.Spec.MinReadySeconds = m.minReadySeconds
	}

	machineSet := &clusterv1.MachineSet{
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
		Spec: clusterv1.MachineSetSpec{
			ClusterName: m.clusterName,
			Deletion: clusterv1.MachineSetDeletionSpec{
				Order: m.deleteionOrder,
			},
			MachineNaming: m.machineNaming,
			Replicas:      m.replicas,
			Selector:      m.selector,
			Template:      m.template,
		},
		Status: clusterv1.MachineSetStatus{
			AvailableReplicas: m.availableReplicas,
			Conditions:        m.conditions,
			Deprecated: &clusterv1.MachineSetDeprecatedStatus{
				V1Beta1: &clusterv1.MachineSetV1Beta1DeprecatedStatus{
					AvailableReplicas:    m.v1Beta1AvailableReplicas,
					Conditions:           m.v1Beta1Conditions,
					FailureMessage:       m.v1Beta1FailureMessage,
					FailureReason:        m.v1Beta1FailureReason,
					FullyLabeledReplicas: m.v1Beta1FullyLabeledReplicas,
					ReadyReplicas:        m.v1Beta1ReadyReplicas,
				},
			},
			ObservedGeneration: m.observedGeneration,
			ReadyReplicas:      m.readyReplicas,
			Replicas:           m.statusReplicas,
			Selector:           m.statusSelector,
			UpToDateReplicas:   m.upToDateReplicas,
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

// WithDeletionOrder sets the deletionOrder for the MachineSet builder.
func (m MachineSetBuilder) WithDeletionOrder(deletionOrder clusterv1.MachineSetDeletionOrder) MachineSetBuilder {
	m.deleteionOrder = deletionOrder
	return m
}

// WithMachineNaming sets the machineNaming for the MachineSet builder.
func (m MachineSetBuilder) WithMachineNaming(machineNaming clusterv1.MachineNamingSpec) MachineSetBuilder {
	m.machineNaming = machineNaming
	return m
}

// WithMinReadySeconds sets the minReadySeconds for the MachineSet builder.
func (m MachineSetBuilder) WithMinReadySeconds(minReadySeconds int32) MachineSetBuilder {
	m.minReadySeconds = &minReadySeconds
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
func (m MachineSetBuilder) WithTemplate(template clusterv1.MachineTemplateSpec) MachineSetBuilder {
	m.template = template
	return m
}

// Status.

// WithStatusConditions sets the status conditions for the MachineSet builder.
func (m MachineSetBuilder) WithStatusConditions(conditions []metav1.Condition) MachineSetBuilder {
	m.conditions = conditions
	return m
}

// WithStatusV1Beta1AvailableReplicas sets the status v1beta1 availableReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1AvailableReplicas(availableReplicas int32) MachineSetBuilder {
	m.v1Beta1AvailableReplicas = availableReplicas
	return m
}

// WithStatusV1Beta1Conditions sets the status v1beta1 conditions for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1Conditions(conditions clusterv1.Conditions) MachineSetBuilder {
	m.v1Beta1Conditions = conditions
	return m
}

// WithStatusV1Beta1FailureMessage sets the status v1beta1 failureMessage for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1FailureMessage(failureMessage string) MachineSetBuilder {
	m.v1Beta1FailureMessage = &failureMessage
	return m
}

// WithStatusV1Beta1FailureReason sets the status v1beta1 failureReason for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1FailureReason(failureReason capierrors.MachineSetStatusError) MachineSetBuilder {
	m.v1Beta1FailureReason = &failureReason
	return m
}

// WithStatusV1Beta1FullyLabeledReplicas sets the status v1beta1 fullyLabeledReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1FullyLabeledReplicas(fullyLabeledReplicas int32) MachineSetBuilder {
	m.v1Beta1FullyLabeledReplicas = fullyLabeledReplicas
	return m
}

// WithStatusObservedGeneration sets the status observedGeneration for the MachineSet builder.
func (m MachineSetBuilder) WithStatusObservedGeneration(observedGeneration int64) MachineSetBuilder {
	m.observedGeneration = observedGeneration
	return m
}

// WithStatusV1Beta1ReadyReplicas sets the status v1beta1 readyReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusV1Beta1ReadyReplicas(readyReplicas int32) MachineSetBuilder {
	m.v1Beta1ReadyReplicas = readyReplicas
	return m
}

// WithStatusReplicas sets the status replicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusReplicas(replicas int32) MachineSetBuilder {
	m.statusReplicas = &replicas
	return m
}

// WithStatusSelector sets the status selector for the MachineSet builder.
func (m MachineSetBuilder) WithStatusSelector(selector string) MachineSetBuilder {
	m.statusSelector = selector
	return m
}

// WithStatusReadyReplicas sets the status readyReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusReadyReplicas(readyReplicas int32) MachineSetBuilder {
	m.readyReplicas = &readyReplicas
	return m
}

// WithStatusAvailableReplicas sets the status availableReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusAvailableReplicas(availableReplicas int32) MachineSetBuilder {
	m.availableReplicas = &availableReplicas
	return m
}

// WithStatusUpToDateReplicas sets the status upToDateReplicas for the MachineSet builder.
func (m MachineSetBuilder) WithStatusUpToDateReplicas(upToDateReplicas int32) MachineSetBuilder {
	m.upToDateReplicas = &upToDateReplicas
	return m
}
