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

package v1

import (
	"encoding/json"

	machinev1 "github.com/openshift/api/machine/v1"
	"github.com/openshift/cluster-api-actuator-pkg/testutils/resourcebuilder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var (
	defaultCredentialsSecretName       = "powervs-cloud-credentials"
	defaultKeyPairName                 = "powervs-key-12345678"
	defaultMemory                int32 = 32
	defaultProcessors                  = intstr.FromString("1")
	defaultProcessorType               = machinev1.PowerVSProcessorTypeShared
	defaultSystemType                  = "s922"
	defaultUserDataSecretName          = "powervs-user-data-12345678"
	defaultLoadBalancer                = []machinev1.LoadBalancerReference{
		{
			Name: "default-lbName",
			Type: machinev1.ApplicationLoadBalancerType,
		},
	}
	defaultImage = machinev1.PowerVSResource{
		Type: machinev1.PowerVSResourceTypeID,
		ID:   ptr.To("default-imageID"),
	}
	defaultNetwork = machinev1.PowerVSResource{
		Type: machinev1.PowerVSResourceTypeID,
		ID:   ptr.To("default-networkID"),
	}
	defaultServiceInstance = machinev1.PowerVSResource{
		Type: machinev1.PowerVSResourceTypeID,
		ID:   ptr.To("default-serviceInstanceID"),
	}
)

// PowerVSMachineProviderConfigBuilder is used to build a PowerVSMachineProviderConfig.
type PowerVSMachineProviderConfigBuilder struct {
	credentialsSecret **machinev1.PowerVSSecretReference
	image             *machinev1.PowerVSResource
	keyPairName       *string
	loadBalancers     *[]machinev1.LoadBalancerReference
	memoryGIB         *int32
	network           *machinev1.PowerVSResource
	processors        *intstr.IntOrString
	processorType     *machinev1.PowerVSProcessorType
	serviceInstance   *machinev1.PowerVSResource
	systemType        *string
	userDataSecret    **machinev1.PowerVSSecretReference
}

// PowerVSProviderSpec creates a new PowerVS machine config builder.
func PowerVSProviderSpec() PowerVSMachineProviderConfigBuilder {
	return PowerVSMachineProviderConfigBuilder{}
}

// Build returns the generated NutanixMachineProviderConfig.
func (p PowerVSMachineProviderConfigBuilder) Build() machinev1.PowerVSMachineProviderConfig {
	return machinev1.PowerVSMachineProviderConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PowerVSMachineProviderConfig",
			APIVersion: machinev1.GroupVersion.String(),
		},
		CredentialsSecret: resourcebuilder.Coalesce(p.credentialsSecret, &machinev1.PowerVSSecretReference{Name: defaultCredentialsSecretName}),
		Image:             resourcebuilder.Coalesce(p.image, defaultImage),
		KeyPairName:       resourcebuilder.Coalesce(p.keyPairName, defaultKeyPairName),
		LoadBalancers:     coalesceLoadBalancers(p.loadBalancers, defaultLoadBalancer),
		MemoryGiB:         resourcebuilder.Coalesce(p.memoryGIB, defaultMemory),
		Network:           resourcebuilder.Coalesce(p.network, defaultNetwork),
		Processors:        resourcebuilder.Coalesce(p.processors, defaultProcessors),
		ProcessorType:     resourcebuilder.Coalesce(p.processorType, defaultProcessorType),
		ServiceInstance:   resourcebuilder.Coalesce(p.serviceInstance, defaultServiceInstance),
		SystemType:        resourcebuilder.Coalesce(p.systemType, defaultSystemType),
		UserDataSecret:    resourcebuilder.Coalesce(p.userDataSecret, &machinev1.PowerVSSecretReference{Name: defaultUserDataSecretName}),
	}
}

// BuildRawExtension builds a new PowerVS machine config based on the configuration provided.
func (p PowerVSMachineProviderConfigBuilder) BuildRawExtension() *runtime.RawExtension {
	providerConfig := p.Build()

	raw, err := json.Marshal(providerConfig)
	if err != nil {
		// As we are building the input to json.Marshal, this should never happen.
		panic(err)
	}

	return &runtime.RawExtension{
		Raw: raw,
	}
}

// WithCredentialSecret sets the credentialsSecret for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithCredentialSecret(credentialSecret *machinev1.PowerVSSecretReference) PowerVSMachineProviderConfigBuilder {
	p.credentialsSecret = &credentialSecret
	return p
}

// WithImage sets the image for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithImage(image machinev1.PowerVSResource) PowerVSMachineProviderConfigBuilder {
	p.image = &image
	return p
}

// WithKeyPairName sets the keyPairName for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithKeyPairName(keyPairName string) PowerVSMachineProviderConfigBuilder {
	p.keyPairName = &keyPairName
	return p
}

// WithLoadBalancers sets the processorType for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithLoadBalancers(loadBalancers []machinev1.LoadBalancerReference) PowerVSMachineProviderConfigBuilder {
	p.loadBalancers = &loadBalancers
	return p
}

// WithMemoryGIB sets the processorType for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithMemoryGIB(memoryGIB int32) PowerVSMachineProviderConfigBuilder {
	p.memoryGIB = &memoryGIB
	return p
}

// WithNetwork sets the serviceInstance for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithNetwork(network machinev1.PowerVSResource) PowerVSMachineProviderConfigBuilder {
	p.network = &network
	return p
}

// WithProcessors sets the processors for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithProcessors(processors intstr.IntOrString) PowerVSMachineProviderConfigBuilder {
	p.processors = &processors
	return p
}

// WithProcessorType sets the processorType for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithProcessorType(processorType machinev1.PowerVSProcessorType) PowerVSMachineProviderConfigBuilder {
	p.processorType = &processorType
	return p
}

// WithServiceInstance sets the serviceInstance for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithServiceInstance(serviceInstance machinev1.PowerVSResource) PowerVSMachineProviderConfigBuilder {
	p.serviceInstance = &serviceInstance
	return p
}

// WithSystemType sets the systemType for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithSystemType(systemType string) PowerVSMachineProviderConfigBuilder {
	p.systemType = &systemType
	return p
}

// WithUserDataSecret sets the userDataSecret for the PowerVS machine config builder.
func (p PowerVSMachineProviderConfigBuilder) WithUserDataSecret(userDataSecret *machinev1.PowerVSSecretReference) PowerVSMachineProviderConfigBuilder {
	p.userDataSecret = &userDataSecret
	return p
}

func coalesceLoadBalancers(v1 *[]machinev1.LoadBalancerReference, v2 []machinev1.LoadBalancerReference) []machinev1.LoadBalancerReference {
	if v1 == nil {
		return v2
	}

	return *v1
}
