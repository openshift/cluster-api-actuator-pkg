package backends

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	yaml "sigs.k8s.io/yaml"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
)

// mapiBackend implements the MachineBackend interface for MAPI backend.
type mapiBackend struct {
	backendType      BackendType
	authoritativeAPI BackendType
}

func (m *mapiBackend) GetBackendType() BackendType      { return m.backendType }
func (m *mapiBackend) GetAuthoritativeAPI() BackendType { return m.authoritativeAPI }
func (m *mapiBackend) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	infra, err := framework.GetInfrastructure(ctx, client)
	Expect(err).NotTo(HaveOccurred(), "Should get infrastructure global object")
	Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "Should have infrastructure name on Infrastructure.Status")

	clusterName := infra.Status.InfrastructureName
	machineSet := &machinev1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.Name,
			Namespace:   framework.MachineAPINamespace,
			Labels:      params.Labels,
			Annotations: params.Annotations,
		},
		Spec: machinev1.MachineSetSpec{
			Replicas: &params.Replicas,
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{
				"machine.openshift.io/cluster-api-cluster":    clusterName,
				"machine.openshift.io/cluster-api-machineset": params.Name,
			}},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{Labels: framework.MergeLabels(params.Labels, map[string]string{
					"machine.openshift.io/cluster-api-cluster":      clusterName,
					"machine.openshift.io/cluster-api-machine-role": "worker",
					"machine.openshift.io/cluster-api-machine-type": "worker",
					"machine.openshift.io/cluster-api-machineset":   params.Name,
				})},
				Spec: machinev1.MachineSpec{},
			},
		},
	}

	// Set AuthoritativeAPI based on backend configuration
	if m.authoritativeAPI == BackendTypeCAPI {
		machineSet.Spec.Template.Spec.AuthoritativeAPI = machinev1.MachineAuthorityClusterAPI
	} else {
		machineSet.Spec.Template.Spec.AuthoritativeAPI = machinev1.MachineAuthorityMachineAPI
	}

	// Set ProviderSpec based on params.Template
	if params.Template != nil {
		providerSpec, err := m.convertTemplateToProviderSpec(params.Template)
		Expect(err).NotTo(HaveOccurred(), "Should convert template to provider spec")

		machineSet.Spec.Template.Spec.ProviderSpec = *providerSpec
	}

	Eventually(client.Create(ctx, machineSet), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should create MAPI MachineSet %s", machineSet.Name)

	return machineSet, nil
}

func (m *mapiBackend) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*machinev1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)

	Eventually(func() error {
		return client.Delete(ctx, ms)
	}, framework.WaitShort, framework.RetryShort).Should(SatisfyAny(
		Succeed(),
		WithTransform(apierrors.IsNotFound, BeTrue()),
	), "Should delete MachineSet %s/%s successfully or MachineSet should not be found",
		ms.Namespace, ms.Name)

	return nil
}

func (m *mapiBackend) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*machinev1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)

	framework.WaitForMachineSetsDeleted(ctx, client, ms)

	return nil
}

func (m *mapiBackend) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*machinev1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)

	framework.WaitForMachineSet(ctx, client, ms.Name)

	return nil
}

func (m *mapiBackend) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	ms, ok := machineSet.(*machinev1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)

	status := &MachineSetStatus{
		Replicas:          0,
		AvailableReplicas: ms.Status.AvailableReplicas,
		ReadyReplicas:     ms.Status.ReadyReplicas,
		AuthoritativeAPI:  "MachineAPI",
	}

	if ms.Spec.Replicas != nil {
		status.Replicas = *ms.Spec.Replicas
	}

	if ms.Status.AuthoritativeAPI != "" {
		status.AuthoritativeAPI = string(ms.Status.AuthoritativeAPI)
	}

	return status, nil
}

func (m *mapiBackend) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	_, ok := machineSet.(*machinev1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be MAPI MachineSet, got %T", machineSet)

	// TODO: Implement node query when needed. Currently not required by any test scenarios.
	return []corev1.Node{}, nil
}

func (m *mapiBackend) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return m.createAWSMachineTemplate(params)
	case configv1.AzurePlatformType:
		Fail("Azure machine template creation not yet implemented")
	case configv1.GCPPlatformType:
		Fail("GCP machine template creation not yet implemented")
	default:
		Fail(fmt.Sprintf("Unsupported platform: %s", platform))
	}

	return nil, fmt.Errorf("unreachable")
}

func (m *mapiBackend) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	return nil
}

// createAWSMachineTemplate creates AWS machine template, returns MAPI ProviderSpec.
func (m *mapiBackend) createAWSMachineTemplate(params BackendMachineTemplateParams) (interface{}, error) {
	// Get the default AWS MAPI ProviderSpec
	mapiProviderSpec := m.getDefaultAWSMAPIProviderSpec()

	// For MAPI backend, we directly return serialized ProviderSpec
	// This is because MAPI backend uses ProviderSpec, not independent template resources
	// Serialize ProviderSpec to RawExtension
	providerSpecBytes, err := json.Marshal(mapiProviderSpec)
	Expect(err).NotTo(HaveOccurred(), "Should marshal provider spec")

	providerSpecRaw := &runtime.RawExtension{
		Raw: providerSpecBytes,
	}

	// Apply custom configuration from params.Spec if provided
	if params.Spec != nil {
		// Apply the configuration directly to the ProviderSpec
		templateConfig, ok := params.Spec.(*config.MachineTemplateConfig)
		Expect(ok).To(BeTrue(), "Spec should be *config.MachineTemplateConfig, got %T", params.Spec)

		configErr := config.ConfigureMachineTemplate(providerSpecRaw, templateConfig)
		Expect(configErr).NotTo(HaveOccurred(), "Should apply custom configuration to MAPI ProviderSpec")
	}

	return providerSpecRaw, nil
}

