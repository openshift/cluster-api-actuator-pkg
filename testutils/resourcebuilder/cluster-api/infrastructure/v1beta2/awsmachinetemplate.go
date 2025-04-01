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

package v1beta2

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
)

// AWSMachineTemplate creates a new AWSMachineTemplate builder.
func AWSMachineTemplate() AWSMachineTemplateBuilder {
	return AWSMachineTemplateBuilder{}
}

// AWSMachineTemplateBuilder is used to build out an AWSMachineTemplate object.
type AWSMachineTemplateBuilder struct {
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
	additionalSecurityGroups []capav1.AWSResourceReference
	additionalTags           capav1.Tags
	ami                      capav1.AMIReference
	capacityReservationID    *string
	cloudInit                capav1.CloudInit
	elasticIPPool            *capav1.ElasticIPPool
	iamInstanceProfile       string
	ignition                 *capav1.Ignition
	imageLookupBaseOS        string
	imageLookupFormat        string
	imageLookupOrg           string
	instanceID               *string
	instanceMetadataOptions  *capav1.InstanceMetadataOptions
	instanceType             string
	networkInterfaces        []string
	networkInterfaceType     capav1.NetworkInterfaceType
	nonRootVolumes           []capav1.Volume
	placementGroupName       string
	placementGroupPartition  int64
	privateDNSName           *capav1.PrivateDNSName
	providerID               *string
	publicIP                 *bool
	rootVolume               *capav1.Volume
	securityGroupOverrides   map[capav1.SecurityGroupRole]string
	spotMarketOptions        *capav1.SpotMarketOptions
	sshKeyName               *string
	subnet                   *capav1.AWSResourceReference
	tenancy                  string
	uncompressedUserData     *bool

	// Status fields.
	capacity corev1.ResourceList
}

// Build builds a new AWSMachineTemplate based on the configuration provided.
func (a AWSMachineTemplateBuilder) Build() *capav1.AWSMachineTemplate {
	awsMachineTemplate := &capav1.AWSMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations:       a.annotations,
			CreationTimestamp: a.creationTimestamp,
			DeletionTimestamp: a.deletionTimestamp,
			GenerateName:      a.generateName,
			Labels:            a.labels,
			Name:              a.name,
			Namespace:         a.namespace,
			OwnerReferences:   a.ownerReferences,
		},
		Spec: capav1.AWSMachineTemplateSpec{
			Template: capav1.AWSMachineTemplateResource{
				Spec: capav1.AWSMachineSpec{
					AdditionalSecurityGroups: a.additionalSecurityGroups,
					AdditionalTags:           a.additionalTags,
					AMI:                      a.ami,
					CapacityReservationID:    a.capacityReservationID,
					CloudInit:                a.cloudInit,
					ElasticIPPool:            a.elasticIPPool,
					IAMInstanceProfile:       a.iamInstanceProfile,
					Ignition:                 a.ignition,
					ImageLookupBaseOS:        a.imageLookupBaseOS,
					ImageLookupFormat:        a.imageLookupFormat,
					ImageLookupOrg:           a.imageLookupOrg,
					InstanceID:               a.instanceID,
					InstanceMetadataOptions:  a.instanceMetadataOptions,
					InstanceType:             a.instanceType,
					NetworkInterfaces:        a.networkInterfaces,
					NetworkInterfaceType:     a.networkInterfaceType,
					NonRootVolumes:           a.nonRootVolumes,
					PlacementGroupName:       a.placementGroupName,
					PlacementGroupPartition:  a.placementGroupPartition,
					PrivateDNSName:           a.privateDNSName,
					ProviderID:               a.providerID,
					PublicIP:                 a.publicIP,
					RootVolume:               a.rootVolume,
					SecurityGroupOverrides:   a.securityGroupOverrides,
					SpotMarketOptions:        a.spotMarketOptions,
					SSHKeyName:               a.sshKeyName,
					Subnet:                   a.subnet,
					Tenancy:                  a.tenancy,
					UncompressedUserData:     a.uncompressedUserData,
				},
			},
		},
		Status: capav1.AWSMachineTemplateStatus{
			Capacity: a.capacity,
		},
	}

	return awsMachineTemplate
}

// Object meta fields.

// WithAnnotations sets the annotations for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithAnnotations(annotations map[string]string) AWSMachineTemplateBuilder {
	a.annotations = annotations
	return a
}

// WithCreationTimestamp sets the creationTimestamp for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithCreationTimestamp(timestamp metav1.Time) AWSMachineTemplateBuilder {
	a.creationTimestamp = timestamp
	return a
}

// WithDeletionTimestamp sets the deletionTimestamp for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithDeletionTimestamp(timestamp *metav1.Time) AWSMachineTemplateBuilder {
	a.deletionTimestamp = timestamp
	return a
}

// WithGenerateName sets the generateName for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithGenerateName(generateName string) AWSMachineTemplateBuilder {
	a.generateName = generateName
	return a
}

// WithLabels sets the labels for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithLabels(labels map[string]string) AWSMachineTemplateBuilder {
	a.labels = labels
	return a
}

// WithName sets the name for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithName(name string) AWSMachineTemplateBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithNamespace(namespace string) AWSMachineTemplateBuilder {
	a.namespace = namespace
	return a
}

// WithOwnerReferences sets the OwnerReferences for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithOwnerReferences(ownerRefs []metav1.OwnerReference) AWSMachineTemplateBuilder {
	a.ownerReferences = ownerRefs
	return a
}

// Spec fields.

// WithAdditionalSecurityGroups sets the additionalSecurityGroups for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithAdditionalSecurityGroups(groups []capav1.AWSResourceReference) AWSMachineTemplateBuilder {
	a.additionalSecurityGroups = groups
	return a
}

