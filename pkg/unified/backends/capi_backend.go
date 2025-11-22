package backends

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	awsv1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	azurev1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	gcpv1 "sigs.k8s.io/cluster-api-provider-gcp/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/unified/config"
)

const (
	// DefaultAMIID is a example AMI ID for testing.
	DefaultAMIID = "ami-0c55b159cbfafe1d0"
)

// capiBackend implements the MachineBackend interface for CAPI backend.
type capiBackend struct {
	backendType      BackendType
	authoritativeAPI BackendType
}

func (c *capiBackend) GetBackendType() BackendType      { return c.backendType }
func (c *capiBackend) GetAuthoritativeAPI() BackendType { return c.authoritativeAPI }

func (c *capiBackend) CreateMachineSet(ctx context.Context, client runtimeclient.Client, params BackendMachineSetParams) (interface{}, error) {
	infra, err := framework.GetInfrastructure(ctx, client)
	Expect(err).NotTo(HaveOccurred(), "Should get infrastructure global object")
	Expect(infra.Status.InfrastructureName).ShouldNot(BeEmpty(), "Should have infrastructure name on Infrastructure.Status")

	clusterName := infra.Status.InfrastructureName
	userDataSecret := "worker-user-data"
	machineSet := &clusterv1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        params.Name,
			Namespace:   framework.ClusterAPINamespace,
			Labels:      params.Labels,
			Annotations: params.Annotations,
		},
		Spec: clusterv1.MachineSetSpec{
			ClusterName: clusterName,
			Replicas:    &params.Replicas,
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{
				"cluster.x-k8s.io/set-name":     params.Name,
				"cluster.x-k8s.io/cluster-name": clusterName,
			}},

			Template: clusterv1.MachineTemplateSpec{
				ObjectMeta: clusterv1.ObjectMeta{Labels: framework.MergeLabels(params.Labels, map[string]string{
					"cluster.x-k8s.io/set-name":     params.Name,
					"cluster.x-k8s.io/cluster-name": clusterName,
					framework.WorkerNodeRoleLabel:   "",
				})},
				Spec: clusterv1.MachineSpec{
					Bootstrap: clusterv1.Bootstrap{
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

	Eventually(client.Create(ctx, machineSet), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should create CAPI MachineSet %s", machineSet.Name)

	return machineSet, nil
}

func (c *capiBackend) DeleteMachineSet(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*clusterv1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.DeleteCAPIMachineSets(ctx, client, ms)

	return nil
}

func (c *capiBackend) WaitForMachineSetDeleted(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*clusterv1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.WaitForCAPIMachineSetsDeleted(ctx, client, ms)

	return nil
}

func (c *capiBackend) WaitForMachinesRunning(ctx context.Context, client runtimeclient.Client, machineSet interface{}) error {
	ms, ok := machineSet.(*clusterv1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	framework.WaitForCAPIMachinesRunning(ctx, client, ms.Name)

	return nil
}

func (c *capiBackend) GetMachineSetStatus(ctx context.Context, client runtimeclient.Client, machineSet interface{}) (*MachineSetStatus, error) {
	ms, ok := machineSet.(*clusterv1.MachineSet)
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
	_, ok := machineSet.(*clusterv1.MachineSet)
	Expect(ok).To(BeTrue(), "Should be CAPI MachineSet, got %T", machineSet)

	// TODO: Implement node query when needed. Currently not required by any test scenarios.
	return []corev1.Node{}, nil
}

func (c *capiBackend) CreateMachineTemplate(ctx context.Context, client runtimeclient.Client, platform configv1.PlatformType, params BackendMachineTemplateParams) (interface{}, error) {
	switch platform {
	case configv1.AWSPlatformType:
		return c.createAWSMachineTemplate(ctx, client, params)
	case configv1.AzurePlatformType:
		Fail("Azure machine template creation not yet implemented")
	case configv1.GCPPlatformType:
		Fail("GCP machine template creation not yet implemented")
	default:
		Fail(fmt.Sprintf("Unsupported platform: %s", platform))
	}

	return nil, fmt.Errorf("unreachable")
}

func (c *capiBackend) DeleteMachineTemplate(ctx context.Context, client runtimeclient.Client, template interface{}) error {
	obj, ok := template.(runtimeclient.Object)
	if !ok {
		Fail(fmt.Sprintf("Unsupported machine template type: %T", template))
		return nil
	}

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
	awsMachineSpec := c.getDefaultAWSCAPIMachineSpec()

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
		Expect(ok).To(BeTrue(), "Spec should be *config.MachineTemplateConfig, got %T", params.Spec)

		configErr := config.ConfigureMachineTemplate(awsMachineTemplate, templateConfig)
		Expect(configErr).NotTo(HaveOccurred(), "Should apply custom configuration to template before creation")
	}

	Eventually(client.Create(ctx, awsMachineTemplate), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should create AWS machine template %s", awsMachineTemplate.Name)

	return awsMachineTemplate, nil
}

// getDefaultAWSCAPIMachineSpec gets default AWS CAPI machine specification.
func (c *capiBackend) getDefaultAWSCAPIMachineSpec() *awsv1.AWSMachineSpec {
	// Find existing AWS machine templates
	awsTemplateList := &awsv1.AWSMachineTemplateList{}

	Eventually(komega.List(awsTemplateList, runtimeclient.InNamespace(framework.ClusterAPINamespace)), framework.WaitMedium, framework.RetryMedium).Should(Succeed(), "Should list AWS machine templates")

	if len(awsTemplateList.Items) == 0 {
		// If no existing templates found, create a default one
		return c.createDefaultAWSCAPIMachineSpec()
	}

	// Use the first template's specification as default
	return &awsTemplateList.Items[0].Spec.Template.Spec
}

// createDefaultAWSCAPIMachineSpec creates default AWS CAPI machine specification.
func (c *capiBackend) createDefaultAWSCAPIMachineSpec() *awsv1.AWSMachineSpec {
	return &awsv1.AWSMachineSpec{
		InstanceType: "m5.large",
		AMI: awsv1.AMIReference{
			ID: ptr.To(DefaultAMIID),
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
}

// setInfrastructureRef sets InfrastructureRef based on template type.
func (c *capiBackend) setInfrastructureRef(machineSpec *clusterv1.MachineSpec, template interface{}) {
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
		Fail(fmt.Sprintf("Unsupported template type for infrastructure ref: %T", template))
	}
}