// getDefaultAWSMAPIProviderSpec gets default AWS MAPI ProviderSpec.
func (m *mapiBackend) getDefaultAWSMAPIProviderSpec() *machinev1.AWSMachineProviderConfig {
	machineSetList := &machinev1.MachineSetList{}

	// List existing MAPI MachineSets
	Eventually(komega.List(machineSetList, runtimeclient.InNamespace(framework.MachineAPINamespace)), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should list MAPI machinesets")

	Expect(machineSetList.Items).NotTo(BeEmpty(), "Should have MAPI machinesets")

	// Use the first MachineSet's ProviderSpec as template
	machineSet := &machineSetList.Items[0]
	Expect(machineSet.Spec.Template.Spec.ProviderSpec.Value).NotTo(BeNil(), "Should have MAPI machineset ProviderSpec value")

	providerSpec := &machinev1.AWSMachineProviderConfig{}
	err := yaml.Unmarshal(machineSet.Spec.Template.Spec.ProviderSpec.Value.Raw, providerSpec)
	Expect(err).NotTo(HaveOccurred(), "Should unmarshal MAPI provider spec")

	return providerSpec
}

// convertTemplateToProviderSpec converts CAPI template to MAPI ProviderSpec.
func (m *mapiBackend) convertTemplateToProviderSpec(template interface{}) (*machinev1.ProviderSpec, error) {
	switch t := template.(type) {
	case *awsv1.AWSMachineTemplate:
		return m.convertAWSTemplateToProviderSpec(t)
	case *runtime.RawExtension:
		// If already RawExtension, use directly
		return &machinev1.ProviderSpec{Value: t}, nil
	default:
		Fail(fmt.Sprintf("Unsupported template type: %T", template))
	}

	return nil, fmt.Errorf("unreachable")
}

// convertAWSTemplateToProviderSpec converts AWS CAPI template to MAPI ProviderSpec.
func (m *mapiBackend) convertAWSTemplateToProviderSpec(awsTemplate *awsv1.AWSMachineTemplate) (*machinev1.ProviderSpec, error) {
	// Get default AWS MAPI ProviderSpec as base
	defaultProviderSpec := m.getDefaultAWSMAPIProviderSpec()

	// Update MAPI ProviderSpec using CAPI template configuration
	awsSpec := awsTemplate.Spec.Template.Spec

	// Update instance type
	if awsSpec.InstanceType != "" {
		defaultProviderSpec.InstanceType = awsSpec.InstanceType
	}

	// Update AMI information
	if awsSpec.AMI.ID != nil {
		defaultProviderSpec.AMI.ID = awsSpec.AMI.ID
	}

	// Update IAM instance profile
	if awsSpec.IAMInstanceProfile != "" {
		defaultProviderSpec.IAMInstanceProfile = &machinev1.AWSResourceReference{
			ID: &awsSpec.IAMInstanceProfile,
		}
	}

	// Update subnet configuration
	if awsSpec.Subnet != nil {
		if awsSpec.Subnet.ID != nil {
			defaultProviderSpec.Subnet = machinev1.AWSResourceReference{
				ID: awsSpec.Subnet.ID,
			}
		} else if len(awsSpec.Subnet.Filters) > 0 {
			filters := make([]machinev1.Filter, len(awsSpec.Subnet.Filters))
			for i, filter := range awsSpec.Subnet.Filters {
				filters[i] = machinev1.Filter{
					Name:   filter.Name,
					Values: filter.Values,
				}
			}

			defaultProviderSpec.Subnet = machinev1.AWSResourceReference{
				Filters: filters,
			}
		}
	}

	// Update security group configuration
	if len(awsSpec.AdditionalSecurityGroups) > 0 {
		securityGroups := make([]machinev1.AWSResourceReference, len(awsSpec.AdditionalSecurityGroups))

		for i, sg := range awsSpec.AdditionalSecurityGroups {
			if sg.ID != nil {
				securityGroups[i] = machinev1.AWSResourceReference{
					ID: sg.ID,
				}
			} else if len(sg.Filters) > 0 {
				filters := make([]machinev1.Filter, len(sg.Filters))
				for j, filter := range sg.Filters {
					filters[j] = machinev1.Filter{
						Name:   filter.Name,
						Values: filter.Values,
					}
				}

				securityGroups[i] = machinev1.AWSResourceReference{
					Filters: filters,
				}
			}
		}

		defaultProviderSpec.SecurityGroups = securityGroups
	}

	providerSpecBytes, err := json.Marshal(defaultProviderSpec)
	Expect(err).NotTo(HaveOccurred(), "Should marshal provider spec")

	return &machinev1.ProviderSpec{
		Value: &runtime.RawExtension{
			Raw: providerSpecBytes,
		},
	}, nil
}
