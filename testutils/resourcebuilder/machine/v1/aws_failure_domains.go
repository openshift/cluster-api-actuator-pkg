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
	machinev1 "github.com/openshift/api/machine/v1"
	"k8s.io/utils/ptr"
)

// AWSFailureDomains creates a new failure domains builder for AWS.
func AWSFailureDomains() AWSFailureDomainsBuilder {
	return AWSFailureDomainsBuilder{
		failureDomainsBuilders: []AWSFailureDomainBuilder{
			AWSFailureDomain().WithAvailabilityZone("us-east-1a").
				WithSubnet(machinev1.AWSResourceReference{
					Type: machinev1.AWSIDReferenceType,
					ID:   ptr.To[string]("subenet-us-east-1a"),
				}),
			AWSFailureDomain().WithAvailabilityZone("us-east-1b").
				WithSubnet(machinev1.AWSResourceReference{
					Type: machinev1.AWSIDReferenceType,
					ID:   ptr.To[string]("subenet-us-east-1b"),
				}),
			AWSFailureDomain().WithAvailabilityZone("us-east-1c").
				WithSubnet(machinev1.AWSResourceReference{
					Type: machinev1.AWSIDReferenceType,
					ID:   ptr.To[string]("subenet-us-east-1c"),
				}),
		},
	}
}

// AWSFailureDomainsBuilder is used to build a failuredomains.
type AWSFailureDomainsBuilder struct {
	failureDomainsBuilders []AWSFailureDomainBuilder
}

// BuildFailureDomains builds a failuredomains from the configuration.
func (a AWSFailureDomainsBuilder) BuildFailureDomains() machinev1.FailureDomains {
	fds := machinev1.FailureDomains{
		Platform: configv1.AWSPlatformType,
		AWS:      &[]machinev1.AWSFailureDomain{},
	}

	for _, builder := range a.failureDomainsBuilders {
		*fds.AWS = append(*fds.AWS, builder.Build())
	}

	return fds
}

// WithFailureDomainBuilder adds a failure domain builder to the failure domains builder's builders.
func (a AWSFailureDomainsBuilder) WithFailureDomainBuilder(fdBuilder AWSFailureDomainBuilder) AWSFailureDomainsBuilder {
	a.failureDomainsBuilders = append(a.failureDomainsBuilders, fdBuilder)
	return a
}

// WithFailureDomainBuilders replaces the failure domains builder's builders with the given builders.
func (a AWSFailureDomainsBuilder) WithFailureDomainBuilders(fdBuilders ...AWSFailureDomainBuilder) AWSFailureDomainsBuilder {
	a.failureDomainsBuilders = fdBuilders
	return a
}

// AWSFailureDomain creates a new failure domain builder for AWS.
func AWSFailureDomain() AWSFailureDomainBuilder {
	return AWSFailureDomainBuilder{}
}

// AWSFailureDomainBuilder is used to build an AWS failuredomain.
type AWSFailureDomainBuilder struct {
	availabilityZone string
	subnet           *machinev1.AWSResourceReference
}

// Build builds an AWS failuredomain from the configuration.
func (a AWSFailureDomainBuilder) Build() machinev1.AWSFailureDomain {
	return machinev1.AWSFailureDomain{
		Placement: machinev1.AWSFailureDomainPlacement{
			AvailabilityZone: a.availabilityZone,
		},
		Subnet: a.subnet,
	}
}

// WithAvailabilityZone sets the availabilityZone for the AWS failuredomain builder.
func (a AWSFailureDomainBuilder) WithAvailabilityZone(az string) AWSFailureDomainBuilder {
	a.availabilityZone = az
	return a
}

// WithSubnet sets the subnet for the AWS failuredomain builder.
func (a AWSFailureDomainBuilder) WithSubnet(subnet machinev1.AWSResourceReference) AWSFailureDomainBuilder {
	a.subnet = &subnet
	return a
}
