/*
Copyright 2023 Red Hat, Inc.

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

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DaemonSet creates a new DaemonSet builder.
func DaemonSet() DaemonSetBuilder {
	return DaemonSetBuilder{}
}

// DaemonSetBuilder is used to build out a DaemonSet object.
type DaemonSetBuilder struct {
	generateName string
	name         string
	namespace    string
	labels       map[string]string
	volumes      []corev1.Volume
	containers   []corev1.Container
}

// Build builds a new DaemonSet based on the configuration provided.
func (m DaemonSetBuilder) Build() *appsv1.DaemonSet {
	DaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: m.generateName,
			Name:         m.name,
			Namespace:    m.namespace,
			Labels:       m.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      m.name,
					Namespace: m.namespace,
					Labels:    m.labels,
				},
				Spec: corev1.PodSpec{
					Volumes:    m.volumes,
					Containers: m.containers,
				},
			},
		},
	}

	return DaemonSet
}

// WithContainers sets the containers for the DaemonSet builder.
func (m DaemonSetBuilder) WithContainers(containers []corev1.Container) DaemonSetBuilder {
	m.containers = containers
	return m
}

// WithGenerateName sets the generateName for the DaemonSet builder.
func (m DaemonSetBuilder) WithGenerateName(generateName string) DaemonSetBuilder {
	m.generateName = generateName
	return m
}

// WithLabel sets the labels for the DaemonSet builder.
func (m DaemonSetBuilder) WithLabel(key, value string) DaemonSetBuilder {
	if m.labels == nil {
		m.labels = make(map[string]string)
	}

	m.labels[key] = value

	return m
}

// WithLabels sets the labels for the DaemonSet builder.
func (m DaemonSetBuilder) WithLabels(labels map[string]string) DaemonSetBuilder {
	m.labels = labels
	return m
}

// WithName sets the name for the DaemonSet builder.
func (m DaemonSetBuilder) WithName(name string) DaemonSetBuilder {
	m.name = name
	return m
}

// WithNamespace sets the namespace for the DaemonSet builder.
func (m DaemonSetBuilder) WithNamespace(namespace string) DaemonSetBuilder {
	m.namespace = namespace
	return m
}

// WithVolumes sets the volumes for the DaemonSet builder.
func (m DaemonSetBuilder) WithVolumes(volumes []corev1.Volume) DaemonSetBuilder {
	m.volumes = volumes
	return m
}