// WithAdditionalTags sets the additionalTags for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithAdditionalTags(tags capav1.Tags) AWSMachineTemplateBuilder {
	a.additionalTags = tags
	return a
}

// WithAMI sets the AMI for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithAMI(ami capav1.AMIReference) AWSMachineTemplateBuilder {
	a.ami = ami
	return a
}

// WithCapacityReservationID sets the capacityReservationID for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithCapacityReservationID(id string) AWSMachineTemplateBuilder {
	a.capacityReservationID = &id
	return a
}

// WithCloudInit sets the cloudInit for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithCloudInit(cloudInit capav1.CloudInit) AWSMachineTemplateBuilder {
	a.cloudInit = cloudInit
	return a
}

// WithElasticIPPool sets the elasticIPPool for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithElasticIPPool(pool *capav1.ElasticIPPool) AWSMachineTemplateBuilder {
	a.elasticIPPool = pool
	return a
}

// WithIAMInstanceProfile sets the iamInstanceProfile for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithIAMInstanceProfile(profile string) AWSMachineTemplateBuilder {
	a.iamInstanceProfile = profile
	return a
}

// WithIgnition sets the ignition for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithIgnition(ignition *capav1.Ignition) AWSMachineTemplateBuilder {
	a.ignition = ignition
	return a
}

// WithImageLookupBaseOS sets the imageLookupBaseOS for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithImageLookupBaseOS(baseOS string) AWSMachineTemplateBuilder {
	a.imageLookupBaseOS = baseOS
	return a
}

// WithImageLookupFormat sets the imageLookupFormat for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithImageLookupFormat(format string) AWSMachineTemplateBuilder {
	a.imageLookupFormat = format
	return a
}

// WithImageLookupOrg sets the imageLookupOrg for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithImageLookupOrg(org string) AWSMachineTemplateBuilder {
	a.imageLookupOrg = org
	return a
}

// WithInstanceID sets the instanceID for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithInstanceID(instanceID string) AWSMachineTemplateBuilder {
	a.instanceID = &instanceID
	return a
}

// WithInstanceMetadataOptions sets the instanceMetadataOptions for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithInstanceMetadataOptions(options *capav1.InstanceMetadataOptions) AWSMachineTemplateBuilder {
	a.instanceMetadataOptions = options
	return a
}

// WithInstanceType sets the instanceType for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithInstanceType(instanceType string) AWSMachineTemplateBuilder {
	a.instanceType = instanceType
	return a
}

// WithNetworkInterfaces sets the networkInterfaces for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithNetworkInterfaces(interfaces []string) AWSMachineTemplateBuilder {
	a.networkInterfaces = interfaces
	return a
}

// WithNetworkInterfaceType sets the networkInterfaceType networkInterfaceType for the AWSMAchineTemplate builder.
func (a AWSMachineTemplateBuilder) WithNetworkInterfaceType(networkInterfaceType capav1.NetworkInterfaceType) AWSMachineTemplateBuilder {
	a.networkInterfaceType = networkInterfaceType
	return a
}

// WithNonRootVolumes sets the nonRootVolumes for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithNonRootVolumes(volumes []capav1.Volume) AWSMachineTemplateBuilder {
	a.nonRootVolumes = volumes
	return a
}

// WithPlacementGroupName sets the placementGroupName for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithPlacementGroupName(name string) AWSMachineTemplateBuilder {
	a.placementGroupName = name
	return a
}

// WithPlacementGroupPartition sets the placementGroupPartition for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithPlacementGroupPartition(partition int64) AWSMachineTemplateBuilder {
	a.placementGroupPartition = partition
	return a
}

// WithPrivateDNSName sets the privateDNSName for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithPrivateDNSName(dnsName *capav1.PrivateDNSName) AWSMachineTemplateBuilder {
	a.privateDNSName = dnsName
	return a
}

// WithProviderID sets the providerID for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithProviderID(providerID string) AWSMachineTemplateBuilder {
	a.providerID = &providerID
	return a
}

// WithPublicIP sets the publicIP for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithPublicIP(publicIP bool) AWSMachineTemplateBuilder {
	a.publicIP = &publicIP
	return a
}

// WithRootVolume sets the rootVolume for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithRootVolume(volume *capav1.Volume) AWSMachineTemplateBuilder {
	a.rootVolume = volume
	return a
}

// WithSecurityGroupOverrides sets the securityGroupOverrides for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithSecurityGroupOverrides(overrides map[capav1.SecurityGroupRole]string) AWSMachineTemplateBuilder {
	a.securityGroupOverrides = overrides
	return a
}

// WithSpotMarketOptions sets the spotMarketOptions for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithSpotMarketOptions(options *capav1.SpotMarketOptions) AWSMachineTemplateBuilder {
	a.spotMarketOptions = options
	return a
}

// WithSSHKeyName sets the sshKeyName for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithSSHKeyName(keyName string) AWSMachineTemplateBuilder {
	a.sshKeyName = &keyName
	return a
}

// WithSubnet sets the subnet for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithSubnet(subnet *capav1.AWSResourceReference) AWSMachineTemplateBuilder {
	a.subnet = subnet
	return a
}

// WithTenancy sets the tenancy for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithTenancy(tenancy string) AWSMachineTemplateBuilder {
	a.tenancy = tenancy
	return a
}

// WithUncompressedUserData sets the uncompressedUserData for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithUncompressedUserData(uncompressed bool) AWSMachineTemplateBuilder {
	a.uncompressedUserData = &uncompressed
	return a
}

// Status fields.

// WithCapacity sets the capacity for the AWSMachineTemplate builder.
func (a AWSMachineTemplateBuilder) WithCapacity(capacity corev1.ResourceList) AWSMachineTemplateBuilder {
	a.capacity = capacity
	return a
}
