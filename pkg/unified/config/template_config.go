package config

import (
	"encoding/json"
	"fmt"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
)

// MachineTemplateConfig defines common configuration for machine templates.
type MachineTemplateConfig struct {
	// AWS specific configurations
	AWS *AWSMachineConfig `json:"aws,omitempty"`
	// Azure specific configurations
	Azure *AzureMachineConfig `json:"azure,omitempty"`
	// GCP specific configurations
	GCP *GCPMachineConfig `json:"gcp,omitempty"`
}

// AWSMachineConfig AWS platform machine configuration.
type AWSMachineConfig struct {
	InstanceType         *string               `json:"instanceType,omitempty"`
	SpotMarketOptions    *SpotMarketConfig     `json:"spotMarketOptions,omitempty"`
	PlacementGroup       *PlacementGroupConfig `json:"placementGroup,omitempty"`
	KMSKey               *KMSKeyConfig         `json:"kmsKey,omitempty"`
	AdditionalTags       map[string]string     `json:"additionalTags,omitempty"`
	Tenancy              *string               `json:"tenancy,omitempty"`
	NetworkInterfaceType *string               `json:"networkInterfaceType,omitempty"`
	NonRootVolumes       []VolumeConfig        `json:"nonRootVolumes,omitempty"`
}

// SpotMarketConfig spot instance configuration.
type SpotMarketConfig struct {
	MaxPrice *string `json:"maxPrice,omitempty"`
}

// PlacementGroupConfig placement group configuration.
type PlacementGroupConfig struct {
	Name string `json:"name"`
}

// KMSKeyConfig KMS key configuration.
type KMSKeyConfig struct {
	KeyID string `json:"keyId"`
}

