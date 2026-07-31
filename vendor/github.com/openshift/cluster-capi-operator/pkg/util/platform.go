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
package util

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metal3v1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	ibmpowervsv1 "sigs.k8s.io/cluster-api-provider-ibmcloud/api/v1beta2"
	openstackv1 "sigs.k8s.io/cluster-api-provider-openstack/api/v1beta1"
	vspherev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"

	configv1 "github.com/openshift/api/config/v1"
)

const (
	infrastructureResourceName = "cluster"
)

var (
	errNilInfrastructure = errors.New("error infrastructure is nil")
	errNoPlatformStatus  = errors.New("error getting PlatformStatus, field not set")

	// ErrUnsupportedPlatform is returned when the platform is not supported.
	ErrUnsupportedPlatform = errors.New("unsupported platform")
)

// GetPlatform returns the platform type and the infrastructure resource.
func GetPlatform(ctx context.Context, cl client.Reader) (configv1.PlatformType, *configv1.Infrastructure, error) {
	infra, err := GetInfra(ctx, cl)
	if err != nil {
		return "", nil, err
	}

	platform, err := GetPlatformFromInfra(infra)
	if err != nil {
		return "", nil, err
	}

	return platform, infra, nil
}

// GetPlatformFromInfra returns the platform type from the infrastructure resource.
func GetPlatformFromInfra(infra *configv1.Infrastructure) (configv1.PlatformType, error) {
	if infra == nil {
		return "", errNilInfrastructure
	}

	if infra.Status.PlatformStatus == nil {
		return "", errNoPlatformStatus
	}

	return infra.Status.PlatformStatus.Type, nil
}

// GetInfra returns the infrastructure resource.
func GetInfra(ctx context.Context, cl client.Reader) (*configv1.Infrastructure, error) {
	infra := &configv1.Infrastructure{}

	if err := cl.Get(ctx, client.ObjectKey{Name: infrastructureResourceName}, infra); err != nil {
		return nil, fmt.Errorf("failed to get infrastructure %q: %w", infra.Kind, err)
	}

	return infra, nil
}

// InfraTypes provides infrastructure object constructors for the current platform.
type InfraTypes interface {
	// Machine returns a new Machine object for the current platform.
	Machine() client.Object
	// Cluster returns a new Cluster object for the current platform.
	Cluster() client.Object
	// Template returns a new MachineTemplate object for the current platform.
	Template() client.Object
	// ClusterTemplate returns a new ClusterTemplate object for the current platform.
	ClusterTemplate() client.Object
}

// GetCAPITypesForInfrastructure returns the infrastructure objects for a given platform.
// Returns ErrUnsupportedPlatform for unsupported platforms.
func GetCAPITypesForInfrastructure(infra *configv1.Infrastructure) (InfraTypes, configv1.PlatformType, error) {
	platform, err := GetPlatformFromInfra(infra)
	if err != nil {
		return nil, "", err
	}

	switch platform {
	case configv1.AWSPlatformType:
		return newInfraTypes[
			*awsv1.AWSMachine, *awsv1.AWSCluster,
			*awsv1.AWSMachineTemplate, *awsv1.AWSClusterTemplate,
		](), platform, nil
	case configv1.GCPPlatformType:
		return newInfraTypes[
			*gcpv1.GCPMachine, *gcpv1.GCPCluster,
			*gcpv1.GCPMachineTemplate, *gcpv1.GCPClusterTemplate,
		](), platform, nil
	case configv1.AzurePlatformType:
		azureCloudEnvironment := getAzureCloudEnvironment(infra.Status.PlatformStatus)
		if azureCloudEnvironment == configv1.AzureStackCloud {
			klog.Infof("Detected Azure Cloud Environment %q on platform %q is not supported", azureCloudEnvironment, platform)
			return nil, platform, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, platform)
		}

		return newInfraTypes[
			*azurev1.AzureMachine, *azurev1.AzureCluster,
			*azurev1.AzureMachineTemplate, *azurev1.AzureClusterTemplate,
		](), platform, nil
	case configv1.PowerVSPlatformType:
		return newInfraTypes[
			*ibmpowervsv1.IBMPowerVSMachine, *ibmpowervsv1.IBMPowerVSCluster,
			*ibmpowervsv1.IBMPowerVSMachineTemplate, *ibmpowervsv1.IBMPowerVSClusterTemplate,
		](), platform, nil
	case configv1.VSpherePlatformType:
		return newInfraTypes[
			*vspherev1.VSphereMachine, *vspherev1.VSphereCluster,
			*vspherev1.VSphereMachineTemplate, *vspherev1.VSphereClusterTemplate,
		](), platform, nil
	case configv1.OpenStackPlatformType:
		return newInfraTypes[
			*openstackv1.OpenStackMachine, *openstackv1.OpenStackCluster,
			*openstackv1.OpenStackMachineTemplate, *openstackv1.OpenStackClusterTemplate,
		](), platform, nil
	case configv1.BareMetalPlatformType:
		return newInfraTypes[
			*metal3v1.Metal3Machine, *metal3v1.Metal3Cluster,
			*metal3v1.Metal3MachineTemplate, *metal3v1.Metal3ClusterTemplate,
		](), platform, nil
	default:
		return nil, platform, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, platform)
	}
}

// getAzureCloudEnvironment returns the current AzureCloudEnvironment.
func getAzureCloudEnvironment(ps *configv1.PlatformStatus) configv1.AzureCloudEnvironment {
	if ps == nil || ps.Azure == nil {
		return ""
	}

	return ps.Azure.CloudName
}

type clientObjectPtr[T any] interface {
	client.Object
	*T
}

type infraTypes[Mptr clientObjectPtr[M], Cptr clientObjectPtr[C], Tptr clientObjectPtr[T], CTptr clientObjectPtr[CT], M, C, T, CT any] struct{}

// Machine returns a new Machine object for the current platform.
func (_ infraTypes[Mptr, Cptr, Tptr, CTptr, M, C, T, CT]) Machine() client.Object {
	return Mptr(new(M))
}

// Cluster returns a new Cluster object for the current platform.
func (_ infraTypes[Mptr, Cptr, Tptr, CTptr, M, C, T, CT]) Cluster() client.Object {
	return Cptr(new(C))
}

// Template returns a new MachineTemplate object for the current platform.
func (_ infraTypes[Mptr, Cptr, Tptr, CTptr, M, C, T, CT]) Template() client.Object {
	return Tptr(new(T))
}

// ClusterTemplate returns a new ClusterTemplate object for the current platform.
func (_ infraTypes[Mptr, Cptr, Tptr, CTptr, M, C, T, CT]) ClusterTemplate() client.Object {
	return CTptr(new(CT))
}

func newInfraTypes[Mptr clientObjectPtr[M], Cptr clientObjectPtr[C], Tptr clientObjectPtr[T], CTptr clientObjectPtr[CT], M, C, T, CT any]() InfraTypes {
	return infraTypes[Mptr, Cptr, Tptr, CTptr, M, C, T, CT]{}
}
