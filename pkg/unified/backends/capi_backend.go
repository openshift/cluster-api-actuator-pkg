package backends

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	yaml "sigs.k8s.io/yaml"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
)

// capiBackend implements the MachineBackend interface for CAPI backend.
type capiBackend struct {
	backendType      config.BackendType
	authoritativeAPI config.BackendType
}

func (c *capiBackend) GetBackendType() config.BackendType      { return c.backendType }
func (c *capiBackend) GetAuthoritativeAPI() config.BackendType { return c.authoritativeAPI }

func (c *capiBackend) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	GinkgoHelper()

	infra, err := framework.GetInfrastructure(ctx, client)
	Expect(err).NotTo(HaveOccurred(), "Should get infrastructure global object")
	Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "Should have infrastructure name on Infrastructure.Status")

	clusterName := infra.Status.InfrastructureName
	userDataSecret := "worker-user-data"
	machineSet := &clusterv1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.Name,
			Namespace:   framework.ClusterAPINamespace,
			Labels:      params.Labels,
			Annotations: params.Annotations,
		},
		Spec: clusterv1beta1.MachineSetSpec{
			ClusterName: clusterName,
			Replicas:    &params.Replicas,
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{
				"cluster.x-k8s.io/set-name":     params.Name,
				"cluster.x-k8s.io/cluster-name": clusterName,
			}},

			Template: clusterv1beta1.MachineTemplateSpec{
				ObjectMeta: clusterv1beta1.ObjectMeta{Labels: framework.MergeLabels(params.Labels, map[string]string{
					"cluster.x-k8s.io/set-name":     params.Name,
					"cluster.x-k8s.io/cluster-name": clusterName,
					framework.WorkerNodeRoleLabel:   "",
				})},
				Spec: clusterv1beta1.MachineSpec{
					Bootstrap: clusterv1beta1.Bootstrap{
						DataSecretName: &userDataSecret,
					},
					ClusterName: clusterName,
				},
			},
		},
	}

	// Set InfrastructureRef based on params.Template
	if params.Template != nil {
		c.setInfrastructureRef(&machineSet.Spec.Template.Spec, params.Template)
	}

	Eventually(func() error {
		return client.Create(ctx, machineSet)
	}, framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should create CAPI MachineSet %s", machineSet.Name)

	return machineSet, nil
}

func (c *capiBackend) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	GinkgoHelper()

	ms, ok := machineSet.(*clusterv1beta1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.DeleteCAPIMachineSets(ctx, client, ms)

	return nil
}

func (c *capiBackend) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	GinkgoHelper()

	ms, ok := machineSet.(*clusterv1beta1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.WaitForCAPIMachineSetsDeleted(ctx, client, ms)

	return nil
}

func (c *capiBackend) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	GinkgoHelper()

	ms, ok := machineSet.(*clusterv1beta1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.WaitForCAPIMachinesRunning(ctx, client, ms.Name)

	return nil
}

func (c *capiBackend) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	GinkgoHelper()

	ms, ok := machineSet.(*clusterv1beta1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	status := &MachineSetStatus{
		Replicas:          0,
		AvailableReplicas: ms.Status.AvailableReplicas,
		ReadyReplicas:     ms.Status.ReadyReplicas,
	}
	if ms.Spec.Replicas != nil {
		status.Replicas = *ms.Spec.Replicas
	}

	return status, nil
}