// VolumeConfig storage volume configuration.
type VolumeConfig struct {
	DeviceName string `json:"deviceName"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
}

// AzureMachineConfig Azure platform machine configuration (reserved for future extension).
type AzureMachineConfig struct {
	// Future: Azure specific configurations
}

// GCPMachineConfig GCP platform machine configuration (reserved for future extension).
type GCPMachineConfig struct {
	// Future: GCP specific configurations
}

// ConfigureMachineTemplate configures machine template using configuration object.
func ConfigureMachineTemplate(template interface{}, config *MachineTemplateConfig) error {
	if config == nil {
		return nil
	}

	switch t := template.(type) {
	case *awsv1.AWSMachineTemplate:
		return configureAWSCAPITemplate(t, config.AWS)
	case *runtime.RawExtension:
		return configureAWSMAPIProviderSpec(t, config.AWS)
	default:
		return fmt.Errorf("unsupported template type: %T", template)
	}
}

// configureAWSCAPITemplate configures CAPI AWS template.
func configureAWSCAPITemplate(template *awsv1.AWSMachineTemplate, config *AWSMachineConfig) error {
	if config == nil {
		return nil
	}

	spec := &template.Spec.Template.Spec

	// Configure instance type
	if config.InstanceType != nil {
		spec.InstanceType = *config.InstanceType
	}

	// Configure spot instance
	if config.SpotMarketOptions != nil {
		spec.SpotMarketOptions = &awsv1.SpotMarketOptions{}
		if config.SpotMarketOptions.MaxPrice != nil {
			spec.SpotMarketOptions.MaxPrice = config.SpotMarketOptions.MaxPrice
		}
	}

	// Configure placement group
	if config.PlacementGroup != nil {
		spec.PlacementGroupName = config.PlacementGroup.Name
	}

	// Configure tenancy type
	if config.Tenancy != nil {
		spec.Tenancy = *config.Tenancy
	}

	// Configure network interface type
	if config.NetworkInterfaceType != nil {
		if *config.NetworkInterfaceType == "efa" {
			spec.NetworkInterfaceType = awsv1.NetworkInterfaceTypeEFAWithENAInterface
		}
	}

	// Configure additional tags
	if len(config.AdditionalTags) > 0 {
		if spec.AdditionalTags == nil {
			spec.AdditionalTags = make(map[string]string)
		}

		for k, v := range config.AdditionalTags {
			spec.AdditionalTags[k] = v
		}
	}

	// Configure non-root volumes
	if len(config.NonRootVolumes) > 0 {
		volumes := make([]awsv1.Volume, len(config.NonRootVolumes))
		for i, v := range config.NonRootVolumes {
			volumes[i] = awsv1.Volume{
				DeviceName: v.DeviceName,
				Size:       v.Size,
				Type:       awsv1.VolumeType(v.Type),
			}
		}

		spec.NonRootVolumes = volumes
	}

	// Configure KMS encryption for root volume
	if config.KMSKey != nil {
		if spec.RootVolume == nil {
			// Initialize RootVolume if it doesn't exist
			// We need at least Size to create a volume, use a sensible default
			spec.RootVolume = &awsv1.Volume{
				Size: 120, // Default root volume size
			}
		}

		spec.RootVolume.EncryptionKey = config.KMSKey.KeyID
		// KMS encryption requires Encrypted to be true
		encrypted := true
		spec.RootVolume.Encrypted = &encrypted
	}

	return nil
}

// configureAWSMAPIProviderSpec configures MAPI AWS ProviderSpec.
func configureAWSMAPIProviderSpec(providerSpec *runtime.RawExtension, config *AWSMachineConfig) error {
	if config == nil {
		return nil
	}

	var spec machinev1.AWSMachineProviderConfig

	err := json.Unmarshal(providerSpec.Raw, &spec)
	if err != nil {
		return fmt.Errorf("failed to unmarshal providerspec: %w", err)
	}

	// Configure instance type
	if config.InstanceType != nil {
		spec.InstanceType = *config.InstanceType
	}

	// Configure spot instance
	if config.SpotMarketOptions != nil {
		spec.SpotMarketOptions = &machinev1.SpotMarketOptions{}
		if config.SpotMarketOptions.MaxPrice != nil {
			spec.SpotMarketOptions.MaxPrice = config.SpotMarketOptions.MaxPrice
		}
	}

	// Configure placement group
	if config.PlacementGroup != nil {
		spec.PlacementGroupName = config.PlacementGroup.Name
	}

	// Configure tenancy type
	if config.Tenancy != nil {
		spec.Placement.Tenancy = machinev1.InstanceTenancy(*config.Tenancy)
	}

	// Configure network interface type
	if config.NetworkInterfaceType != nil {
		if *config.NetworkInterfaceType == "efa" {
			spec.NetworkInterfaceType = machinev1.AWSEFANetworkInterfaceType
		}
	}

	// Configure additional tags
	if len(config.AdditionalTags) > 0 {
		if spec.Tags == nil {
			spec.Tags = make([]machinev1.TagSpecification, 0)
		}
		// Note: MAPI tag structure differs from CAPI, adaptation required here
		for k, v := range config.AdditionalTags {
			spec.Tags = append(spec.Tags, machinev1.TagSpecification{
				Name:  k,
				Value: v,
			})
		}
	}

	// Configure non-root volumes
	if len(config.NonRootVolumes) > 0 {
		for _, v := range config.NonRootVolumes {
			blockDevice := machinev1.BlockDeviceMappingSpec{
				DeviceName: &v.DeviceName,
				EBS: &machinev1.EBSBlockDeviceSpec{
					VolumeSize: &v.Size,
					VolumeType: &v.Type,
				},
			}
			spec.BlockDevices = append(spec.BlockDevices, blockDevice)
		}
	}

	// Configure KMS encryption for root volume
	if config.KMSKey != nil {
		// Find or create root volume (BlockDevice without DeviceName or with empty DeviceName)
		var rootVolume *machinev1.BlockDeviceMappingSpec

		for i := range spec.BlockDevices {
			if spec.BlockDevices[i].DeviceName == nil || *spec.BlockDevices[i].DeviceName == "" {
				rootVolume = &spec.BlockDevices[i]
				break
			}
		}

		// If no root volume exists, create one
		if rootVolume == nil {
			rootSize := int64(120) // Default root volume size
			blockDevice := machinev1.BlockDeviceMappingSpec{
				EBS: &machinev1.EBSBlockDeviceSpec{
					VolumeSize: &rootSize,
				},
			}
			spec.BlockDevices = append(spec.BlockDevices, blockDevice)
			rootVolume = &spec.BlockDevices[len(spec.BlockDevices)-1]
		}

		// Ensure EBS is initialized
		if rootVolume.EBS == nil {
			rootSize := int64(120)
			rootVolume.EBS = &machinev1.EBSBlockDeviceSpec{
				VolumeSize: &rootSize,
			}
		}

		// Ensure KMSKey is initialized
		if rootVolume.EBS.KMSKey.ID == nil {
			rootVolume.EBS.KMSKey = machinev1.AWSResourceReference{}
		}

		// Apply KMS encryption
		rootVolume.EBS.KMSKey.ID = &config.KMSKey.KeyID
		encrypted := true
		rootVolume.EBS.Encrypted = &encrypted
	}

	// Re-serialize
	providerSpec.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal providerspec: %w", err)
	}

	return nil
}
