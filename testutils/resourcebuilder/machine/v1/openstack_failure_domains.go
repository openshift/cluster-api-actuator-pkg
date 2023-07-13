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

// Linter warns about duplicated code with OpenStack, GCP, and OpenStack FailureDomains builders.
// While the builders are almost identical, we need to keep them separate because they build different objects.
//
//nolint:dupl
package v1

import (
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1"
)

// OpenStackFailureDomains creates a new failure domains builder for OpenStack.
func OpenStackFailureDomains() OpenStackFailureDomainsBuilder {
	return OpenStackFailureDomainsBuilder{[]OpenStackFailureDomainBuilder{
		OpenStackFailureDomain().WithComputeAvailabilityZone("nova-az0").
			WithRootVolume(&machinev1.RootVolume{
				AvailabilityZone: "cinder-az0",
				VolumeType:       "fast-az0",
			}),
		OpenStackFailureDomain().WithComputeAvailabilityZone("nova-az1").
			WithRootVolume(&machinev1.RootVolume{
				AvailabilityZone: "cinder-az1",
				VolumeType:       "fast-az1",
			}),
		OpenStackFailureDomain().WithComputeAvailabilityZone("nova-az2").
			WithRootVolume(&machinev1.RootVolume{
				AvailabilityZone: "cinder-az2",
				VolumeType:       "fast-az2",
			}),
	}}
}

// OpenStackFailureDomainsBuilder is used to build a failuredomains.
type OpenStackFailureDomainsBuilder struct {
	failureDomainsBuilders []OpenStackFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (a OpenStackFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform:  configv1.OpenStackPlatformType,
		OpenStack: []machinev1.OpenStackFailureDomain{},
	}

	for _, builder := range a.failureDomainsBuilders {
		fds.OpenStack = append(fds.OpenStack, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (a OpenStackFailureDomainsBuilder) WithFailureDomainBuilder(fdbuilder OpenStackFailureDomainBuilder) OpenStackFailureDomainsBuilder {
	a.failureDomainsBuilders = append(a.failureDomainsBuilders, fdbuilder)
	return a
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (a OpenStackFailureDomainsBuilder) WithFailureDomainBuilders(fdbuilders ...OpenStackFailureDomainBuilder) OpenStackFailureDomainsBuilder {
	a.failureDomainsBuilders = fdbuilders
	return a
}

// OpenStackFailureDomain creates a new OpenStack failure domain builder for OpenStack.
func OpenStackFailureDomain() OpenStackFailureDomainBuilder {
	return OpenStackFailureDomainBuilder{}
}

// OpenStackFailureDomainBuilder is used to build an OpenStack failuredomain.
type OpenStackFailureDomainBuilder struct {
	AvailabilityZone string
	RootVolume       *machinev1.RootVolume
}

// Build builds a OpenStack failuredomain from the configuration.
func (a OpenStackFailureDomainBuilder) Build() machinev1.OpenStackFailureDomain {
	return machinev1.OpenStackFailureDomain{
		AvailabilityZone: a.AvailabilityZone,
		RootVolume:       a.RootVolume,
	}
}

// WithComputeAvailabilityZone sets the nova availability zone for the OpenStack failuredomain builder.
func (a OpenStackFailureDomainBuilder) WithComputeAvailabilityZone(zone string) OpenStackFailureDomainBuilder {
	a.AvailabilityZone = zone
	return a
}

// WithRootVolume sets the root volume for the OpenStack failuredomain builder.
func (a OpenStackFailureDomainBuilder) WithRootVolume(rootVolume *machinev1.RootVolume) OpenStackFailureDomainBuilder {
	a.RootVolume = rootVolume
	return a
}
