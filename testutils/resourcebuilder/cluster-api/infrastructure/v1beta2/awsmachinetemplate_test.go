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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
)

var _ = Describe("AWSMachineTemplate", func() {
	Describe("Build", func() {
		It("should return a default AWSMachineTemplate when no options are specified", func() {
			awsMachineTemplate := AWSMachineTemplate().Build()
			Expect(awsMachineTemplate).ToNot(BeNil())
			Expect(awsMachineTemplate.TypeMeta.APIVersion).To(Equal("infrastructure.cluster.x-k8s.io/v1beta2"))
			Expect(awsMachineTemplate.TypeMeta.Kind).To(Equal("AWSMachineTemplate"))
		})
	})
	// Object meta fields.

	Describe("WithAnnotations", func() {
		It("should return the custom value when specified", func() {
			annotations := map[string]string{"key": "value"}
			awsMachineTemplate := AWSMachineTemplate().WithAnnotations(annotations).Build()
			Expect(awsMachineTemplate.Annotations).To(Equal(annotations))
		})
	})

	Describe("WithLabels", func() {
		It("should return the custom value when specified", func() {
			labels := map[string]string{"key": "value"}
			awsMachineTemplate := AWSMachineTemplate().WithLabels(labels).Build()
			Expect(awsMachineTemplate.Labels).To(Equal(labels))
		})
	})

	Describe("WithGenerateName", func() {
		It("should return the custom value when specified", func() {
			awsMachineTemplate := AWSMachineTemplate().WithGenerateName(testPrefix).Build()
			Expect(awsMachineTemplate.GenerateName).To(Equal(testPrefix))
		})
	})

	Describe("WithCreationTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			awsMachineTemplate := AWSMachineTemplate().WithCreationTimestamp(timestamp).Build()
			Expect(awsMachineTemplate.CreationTimestamp).To(Equal(timestamp))
		})
	})

	Describe("WithDeletionTimestamp", func() {
		It("should return the custom value when specified", func() {
			timestamp := metav1.Now()
			awsMachineTemplate := AWSMachineTemplate().WithDeletionTimestamp(&timestamp).Build()
			Expect(awsMachineTemplate.DeletionTimestamp).To(Equal(&timestamp))
		})
	})

	Describe("WithName", func() {
		It("should return the custom value when specified", func() {
			name := "test-aws-machine-template"
			awsMachineTemplate := AWSMachineTemplate().WithName(name).Build()
			Expect(awsMachineTemplate.Name).To(Equal(name))
		})
	})

	Describe("WithNamespace", func() {
		It("should return the custom value when specified", func() {
			awsMachineTemplate := AWSMachineTemplate().WithNamespace("test-ns").Build()
			Expect(awsMachineTemplate.Namespace).To(Equal("test-ns"))
		})
	})

	Describe("WithOwnerReferences", func() {
		It("should return the custom value when specified", func() {
			ownerReferences := []metav1.OwnerReference{{Name: "cluster"}}
			awsMachineTemplate := AWSMachineTemplate().WithOwnerReferences(ownerReferences).Build()
			Expect(awsMachineTemplate.OwnerReferences).To(Equal(ownerReferences))
		})
	})

	// Spec fields.

	Describe("WithAdditionalSecurityGroups", func() {
		It("should return the custom value when specified", func() {
			groups := []capav1.AWSResourceReference{{ID: ptr.To("sg-12345")}}
			awsMachineTemplate := AWSMachineTemplate().WithAdditionalSecurityGroups(groups).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.AdditionalSecurityGroups).To(Equal(groups))
		})
	})

	Describe("WithAdditionalTags", func() {
		It("should return the custom value when specified", func() {
			tags := capav1.Tags{"key": "value"}
			awsMachineTemplate := AWSMachineTemplate().WithAdditionalTags(tags).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.AdditionalTags).To(Equal(tags))
		})
	})

	Describe("WithAMI", func() {
		It("should return the custom value when specified", func() {
			ami := capav1.AMIReference{ID: ptr.To("ami-12345")}
			awsMachineTemplate := AWSMachineTemplate().WithAMI(ami).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.AMI).To(Equal(ami))
		})
	})

	Describe("WithCapacityReservationID", func() {
		It("should return the custom value when specified", func() {
			reservationID := "cr-12345"
			awsMachineTemplate := AWSMachineTemplate().WithCapacityReservationID(reservationID).Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.CapacityReservationID).To(Equal(reservationID))
		})
	})

	Describe("WithCloudInit", func() {
		It("should return the custom value when specified", func() {
			cloudInit := capav1.CloudInit{SecretPrefix: "prefix-"}
			awsMachineTemplate := AWSMachineTemplate().WithCloudInit(cloudInit).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.CloudInit).To(Equal(cloudInit))
		})
	})

	Describe("WithElasticIPPool", func() {
		It("should return the custom value when specified", func() {
			pool := &capav1.ElasticIPPool{PublicIpv4Pool: ptr.To("test-pool")}
			awsMachineTemplate := AWSMachineTemplate().WithElasticIPPool(pool).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.ElasticIPPool).To(Equal(pool))
		})
	})

	Describe("WithIgnition", func() {
		It("should return the custom value when specified", func() {
			ignition := &capav1.Ignition{Version: "3.2.0"}
			awsMachineTemplate := AWSMachineTemplate().WithIgnition(ignition).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.Ignition).To(Equal(ignition))
		})
	})

	Describe("WithImageLookupBaseOS", func() {
		It("should return the custom value when specified", func() {
			awsMachineTemplate := AWSMachineTemplate().WithImageLookupBaseOS("RHCOS").Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.ImageLookupBaseOS).To(Equal("RHCOS"))
		})
	})

	Describe("WithImageLookupFormat", func() {
		It("should return the custom value when specified", func() {
			awsMachineTemplate := AWSMachineTemplate().WithImageLookupFormat("{{.BaseOS}}-{{.K8sVersionn}}-*").Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.ImageLookupFormat).To(Equal("{{.BaseOS}}-{{.K8sVersionn}}-*"))
		})
	})

	Describe("WithImageLookupOrg", func() {
		It("should return the custom value when specified", func() {
			org := "123456789012"
			awsMachineTemplate := AWSMachineTemplate().WithImageLookupOrg(org).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.ImageLookupOrg).To(Equal(org))
		})
	})

	Describe("WithInstanceID", func() {
		It("should return the custom value when specified", func() {
			instanceID := "i-1234567890abcdef0"
			awsMachineTemplate := AWSMachineTemplate().WithInstanceID(instanceID).Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.InstanceID).To(Equal(instanceID))
		})
	})

	Describe("WithInstanceMetadataOptions", func() {
		It("should return the custom value when specified", func() {
			options := &capav1.InstanceMetadataOptions{HTTPPutResponseHopLimit: int64(2)}
			awsMachineTemplate := AWSMachineTemplate().WithInstanceMetadataOptions(options).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.InstanceMetadataOptions).To(Equal(options))
		})
	})

	Describe("WithInstanceType", func() {
		It("should return the custom value when specified", func() {
			instanceType := "t2.micro"
			awsMachineTemplate := AWSMachineTemplate().WithInstanceType(instanceType).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.InstanceType).To(Equal(instanceType))
		})
	})

	Describe("WithNetworkInterfaces", func() {
		It("should return the custom value when specified", func() {
			interfaces := []string{"eni-12345", "eni-67890"}
			awsMachineTemplate := AWSMachineTemplate().WithNetworkInterfaces(interfaces).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.NetworkInterfaces).To(Equal(interfaces))
		})
	})

	Describe("WithNonRootVolumes", func() {
		It("should return the custom value when specified", func() {
			volumes := []capav1.Volume{{DeviceName: "test-device", Size: 100}}
			awsMachineTemplate := AWSMachineTemplate().WithNonRootVolumes(volumes).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.NonRootVolumes).To(Equal(volumes))
		})
	})

	Describe("WithPlacementGroupName", func() {
		It("should return the custom value when specified", func() {
			name := "test-placement-group"
			awsMachineTemplate := AWSMachineTemplate().WithPlacementGroupName(name).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.PlacementGroupName).To(Equal(name))
		})
	})

	Describe("WithPlacementGroupPartition", func() {
		It("should return the custom value when specified", func() {
			partition := int64(3)
			awsMachineTemplate := AWSMachineTemplate().WithPlacementGroupPartition(partition).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.PlacementGroupPartition).To(Equal(partition))
		})
	})

	Describe("WithPrivateDNSName", func() {
		It("should return the custom value when specified", func() {
			dnsName := &capav1.PrivateDNSName{EnableResourceNameDNSAAAARecord: ptr.To(true)}
			awsMachineTemplate := AWSMachineTemplate().WithPrivateDNSName(dnsName).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.PrivateDNSName).To(Equal(dnsName))
		})
	})

	Describe("WithPublicIP", func() {
		It("should return the custom value when specified", func() {
			publicIP := true
			awsMachineTemplate := AWSMachineTemplate().WithPublicIP(publicIP).Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.PublicIP).To(Equal(publicIP))
		})
	})

	Describe("WithProviderID", func() {
		It("should return the custom value when specified", func() {
			providerID := "aws://test-provider-id"
			awsMachineTemplate := AWSMachineTemplate().WithProviderID(providerID).Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.ProviderID).To(Equal(providerID))
		})
	})

	Describe("WithSSHKeyName", func() {
		It("should return the custom value when specified", func() {
			awsMachineTemplate := AWSMachineTemplate().WithSSHKeyName("keyy").Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.SSHKeyName).To(Equal("keyy"))
		})
	})

	Describe("WithIAMInstanceProfile", func() {
		It("should return the custom value when specified", func() {
			profile := "test-profile"
			awsMachineTemplate := AWSMachineTemplate().WithIAMInstanceProfile(profile).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.IAMInstanceProfile).To(Equal(profile))
		})
	})

	Describe("WithRootVolume", func() {
		It("should return the custom value when specified", func() {
			rootVolume := &capav1.Volume{Size: 100}
			awsMachineTemplate := AWSMachineTemplate().WithRootVolume(rootVolume).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.RootVolume).To(Equal(rootVolume))
		})
	})

	Describe("WithSecurityGroupOverrides", func() {
		It("should return the custom value when specified", func() {
			overrides := map[capav1.SecurityGroupRole]string{capav1.SecurityGroupNode: "sg-12345"}
			awsMachineTemplate := AWSMachineTemplate().WithSecurityGroupOverrides(overrides).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.SecurityGroupOverrides).To(Equal(overrides))
		})
	})

	Describe("WithSpotMarketOptions", func() {
		It("should return the custom value when specified", func() {
			options := &capav1.SpotMarketOptions{MaxPrice: ptr.To("0.5")}
			awsMachineTemplate := AWSMachineTemplate().WithSpotMarketOptions(options).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.SpotMarketOptions).To(Equal(options))
		})
	})

	Describe("WithSubnet", func() {
		It("should return the custom value when specified", func() {
			subnet := &capav1.AWSResourceReference{ID: ptr.To("subnet-12345")}
			awsMachineTemplate := AWSMachineTemplate().WithSubnet(subnet).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.Subnet).To(Equal(subnet))
		})
	})
	Describe("WithTenancy", func() {
		It("should return the custom value when specified", func() {
			tenancy := "dedicated"
			awsMachineTemplate := AWSMachineTemplate().WithTenancy(tenancy).Build()
			Expect(awsMachineTemplate.Spec.Template.Spec.Tenancy).To(Equal(tenancy))
		})
	})

	Describe("WithUncompressedUserData", func() {
		It("should return the custom value when specified", func() {
			uncompressed := true
			awsMachineTemplate := AWSMachineTemplate().WithUncompressedUserData(uncompressed).Build()
			Expect(*awsMachineTemplate.Spec.Template.Spec.UncompressedUserData).To(Equal(uncompressed))
		})
	})

	// Status fields.

	Describe("WithCapacity", func() {
		It("should return the custom value when specified", func() {
			capacity := corev1.ResourceList{"cpu": resource.MustParse("2")}
			awsMachineTemplate := AWSMachineTemplate().WithCapacity(capacity).Build()
			Expect(awsMachineTemplate.Status.Capacity).To(Equal(capacity))
		})
	})
})
