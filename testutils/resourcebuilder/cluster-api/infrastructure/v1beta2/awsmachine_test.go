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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
)

var _ = Describe("AWSMachineBuilder", func() {
	Describe("Build", func() {
		It("should return a default AWSMachine when no options are specified", func() {
			awsMachine := AWSMachine().Build()
			Expect(awsMachine).ToNot(BeNil())
			Expect(awsMachine.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(awsMachine.TypeMeta.Kind).To(Equal("AWSMachine"))
		})
	})

	// Object meta fields
	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			awsMachine := AWSMachine().WithAnnotations(annotations).Build()
			Expect(awsMachine.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			awsMachine := AWSMachine().WithLabels(labels).Build()
			Expect(awsMachine.Labels).To(Equal(labels))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			generateName := "test-aws-machine-"
			awsMachine := AWSMachine().WithGenerateName(generateName).Build()
			Expect(awsMachine.GenerateName).To(Equal(generateName))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-awsmachine"
			awsMachine := AWSMachine().WithName(name).Build()
			Expect(awsMachine.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			awsMachine := AWSMachine().WithNamespace("ns-test-5").Build()
			Expect(awsMachine.Namespace).To(Equal("ns-test-5"))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerRefs := []metav1.OwnerReference{{Name: "machineName"}}
			awsMachine := AWSMachine().WithOwnerReferences(ownerRefs).Build()
			Expect(awsMachine.OwnerReferences).To(Equal(ownerRefs))
		})
	})

	// Spec fields
	Describe("WithAdditionalSecurityGroups", func() {
		It("should return the custom value when specified", func() {
			groups := []capav1.AWSResourceReference{{ID: ptr.To("sg-12345")}}
			awsMachine := AWSMachine().WithAdditionalSecurityGroups(groups).Build()
			Expect(awsMachine.Spec.AdditionalSecurityGroups).To(Equal(groups))
		})
	})

	Describe("WithAdditionalTags", func() {
		It("should return the custom value when specified", func() {
			tags := capav1.Tags{"key": "value"}
			awsMachine := AWSMachine().WithAdditionalTags(tags).Build()
			Expect(awsMachine.Spec.AdditionalTags).To(Equal(tags))
		})
	})

	Describe("WithAMI", func() {
		It("should return the custom value when specified", func() {
			ami := capav1.AMIReference{ID: ptr.To("ami-12345")}
			awsMachine := AWSMachine().WithAMI(ami).Build()
			Expect(awsMachine.Spec.AMI).To(Equal(ami))
		})
	})

	Describe("WithCapacityReservationID", func() {
		It("should return the custom value when specified", func() {
			reservationID := "cr-12345"
			awsMachine := AWSMachine().WithCapacityReservationID(&reservationID).Build()
			Expect(awsMachine.Spec.CapacityReservationID).To(Equal(&reservationID))
		})
	})

	Describe("WithCloudInit", func() {
		It("should return the custom value when specified", func() {
			cloudInit := capav1.CloudInit{SecretPrefix: "prefix-"}
			awsMachine := AWSMachine().WithCloudInit(cloudInit).Build()
			Expect(awsMachine.Spec.CloudInit).To(Equal(cloudInit))
		})
	})

	Describe("WithElasticIPPool", func() {
		It("should return the custom value when specified", func() {
			pool := &capav1.ElasticIPPool{PublicIpv4Pool: ptr.To("test-pool")}
			awsMachine := AWSMachine().WithElasticIPPool(pool).Build()
			Expect(awsMachine.Spec.ElasticIPPool).To(Equal(pool))
		})
	})

	Describe("WithIAMInstanceProfile", func() {
		It("should return the custom value when specified", func() {
			profile := "test-profile"
			awsMachine := AWSMachine().WithIAMInstanceProfile(profile).Build()
			Expect(awsMachine.Spec.IAMInstanceProfile).To(Equal(profile))
		})
	})

	Describe("WithIgnition", func() {
		It("should return the custom value when specified", func() {
			ignition := &capav1.Ignition{Version: "3.2.0"}
			awsMachine := AWSMachine().WithIgnition(ignition).Build()
			Expect(awsMachine.Spec.Ignition).To(Equal(ignition))
		})
	})

	Describe("WithImageLookupBaseOS", func() {
		It("should return the custom value when specified", func() {
			baseOS := "RHEL"
			awsMachine := AWSMachine().WithImageLookupBaseOS(baseOS).Build()
			Expect(awsMachine.Spec.ImageLookupBaseOS).To(Equal(baseOS))
		})
	})

	Describe("WithImageLookupFormat", func() {
		It("should return the custom value when specified", func() {
			awsMachine := AWSMachine().WithImageLookupFormat("{{.BaseOS}}-{{.K8ssVersion}}-*").Build()
			Expect(awsMachine.Spec.ImageLookupFormat).To(Equal("{{.BaseOS}}-{{.K8ssVersion}}-*"))
		})
	})

	Describe("WithImageLookupOrg", func() {
		It("should return the custom value when specified", func() {
			org := "123456789012"
			awsMachine := AWSMachine().WithImageLookupOrg(org).Build()
			Expect(awsMachine.Spec.ImageLookupOrg).To(Equal(org))
		})
	})

	Describe("WithInstanceID", func() {
		It("should return the custom value when specified", func() {
			instanceID := "i-1234567890abcdef0"
			awsMachine := AWSMachine().WithInstanceID(&instanceID).Build()
			Expect(awsMachine.Spec.InstanceID).To(Equal(&instanceID))
		})
	})

	Describe("WithInstanceMetadataOptions", func() {
		It("should return the custom value when specified", func() {
			options := &capav1.InstanceMetadataOptions{HTTPPutResponseHopLimit: 2}
			awsMachine := AWSMachine().WithInstanceMetadataOptions(options).Build()
			Expect(awsMachine.Spec.InstanceMetadataOptions).To(Equal(options))
		})
	})

	Describe("WithInstanceType", func() {
		It("should return the custom value when specified", func() {
			instanceType := "t2.micro"
			awsMachine := AWSMachine().WithInstanceType(instanceType).Build()
			Expect(awsMachine.Spec.InstanceType).To(Equal(instanceType))
		})
	})

	Describe("WithNetworkInterfaces", func() {
		It("should return the custom value when specified", func() {
			interfaces := []string{"eni-12345", "eni-67890"}
			awsMachine := AWSMachine().WithNetworkInterfaces(interfaces).Build()
			Expect(awsMachine.Spec.NetworkInterfaces).To(Equal(interfaces))
		})
	})

	Describe("WithNetworkInterfaceType", func() {
		It("should return the custom value when specified", func() {
			networkInterfaceType := capav1.NetworkInterfaceTypeEFAWithENAInterface
			awsMachine := AWSMachine().WithNetworkInterfaceType(networkInterfaceType).Build()
			Expect(awsMachine.Spec.NetworkInterfaceType).To(Equal(networkInterfaceType))
		})
	})

	Describe("WithNonRootVolumes", func() {
		It("should return the custom value when specified", func() {
			volumes := []capav1.Volume{{DeviceName: "test-device", Size: 100}}
			awsMachine := AWSMachine().WithNonRootVolumes(volumes).Build()
			Expect(awsMachine.Spec.NonRootVolumes).To(Equal(volumes))
		})
	})

	Describe("WithPlacementGroupName", func() {
		It("should return the custom value when specified", func() {
			name := "test-placement-group"
			awsMachine := AWSMachine().WithPlacementGroupName(name).Build()
			Expect(awsMachine.Spec.PlacementGroupName).To(Equal(name))
		})
	})

	Describe("WithPlacementGroupPartition", func() {
		It("should return the custom value when specified", func() {
			partition := int64(3)
			awsMachine := AWSMachine().WithPlacementGroupPartition(partition).Build()
			Expect(awsMachine.Spec.PlacementGroupPartition).To(Equal(partition))
		})
	})

	Describe("WithPrivateDNSName", func() {
		It("should return the custom value when specified", func() {
			dnsName := &capav1.PrivateDNSName{EnableResourceNameDNSAAAARecord: ptr.To(true)}
			awsMachine := AWSMachine().WithPrivateDNSName(dnsName).Build()
			Expect(awsMachine.Spec.PrivateDNSName).To(Equal(dnsName))
		})
	})

	Describe("WithProviderID", func() {
		It("should return the custom value when specified", func() {
			providerID := "aws://test-provider-id"
			awsMachine := AWSMachine().WithProviderID(&providerID).Build()
			Expect(awsMachine.Spec.ProviderID).To(Equal(&providerID))
		})
	})

	Describe("WithPublicIP", func() {
		It("should return the custom value when specified", func() {
			publicIP := true
			awsMachine := AWSMachine().WithPublicIP(&publicIP).Build()
			Expect(awsMachine.Spec.PublicIP).To(Equal(&publicIP))
		})
	})

	Describe("WithRootVolume", func() {
		It("should return the custom value when specified", func() {
			rootVolume := &capav1.Volume{Size: 100}
			awsMachine := AWSMachine().WithRootVolume(rootVolume).Build()
			Expect(awsMachine.Spec.RootVolume).To(Equal(rootVolume))
		})
	})

	Describe("WithSecurityGroupOverrides", func() {
		It("should return the custom value when specified", func() {
			overrides := map[capav1.SecurityGroupRole]string{capav1.SecurityGroupNode: "sg-12345"}
			awsMachine := AWSMachine().WithSecurityGroupOverrides(overrides).Build()
			Expect(awsMachine.Spec.SecurityGroupOverrides).To(Equal(overrides))
		})
	})

	Describe("WithSpotMarketOptions", func() {
		It("should return the custom value when specified", func() {
			options := &capav1.SpotMarketOptions{MaxPrice: ptr.To("0.5")}
			awsMachine := AWSMachine().WithSpotMarketOptions(options).Build()
			Expect(awsMachine.Spec.SpotMarketOptions).To(Equal(options))
		})
	})

	Describe("WithSSHKeyName", func() {
		It("should return the custom value when specified", func() {
			sshKeyName := "tst-key-1"
			awsMachine := AWSMachine().WithSSHKeyName(&sshKeyName).Build()
			Expect(awsMachine.Spec.SSHKeyName).To(Equal(&sshKeyName))
		})
	})

	Describe("WithSubnet", func() {
		It("should return the custom value when specified", func() {
			subnet := &capav1.AWSResourceReference{ID: ptr.To("subnet-12345")}
			awsMachine := AWSMachine().WithSubnet(subnet).Build()
			Expect(awsMachine.Spec.Subnet).To(Equal(subnet))
		})
	})

	Describe("WithTenancy", func() {
		It("should return the custom value when specified", func() {
			tenancy := "dedicated"
			awsMachine := AWSMachine().WithTenancy(tenancy).Build()
			Expect(awsMachine.Spec.Tenancy).To(Equal(tenancy))
		})
	})

	Describe("WithUncompressedUserData", func() {
		It("should return the custom value when specified", func() {
			uncompressed := true
			awsMachine := AWSMachine().WithUncompressedUserData(&uncompressed).Build()
			Expect(awsMachine.Spec.UncompressedUserData).To(Equal(&uncompressed))
		})
	})

	// Status fields
	Describe("WithAddresses", func() {
		It("should return the custom value when specified", func() {
			addresses := []clusterv1beta1.MachineAddress{{Type: clusterv1beta1.MachineInternalIP, Address: "192.168.1.1"}}
			awsMachine := AWSMachine().WithAddresses(addresses).Build()
			Expect(awsMachine.Status.Addresses).To(Equal(addresses))
		})
	})

	Describe("WithConditions", func() {
		It("should return the custom value when specified", func() {
			conditions := clusterv1beta1.Conditions{{Type: clusterv1beta1.ReadyCondition, Status: corev1.ConditionTrue}}
			awsMachine := AWSMachine().WithConditions(conditions).Build()
			Expect(awsMachine.Status.Conditions).To(Equal(conditions))
		})
	})

	Describe("WithFailureMessage", func() {
		It("should return the custom value when specified", func() {
			message := "test failure message"
			awsMachine := AWSMachine().WithFailureMessage(&message).Build()
			Expect(awsMachine.Status.FailureMessage).To(Equal(&message))
		})
	})

	Describe("WithFailureReason", func() {
		It("should return the custom value when specified", func() {
			reason := "CreateError"
			awsMachine := AWSMachine().WithFailureReason(&reason).Build()
			Expect(awsMachine.Status.FailureReason).To(Equal(&reason))
		})
	})

	Describe("WithInstanceState", func() {
		It("should return the custom value when specified", func() {
			state := capav1.InstanceStateRunning
			awsMachine := AWSMachine().WithInstanceState(&state).Build()
			Expect(*awsMachine.Status.InstanceState).To(Equal(state))
		})
	})

	Describe("WithInterruptible", func() {
		It("should return the custom value when specified", func() {
			awsMachine := AWSMachine().WithInterruptible(true).Build()
			Expect(awsMachine.Status.Interruptible).To(Equal(true))
		})
	})

	Describe("WithReady", func() {
		It("should return the custom value when specified", func() {
			awsMachine := AWSMachine().WithReady(true).Build()
			Expect(awsMachine.Status.Ready).To(Equal(true))
		})
	})
})
