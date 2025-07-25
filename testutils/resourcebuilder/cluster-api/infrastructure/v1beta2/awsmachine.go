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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// AWSMachine creates a new AWSMachine builder.
func AWSMachine() AWSMachineBuilder {
	return AWSMachineBuilder{}
}

// AWSMachineBuilder is used to build out an AWSMachine object.
type AWSMachineBuilder struct {
	// ObjectMeta fields.
	annotations     map[string]string
	labels          map[string]string
	generateName    string
	name            string
	namespace       string
	ownerReferences []metav1.OwnerReference

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
	addresses      []clusterv1.MachineAddress
	conditions     clusterv1.Conditions
	failureMessage *string
	failureReason  *string
	instanceState  *capav1.InstanceState
	interruptible  bool
	ready          bool
}

// Build builds a new AWSMachine based on the configuration provided.
func (a AWSMachineBuilder) Build() *capav1.AWSMachine {
	awsMachine := &capav1.AWSMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:    a.generateName,
			Name:            a.name,
			Namespace:       a.namespace,
			Labels:          a.labels,
			Annotations:     a.annotations,
			OwnerReferences: a.ownerReferences,
		},
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
		Status: capav1.AWSMachineStatus{
			Addresses:      a.addresses,
			Conditions:     a.conditions,
			FailureMessage: a.failureMessage,
			FailureReason:  a.failureReason,
			InstanceState:  a.instanceState,
			Interruptible:  a.interruptible,
			Ready:          a.ready,
		},
	}

	return awsMachine
}

// Object meta fields.

// WithAnnotations sets the annotations for the AWSMachine builder.
func (a AWSMachineBuilder) WithAnnotations(annotations map[string]string) AWSMachineBuilder {
	a.annotations = annotations
	return a
}

// WithLabels sets the labels for the AWSMachine builder.
func (a AWSMachineBuilder) WithLabels(labels map[string]string) AWSMachineBuilder {
	a.labels = labels
	return a
}

// WithGenerateName sets the generateName for the AWSMachine builder.
func (a AWSMachineBuilder) WithGenerateName(generateName string) AWSMachineBuilder {
	a.generateName = generateName
	return a
}

// WithName sets the name for the AWSMachine builder.
func (a AWSMachineBuilder) WithName(name string) AWSMachineBuilder {
	a.name = name
	return a
}

// WithNamespace sets the namespace for the AWSMachine builder.
func (a AWSMachineBuilder) WithNamespace(namespace string) AWSMachineBuilder {
	a.namespace = namespace
	return a
}

// WithOwnerReferences sets the OwnerReferences for the machine builder.
func (a AWSMachineBuilder) WithOwnerReferences(ownerRefs []metav1.OwnerReference) AWSMachineBuilder {
	a.ownerReferences = ownerRefs
	return a
}

// Spec fields.

// WithAdditionalSecurityGroups sets the additionalSecurityGroups for the AWSMachine builder.
func (a AWSMachineBuilder) WithAdditionalSecurityGroups(additionalSecurityGroups []capav1.AWSResourceReference) AWSMachineBuilder {
	a.additionalSecurityGroups = additionalSecurityGroups
	return a
}

// WithAdditionalTags sets the additionalTags for the AWSMachine builder.
func (a AWSMachineBuilder) WithAdditionalTags(additionalTags capav1.Tags) AWSMachineBuilder {
	a.additionalTags = additionalTags
	return a
}

// WithAMI sets the AMI for the AWSMachine builder.
func (a AWSMachineBuilder) WithAMI(ami capav1.AMIReference) AWSMachineBuilder {
	a.ami = ami
	return a
}

// WithCapacityReservationID sets the capacityReservationID for the AWSMachine builder.
func (a AWSMachineBuilder) WithCapacityReservationID(capacityReservationID *string) AWSMachineBuilder {
	a.capacityReservationID = capacityReservationID
	return a
}

// WithCloudInit sets the cloudInit for the AWSMachine builder.
func (a AWSMachineBuilder) WithCloudInit(cloudInit capav1.CloudInit) AWSMachineBuilder {
	a.cloudInit = cloudInit
	return a
}

