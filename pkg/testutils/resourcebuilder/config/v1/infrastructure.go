/*
Copyright 2022 Red Hat, Inc.

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
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Infrastructure creates a new infrastructure builder.
func Infrastructure() InfrastructureBuilder {
	return InfrastructureBuilder{}
}

// InfrastructureBuilder is used to build out an infrastructure object.
type InfrastructureBuilder struct {
	generateName string
	name         string
	namespace    string
	labels       map[string]string
	spec         *configv1.InfrastructureSpec
	status       *configv1.InfrastructureStatus
}

// Build builds a new infrastructure object based on the configuration provided.
func (i InfrastructureBuilder) Build() *configv1.Infrastructure {
	infra := &configv1.Infrastructure{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: i.generateName,
			Name:         i.name,
			Namespace:    i.namespace,
			Labels:       i.labels,
		},
	}

	if i.spec != nil {
		infra.Spec = *i.spec
	}

	if i.status != nil {
		infra.Status = *i.status
	}

	return infra
}

// AsAWS sets the Status for the infrastructure builder.
func (i InfrastructureBuilder) AsAWS(name string, region string) InfrastructureBuilder {
	i.spec = &configv1.InfrastructureSpec{
		PlatformSpec: configv1.PlatformSpec{
			Type: "AWS",
			AWS:  &configv1.AWSPlatformSpec{},
		},
	}
	i.status = &configv1.InfrastructureStatus{
		InfrastructureName:     name,
		APIServerURL:           "https://api.test-cluster.test-domain:6443",
		APIServerInternalURL:   "https://api-int.test-cluster.test-domain:6443",
		EtcdDiscoveryDomain:    "",
		ControlPlaneTopology:   configv1.HighlyAvailableTopologyMode,
		InfrastructureTopology: configv1.HighlyAvailableTopologyMode,
		PlatformStatus: &configv1.PlatformStatus{
			Type: "AWS",
			AWS: &configv1.AWSPlatformStatus{
				Region: region,
			},
		},
	}

	return i
}

// AsAzure sets the Status for the infrastructure builder.
func (i InfrastructureBuilder) AsAzure(name string) InfrastructureBuilder {
	i.spec = &configv1.InfrastructureSpec{
		PlatformSpec: configv1.PlatformSpec{
			Type:  "Azure",
			Azure: &configv1.AzurePlatformSpec{},
		},
	}
	i.status = &configv1.InfrastructureStatus{
		InfrastructureName:     name,
		APIServerURL:           "https://api.test-cluster.test-domain:6443",
		APIServerInternalURL:   "https://api-int.test-cluster.test-domain:6443",
		EtcdDiscoveryDomain:    "",
		ControlPlaneTopology:   configv1.HighlyAvailableTopologyMode,
		InfrastructureTopology: configv1.HighlyAvailableTopologyMode,
		PlatformStatus: &configv1.PlatformStatus{
			Type:  "Azure",
			Azure: &configv1.AzurePlatformStatus{},
		},
	}

	return i
}

// AsGCP sets the Status for the infrastructure builder.
func (i InfrastructureBuilder) AsGCP(name string, region string) InfrastructureBuilder {
	i.spec = &configv1.InfrastructureSpec{
		PlatformSpec: configv1.PlatformSpec{
			Type: configv1.GCPPlatformType,
			GCP:  &configv1.GCPPlatformSpec{},
		},
	}
	i.status = &configv1.InfrastructureStatus{
		InfrastructureName:     name,
		APIServerURL:           "https://api.test-cluster.test-domain:6443",
		APIServerInternalURL:   "https://api-int.test-cluster.test-domain:6443",
		EtcdDiscoveryDomain:    "",
		ControlPlaneTopology:   configv1.HighlyAvailableTopologyMode,
		InfrastructureTopology: configv1.HighlyAvailableTopologyMode,
		PlatformStatus: &configv1.PlatformStatus{
			Type: configv1.GCPPlatformType,
			GCP: &configv1.GCPPlatformStatus{
				Region: region,
			},
		},
	}

	return i
}

// WithGenerateName sets the generateName for the infrastructure builder.
func (i InfrastructureBuilder) WithGenerateName(generateName string) InfrastructureBuilder {
	i.generateName = generateName
	return i
}

// WithLabel sets the labels for the infrastructure builder.
func (i InfrastructureBuilder) WithLabel(key, value string) InfrastructureBuilder {
	if i.labels == nil {
		i.labels = make(map[string]string)
	}

	i.labels[key] = value

	return i
}

// WithLabels sets the labels for the infrastructure builder.
func (i InfrastructureBuilder) WithLabels(labels map[string]string) InfrastructureBuilder {
	i.labels = labels
	return i
}

// WithName sets the name for the infrastructure builder.
func (i InfrastructureBuilder) WithName(name string) InfrastructureBuilder {
	i.name = name
	return i
}

// WithNamespace sets the namespace for the infrastructure builder.
func (i InfrastructureBuilder) WithNamespace(namespace string) InfrastructureBuilder {
	i.namespace = namespace
	return i
}