func (c *capiBackend) GetNodesFromMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) ([]corev1.Node, error) {
	GinkgoHelper()

	_, ok := machineSet.(*clusterv1beta1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	// TODO: Implement node query when needed. Currently not required by any test scenarios.
	return []corev1.Node{}, nil
}

func (c *capiBackend) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	GinkgoHelper()

	switch platform {
	case configv1.AWSPlatformType:
		return c.createAWSMachineTemplate(ctx, client, params)
	case configv1.AzurePlatformType:
		return nil, fmt.Errorf("azure machine template creation not yet implemented")
	case configv1.GCPPlatformType:
		return nil, fmt.Errorf("gcp machine template creation not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

func (c *capiBackend) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	GinkgoHelper()

	obj, ok := template.(runtimeclient.Object)
	Expect(ok).To(BeTrue(), "Should be runtimeclient.Object, got %T", template)

	Eventually(func() error {
		return client.Delete(ctx, obj)
	}, framework.WaitShort, framework.RetryShort).Should(SatisfyAny(
		Succeed(),
		WithTransform(apierrors.IsNotFound, BeTrue()),
	), "Should delete MachineTemplate of type %T %s/%s successfully or MachineTemplate should not be found",
		template, obj.GetNamespace(), obj.GetName())

	return nil
}

// createAWSMachineTemplate creates AWS machine template.
func (c *capiBackend) createAWSMachineTemplate(ctx context.Context, client runtimeclient.Client, params BackendMachineTemplateParams) (interface{}, error) {
	GinkgoHelper()

	awsMachineSpec := c.getDefaultAWSCAPIMachineSpec(ctx, client)

	awsMachineTemplate := &awsv1.AWSMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: framework.ClusterAPINamespace,
		},
		Spec: awsv1.AWSMachineTemplateSpec{
			Template: awsv1.AWSMachineTemplateResource{
				Spec: *awsMachineSpec,
			},
		},
	}

	// Apply custom configuration from params.Spec if provided
	if params.Spec != nil {
		// Apply the configuration directly to the template before creating it
		templateConfig, ok := params.Spec.(*config.MachineTemplateConfig)
		Expect(ok).To(BeTrue(), "Should be *config.MachineTemplateConfig, got %T", params.Spec)

		configErr := config.ConfigureMachineTemplate(awsMachineTemplate, templateConfig)
		Expect(configErr).NotTo(HaveOccurred(), "Should apply custom configuration to template before creation")
	}

	Eventually(func() error {
		return client.Create(ctx, awsMachineTemplate)
	}, framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should create AWS machine template %s", awsMachineTemplate.Name)

	return awsMachineTemplate, nil
}

// getDefaultAWSCAPIMachineSpec gets default AWS CAPI machine specification.
// Uses the first AWSMachineTemplate if exists, otherwise creates spec from worker MachineSet.
func (c *capiBackend) getDefaultAWSCAPIMachineSpec(ctx context.Context, client runtimeclient.Client) *awsv1.AWSMachineSpec {
	GinkgoHelper()
	// Find existing AWS machine templates
	awsTemplateList := &awsv1.AWSMachineTemplateList{}

	Eventually(komega.List(awsTemplateList, runtimeclient.InNamespace(framework.ClusterAPINamespace)), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should list AWS machine templates")

	if len(awsTemplateList.Items) == 0 {
		// If no existing CAPI templates found, create spec from worker MachineSet AMI
		GinkgoWriter.Println("No CAPI AWSMachineTemplate found, creating spec from worker MachineSet")
		return c.createDefaultAWSCAPIMachineSpec(ctx, client)
	}

	// Use the first template's specification as default
	return &awsTemplateList.Items[0].Spec.Template.Spec
}

// createDefaultAWSCAPIMachineSpec creates default AWS CAPI machine spec.
func (c *capiBackend) createDefaultAWSCAPIMachineSpec(ctx context.Context, cl runtimeclient.Client) *awsv1.AWSMachineSpec {
	GinkgoHelper()
	// Get worker MachineSet to extract AMI and other config
	workers, err := framework.GetWorkerMachineSets(ctx, cl)
	Expect(err).ToNot(HaveOccurred(), "Should list worker MachineSets")
	Expect(workers).NotTo(BeEmpty(), "Should find worker MachineSets to determine default machine configuration")

	// Extract AMI and configuration from first worker MachineSet
	workerMS := workers[0]
	Expect(workers[0].Spec.Template.Spec.ProviderSpec.Value).NotTo(BeNil(), "Should have ProviderSpec in worker MachineSet")

	var providerSpec machinev1.AWSMachineProviderConfig

	err = yaml.Unmarshal(workerMS.Spec.Template.Spec.ProviderSpec.Value.Raw, &providerSpec)
	Expect(err).NotTo(HaveOccurred(), "Should unmarshal provider spec")

	// Build CAPI spec from worker MachineSet config
	capiSpec := &awsv1.AWSMachineSpec{
		InstanceType: providerSpec.InstanceType,
		AMI: awsv1.AMIReference{
			ID: providerSpec.AMI.ID,
		},
		Ignition: &awsv1.Ignition{
			Version:     "3.4",
			StorageType: awsv1.IgnitionStorageTypeOptionUnencryptedUserData,
		},
		Subnet: &awsv1.AWSResourceReference{
			Filters: []awsv1.Filter{
				{
					Name:   "tag:Name",
					Values: []string{"*worker*"},
				},
			},
		},
		AdditionalSecurityGroups: []awsv1.AWSResourceReference{
			{
				Filters: []awsv1.Filter{
					{
						Name:   "tag:Name",
						Values: []string{"*worker*"},
					},
				},
			},
		},
	}

	GinkgoWriter.Printf("Created CAPI spec from worker MachineSet %s: AMI=%s, InstanceType=%s\n",
		workerMS.Name, *providerSpec.AMI.ID, providerSpec.InstanceType)

	return capiSpec
}

// setInfrastructureRef sets InfrastructureRef based on template type.
func (c *capiBackend) setInfrastructureRef(machineSpec *clusterv1beta1.MachineSpec, template interface{}) {
	GinkgoHelper()

	switch t := template.(type) {
	case *awsv1.AWSMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta2",
			Kind:       "AWSMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	case *azurev1.AzureMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "AzureMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	case *gcpv1.GCPMachineTemplate:
		machineSpec.InfrastructureRef = corev1.ObjectReference{
			APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
			Kind:       "GCPMachineTemplate",
			Name:       t.Name,
			Namespace:  t.Namespace,
		}
	default:
		// This should never happen as template types are validated during creation
		Expect(false).To(BeTrue(), "Should have supported template type for infrastructure ref, got %T", template)
	}
}