// WithElasticIPPool sets the elasticIPPool for the AWSMachine builder.
func (a AWSMachineBuilder) WithElasticIPPool(elasticIPPool *capav1.ElasticIPPool) AWSMachineBuilder {
	a.elasticIPPool = elasticIPPool
	return a
}

// WithIAMInstanceProfile sets the iamInstanceProfile for the AWSMachine builder.
func (a AWSMachineBuilder) WithIAMInstanceProfile(iamInstanceProfile string) AWSMachineBuilder {
	a.iamInstanceProfile = iamInstanceProfile
	return a
}

// WithIgnition sets the ignition for the AWSMachine builder.
func (a AWSMachineBuilder) WithIgnition(ignition *capav1.Ignition) AWSMachineBuilder {
	a.ignition = ignition
	return a
}

// WithImageLookupBaseOS sets the imageLookupBaseOS for the AWSMachine builder.
func (a AWSMachineBuilder) WithImageLookupBaseOS(imageLookupBaseOS string) AWSMachineBuilder {
	a.imageLookupBaseOS = imageLookupBaseOS
	return a
}

// WithImageLookupFormat sets the imageLookupFormat for the AWSMachine builder.
func (a AWSMachineBuilder) WithImageLookupFormat(imageLookupFormat string) AWSMachineBuilder {
	a.imageLookupFormat = imageLookupFormat
	return a
}

// WithImageLookupOrg sets the imageLookupOrg for the AWSMachine builder.
func (a AWSMachineBuilder) WithImageLookupOrg(imageLookupOrg string) AWSMachineBuilder {
	a.imageLookupOrg = imageLookupOrg
	return a
}

// WithInstanceID sets the instanceID for the AWSMachine builder.
func (a AWSMachineBuilder) WithInstanceID(instanceID *string) AWSMachineBuilder {
	a.instanceID = instanceID
	return a
}

// WithInstanceMetadataOptions sets the instanceMetadataOptions for the AWSMachine builder.
func (a AWSMachineBuilder) WithInstanceMetadataOptions(instanceMetadataOptions *capav1.InstanceMetadataOptions) AWSMachineBuilder {
	a.instanceMetadataOptions = instanceMetadataOptions
	return a
}

// WithInstanceType sets the instanceType for the AWSMachine builder.
func (a AWSMachineBuilder) WithInstanceType(instanceType string) AWSMachineBuilder {
	a.instanceType = instanceType
	return a
}

// WithNetworkInterfaces sets the networkInterfaces for the AWSMachine builder.
func (a AWSMachineBuilder) WithNetworkInterfaces(networkInterfaces []string) AWSMachineBuilder {
	a.networkInterfaces = networkInterfaces
	return a
}

// WithNetworkInterfaceType sets the networkInterfaceType for the AWSMachine builder.
func (a AWSMachineBuilder) WithNetworkInterfaceType(networkInterfaceType capav1.NetworkInterfaceType) AWSMachineBuilder {
	a.networkInterfaceType = networkInterfaceType
	return a
}

// WithNonRootVolumes sets the nonRootVolumes for the AWSMachine builder.
func (a AWSMachineBuilder) WithNonRootVolumes(nonRootVolumes []capav1.Volume) AWSMachineBuilder {
	a.nonRootVolumes = nonRootVolumes
	return a
}

// WithPlacementGroupName sets the placementGroupName for the AWSMachine builder.
func (a AWSMachineBuilder) WithPlacementGroupName(placementGroupName string) AWSMachineBuilder {
	a.placementGroupName = placementGroupName
	return a
}

// WithPlacementGroupPartition sets the placementGroupPartition for the AWSMachine builder.
func (a AWSMachineBuilder) WithPlacementGroupPartition(placementGroupPartition int64) AWSMachineBuilder {
	a.placementGroupPartition = placementGroupPartition
	return a
}

