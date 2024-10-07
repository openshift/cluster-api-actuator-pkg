package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

var _ = Describe("AWSProviderSpec", func() {
	Describe("AMI", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.AMI).To(Equal(machinev1beta1.AWSResourceReference{ID: defaultAMIID}))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			ami := machinev1beta1.AWSResourceReference{}
			awsPs := AWSProviderSpec().WithAMI(ami).Build()
			Expect(awsPs.AMI).To(Equal(ami))
		})

		It("should return the custom value when specified", func() {
			ami := machinev1beta1.AWSResourceReference{ID: ptr.To[string]("custom-ami-id")}
			awsPs := AWSProviderSpec().WithAMI(ami).Build()
			Expect(awsPs.AMI).To(Equal(ami))
		})
	})

	Describe("AvailabilityZone", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.Placement.AvailabilityZone).To(Equal(defaultAvailabilityZone))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			awsPs := AWSProviderSpec().WithAvailabilityZone("").Build()
			Expect(awsPs.Placement.AvailabilityZone).To(Equal(""))
		})

		It("should return the custom value when specified", func() {
			customAZ := "us-west-2b"
			awsPs := AWSProviderSpec().WithAvailabilityZone(customAZ).Build()
			Expect(awsPs.Placement.AvailabilityZone).To(Equal(customAZ))
		})
	})

	Describe("BlockDevices", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.BlockDevices).To(Equal(defaultBlockDevices))
		})

		It("should return nil when specified as such", func() {
			awsPs := AWSProviderSpec().WithBlockDevices(nil).Build()
			Expect(awsPs.BlockDevices).To(BeNil())
		})

		It("should return a non-nil, empty value when specified as such", func() {
			devices := []machinev1beta1.BlockDeviceMappingSpec{}
			awsPs := AWSProviderSpec().WithBlockDevices(devices).Build()
			Expect(awsPs.BlockDevices).To(Equal(devices))
		})

		It("should return the custom value when specified", func() {
			customBlockDevices := []machinev1beta1.BlockDeviceMappingSpec{
				{
					EBS: &machinev1beta1.EBSBlockDeviceSpec{
						VolumeSize: ptr.To[int64](100),
						VolumeType: ptr.To[string]("gp2"),
					},
				},
			}
			awsPs := AWSProviderSpec().WithBlockDevices(customBlockDevices).Build()
			Expect(awsPs.BlockDevices).To(Equal(customBlockDevices))
		})
	})

	Describe("CredentialsSecret", func() {
		It("should return the default value when not specified", func() {
			defaultCredentialsSecret := &corev1.LocalObjectReference{
				Name: defaultCredentialsSecretName,
			}
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.CredentialsSecret).To(Equal(defaultCredentialsSecret))
		})

		It("should return nil when specified as such", func() {
			var credentialsSecret *corev1.LocalObjectReference
			awsPs := AWSProviderSpec().WithCredentialsSecret(credentialsSecret).Build()
			Expect(awsPs.CredentialsSecret).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			customCredentialsSecret := &corev1.LocalObjectReference{Name: "custom-credentials"}
			awsPs := AWSProviderSpec().WithCredentialsSecret(customCredentialsSecret).Build()
			Expect(awsPs.CredentialsSecret).To(Equal(customCredentialsSecret))
		})
	})

	Describe("DeviceIndex", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.DeviceIndex).To(Equal(defaultDeviceIndex))
		})

		It("should return the custom value when specified", func() {
			customDeviceIndex := int64(1)
			awsPs := AWSProviderSpec().WithDeviceIndex(customDeviceIndex).Build()
			Expect(awsPs.DeviceIndex).To(Equal(customDeviceIndex))
		})
	})

	Describe("IAMInstanceProfile", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.IAMInstanceProfile).To(Equal(defaultIAMInstanceProfile))
		})

		It("should return nil when specified as such", func() {
			var iamInstanceProfile *machinev1beta1.AWSResourceReference
			awsPs := AWSProviderSpec().WithIAMInstanceProfile(iamInstanceProfile).Build()
			Expect(awsPs.IAMInstanceProfile).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			iamInstanceProfile := &machinev1beta1.AWSResourceReference{ID: ptr.To[string]("aws-iam-instance-profile-00000")}
			awsPs := AWSProviderSpec().WithIAMInstanceProfile(iamInstanceProfile).Build()
			Expect(awsPs.IAMInstanceProfile).To(Equal(iamInstanceProfile))
		})
	})

	Describe("InstanceType", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.InstanceType).To(Equal(defaultInstanceType))
		})

		It("should return an empty string when specified as such", func() {
			instanceType := ""
			awsPs := AWSProviderSpec().WithInstanceType(instanceType).Build()
			Expect(awsPs.InstanceType).To(Equal(instanceType))
		})

		It("should return the custom value when specified", func() {
			instanceType := "c5.large"
			awsPs := AWSProviderSpec().WithInstanceType(instanceType).Build()
			Expect(awsPs.InstanceType).To(Equal(instanceType))
		})
	})

	Describe("KeyName", func() {
		It("should return nil when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.KeyName).To(BeNil())
		})

		It("should return nil when explicitly set to nil", func() {
			awsPs := AWSProviderSpec().WithKeyName(nil).Build()
			Expect(awsPs.KeyName).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			customKeyName := "my-key"
			awsPs := AWSProviderSpec().WithKeyName(&customKeyName).Build()
			Expect(*awsPs.KeyName).To(Equal(customKeyName))
		})
	})

	Describe("LoadBalancers", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.LoadBalancers).To(Equal(defaultLoadBalancers))
		})

		It("should return nil when specified as such", func() {
			awsPs := AWSProviderSpec().WithLoadBalancers(nil).Build()
			Expect(awsPs.LoadBalancers).To(BeNil())
		})

		It("should return a non-nil, empty value when specified as such", func() {
			lbs := []machinev1beta1.LoadBalancerReference{}
			awsPs := AWSProviderSpec().WithLoadBalancers(lbs).Build()
			Expect(awsPs.LoadBalancers).To(Equal(lbs))
		})

		It("should return the custom value when specified", func() {
			customLoadBalancers := []machinev1beta1.LoadBalancerReference{
				{
					Type: "classic",
					Name: "custom-lb",
				},
			}
			awsPs := AWSProviderSpec().WithLoadBalancers(customLoadBalancers).Build()
			Expect(awsPs.LoadBalancers).To(Equal(customLoadBalancers))
		})
	})

	Describe("MetadataServiceOptions", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.MetadataServiceOptions).To(Equal(defaultMetadataServiceOptions))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			opts := machinev1beta1.MetadataServiceOptions{}
			awsPs := AWSProviderSpec().WithMetadataServiceOptions(opts).Build()
			Expect(awsPs.MetadataServiceOptions).To(Equal(opts))
		})

		It("should return the custom value when specified", func() {
			customOpts := machinev1beta1.MetadataServiceOptions{
				Authentication: "optional",
			}
			awsPs := AWSProviderSpec().WithMetadataServiceOptions(customOpts).Build()
			Expect(awsPs.MetadataServiceOptions).To(Equal(customOpts))
		})
	})

	Describe("NetworkInterfaceType", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.NetworkInterfaceType).To(Equal(defaultNetworkInterfaceType))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			nType := machinev1beta1.AWSNetworkInterfaceType("")
			awsPs := AWSProviderSpec().WithNetworkInterfaceType(nType).Build()
			Expect(awsPs.NetworkInterfaceType).To(Equal(nType))
		})

		It("should return the custom value when specified", func() {
			customType := machinev1beta1.AWSNetworkInterfaceType("efa")
			awsPs := AWSProviderSpec().WithNetworkInterfaceType(customType).Build()
			Expect(awsPs.NetworkInterfaceType).To(Equal(customType))
		})
	})

	Describe("Placement", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.Placement).To(Equal(defaultPlacement))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			placement := machinev1beta1.Placement{}
			awsPs := AWSProviderSpec().WithPlacement(placement).Build()
			Expect(awsPs.Placement).To(Equal(placement))
		})

		It("should return the custom value when specified", func() {
			placement := machinev1beta1.Placement{Region: "eu-west-1"}
			awsPs := AWSProviderSpec().WithPlacement(placement).Build()
			Expect(awsPs.Placement).To(Equal(placement))
		})
	})

	Describe("PlacementGroupName", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.PlacementGroupName).To(Equal(defaultPlacementGroupName))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			awsPs := AWSProviderSpec().WithPlacementGroupName("").Build()
			Expect(awsPs.PlacementGroupName).To(Equal(""))
		})

		It("should return the custom value when specified", func() {
			customName := "my-placement-group"
			awsPs := AWSProviderSpec().WithPlacementGroupName(customName).Build()
			Expect(awsPs.PlacementGroupName).To(Equal(customName))
		})
	})

	Describe("PublicIP", func() {
		It("should return nil when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.PublicIP).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			customPublicIP := true
			awsPs := AWSProviderSpec().WithPublicIP(&customPublicIP).Build()
			Expect(*awsPs.PublicIP).To(BeTrue())
		})

		It("should return nil when explicitly set to nil", func() {
			awsPs := AWSProviderSpec().WithPublicIP(nil).Build()
			Expect(awsPs.PublicIP).To(BeNil())
		})
	})

	Describe("Region", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.Placement.Region).To(Equal(defaultRegion))
		})

		It("should return an empty string when specified as such", func() {
			region := ""
			awsPs := AWSProviderSpec().WithRegion(region).Build()
			Expect(awsPs.Placement.Region).To(Equal(region))
		})

		It("should return the custom value when specified", func() {
			region := "eu-west-1"
			awsPs := AWSProviderSpec().WithRegion(region).Build()
			Expect(awsPs.Placement.Region).To(Equal(region))
		})
	})

	Describe("SecurityGroups", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.SecurityGroups).To(Equal(defaultSecurityGroups))
		})

		It("should return nil when explicitly set to nil", func() {
			awsPs := AWSProviderSpec().WithSecurityGroups(nil).Build()
			Expect(awsPs.SecurityGroups).To(BeNil())
		})

		It("should return a non-nil, empty value when specified as such", func() {
			sgs := []machinev1beta1.AWSResourceReference{}
			awsPs := AWSProviderSpec().WithSecurityGroups(sgs).Build()
			Expect(awsPs.SecurityGroups).To(Equal(sgs))
		})

		It("should return the custom value when specified", func() {
			customSecurityGroups := []machinev1beta1.AWSResourceReference{
				{
					ID: ptr.To[string]("sg-custom-id"),
				},
			}
			awsPs := AWSProviderSpec().WithSecurityGroups(customSecurityGroups).Build()
			Expect(awsPs.SecurityGroups).To(Equal(customSecurityGroups))
		})
	})

	Describe("SpotMarketOptions", func() {
		It("should return nil when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.SpotMarketOptions).To(BeNil())
		})

		It("should return nil when explicitly set to nil", func() {
			awsPs := AWSProviderSpec().WithSpotMarketOptions(nil).Build()
			Expect(awsPs.SpotMarketOptions).To(BeNil())
		})

		It("should return a non-nil, empty value when specified as such", func() {
			spotOpts := &machinev1beta1.SpotMarketOptions{}
			awsPs := AWSProviderSpec().WithSpotMarketOptions(spotOpts).Build()
			Expect(awsPs.SpotMarketOptions).To(Equal(spotOpts))
		})

		It("should return the custom value when specified", func() {
			customSpotOpts := &machinev1beta1.SpotMarketOptions{
				MaxPrice: ptr.To[string]("0.5"),
			}
			awsPs := AWSProviderSpec().WithSpotMarketOptions(customSpotOpts).Build()
			Expect(awsPs.SpotMarketOptions).To(Equal(customSpotOpts))
		})
	})

	Describe("Subnet", func() {
		It("should return the default value when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.Subnet).To(Equal(defaultSubnet))
		})

		It("should return a non-nil, empty value when specified as such", func() {
			awsPs := AWSProviderSpec().WithSubnet(machinev1beta1.AWSResourceReference{}).Build()
			Expect(awsPs.Subnet).To(Equal(machinev1beta1.AWSResourceReference{}))
		})

		It("should return the custom value when specified", func() {
			customSubnet := machinev1beta1.AWSResourceReference{
				ID: ptr.To[string]("subnet-custom-id"),
			}
			awsPs := AWSProviderSpec().WithSubnet(customSubnet).Build()
			Expect(awsPs.Subnet).To(Equal(customSubnet))
		})
	})

	Describe("Tags", func() {
		It("should return nil when not specified", func() {
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.Tags).To(BeNil())
		})

		It("should return a non-nil, empty value when specified as such", func() {
			tags := []machinev1beta1.TagSpecification{}
			awsPs := AWSProviderSpec().WithTags(tags).Build()
			Expect(awsPs.Tags).To(Equal(tags))
		})

		It("should return nil when explicitly set to nil", func() {
			awsPs := AWSProviderSpec().WithTags(nil).Build()
			Expect(awsPs.Tags).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			customTags := []machinev1beta1.TagSpecification{
				{
					Name:  "custom-tag",
					Value: "custom-value",
				},
			}
			awsPs := AWSProviderSpec().WithTags(customTags).Build()
			Expect(awsPs.Tags).To(Equal(customTags))
		})
	})

	Describe("UserDataSecret", func() {
		It("should return the default value when not specified", func() {
			defaultUserDataSecret := &corev1.LocalObjectReference{
				Name: defaultUserDataSecretName,
			}
			awsPs := AWSProviderSpec().Build()
			Expect(awsPs.UserDataSecret).To(Equal(defaultUserDataSecret))
		})

		It("should return nil when specified as such", func() {
			awsPs := AWSProviderSpec().WithUserDataSecret(nil).Build()
			Expect(awsPs.UserDataSecret).To(BeNil())
		})

		It("should return the custom value when specified", func() {
			userDataSecret := &corev1.LocalObjectReference{Name: "bla"}
			awsPs := AWSProviderSpec().WithUserDataSecret(userDataSecret).Build()
			Expect(awsPs.UserDataSecret).To(Equal(userDataSecret))
		})
	})
})