// WithPrivateDNSName sets the privateDNSName for the AWSMachine builder.
func (a AWSMachineBuilder) WithPrivateDNSName(privateDNSName *capav1.PrivateDNSName) AWSMachineBuilder {
	a.privateDNSName = privateDNSName
	return a
}

// WithProviderID sets the providerID for the AWSMachine builder.
func (a AWSMachineBuilder) WithProviderID(providerID *string) AWSMachineBuilder {
	a.providerID = providerID
	return a
}

// WithPublicIP sets the publicIP for the AWSMachine builder.
func (a AWSMachineBuilder) WithPublicIP(publicIP *bool) AWSMachineBuilder {
	a.publicIP = publicIP
	return a
}

// WithRootVolume sets the rootVolume for the AWSMachine builder.
func (a AWSMachineBuilder) WithRootVolume(rootVolume *capav1.Volume) AWSMachineBuilder {
	a.rootVolume = rootVolume
	return a
}

// WithSecurityGroupOverrides sets the securityGroupOverrides for the AWSMachine builder.
func (a AWSMachineBuilder) WithSecurityGroupOverrides(securityGroupOverrides map[capav1.SecurityGroupRole]string) AWSMachineBuilder {
	a.securityGroupOverrides = securityGroupOverrides
	return a
}

// WithSpotMarketOptions sets the spotMarketOptions for the AWSMachine builder.
func (a AWSMachineBuilder) WithSpotMarketOptions(spotMarketOptions *capav1.SpotMarketOptions) AWSMachineBuilder {
	a.spotMarketOptions = spotMarketOptions
	return a
}

// WithSSHKeyName sets the sshKeyName for the AWSMachine builder.
func (a AWSMachineBuilder) WithSSHKeyName(sshKeyName *string) AWSMachineBuilder {
	a.sshKeyName = sshKeyName
	return a
}

// WithSubnet sets the subnet for the AWSMachine builder.
func (a AWSMachineBuilder) WithSubnet(subnet *capav1.AWSResourceReference) AWSMachineBuilder {
	a.subnet = subnet
	return a
}

// WithTenancy sets the tenancy for the AWSMachine builder.
func (a AWSMachineBuilder) WithTenancy(tenancy string) AWSMachineBuilder {
	a.tenancy = tenancy
	return a
}

// WithUncompressedUserData sets the uncompressedUserData for the AWSMachine builder.
func (a AWSMachineBuilder) WithUncompressedUserData(uncompressedUserData *bool) AWSMachineBuilder {
	a.uncompressedUserData = uncompressedUserData
	return a
}

// Status fields.

// WithAddresses sets the addresses for the AWSMachine builder.
func (a AWSMachineBuilder) WithAddresses(addresses []clusterv1.MachineAddress) AWSMachineBuilder {
	a.addresses = addresses
	return a
}

// WithConditions sets the conditions for the AWSMachine builder.
func (a AWSMachineBuilder) WithConditions(conditions clusterv1.Conditions) AWSMachineBuilder {
	a.conditions = conditions
	return a
}

// WithFailureMessage sets the failureMessage for the AWSMachine builder.
func (a AWSMachineBuilder) WithFailureMessage(failureMessage *string) AWSMachineBuilder {
	a.failureMessage = failureMessage
	return a
}

// WithFailureReason sets the failureReason for the AWSMachine builder.
func (a AWSMachineBuilder) WithFailureReason(failureReason *string) AWSMachineBuilder {
	a.failureReason = failureReason
	return a
}

// WithInstanceState sets the instanceState for the AWSMachine builder.
func (a AWSMachineBuilder) WithInstanceState(instanceState *capav1.InstanceState) AWSMachineBuilder {
	a.instanceState = instanceState
	return a
}

// WithInterruptible sets the interruptible for the AWSMachine builder.
func (a AWSMachineBuilder) WithInterruptible(interruptible bool) AWSMachineBuilder {
	a.interruptible = interruptible
	return a
}

// WithReady sets the ready for the AWSMachine builder.
func (a AWSMachineBuilder) WithReady(ready bool) AWSMachineBuilder {
	a.ready = ready
	return a
}
