package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

const (
	// Spot machineSet replicas.
	machinesCount = 1

	// Maximum retries when provisioning a spot machineSet.
	spotMachineSetMaxProvisioningRetryCount = 3
)

var _ = Describe("Running on Spot", framework.LabelMachines, framework.LabelSpot, func() {
	var ctx = context.Background()

	var client runtimeclient.Client
	var machineSet *machinev1.MachineSet
	var platform configv1.PlatformType

	var delObjects map[string]runtimeclient.Object

	var gatherer *gatherer.StateGatherer

	BeforeEach(func() {
		delObjects = make(map[string]runtimeclient.Object)

		// Make sure to clean up the resources we created
		DeferCleanup(func() {
			var machineSets []*machinev1.MachineSet

			for _, obj := range delObjects {
				if machineSet, ok := obj.(*machinev1.MachineSet); ok {
					// Once we delete a MachineSet we should make sure that the
					// all of its machines are deleted as well.
					// Collect MachineSets to wait for.
					machineSets = append(machineSets, machineSet)
				}

				Expect(deleteObject(client, obj)).To(Succeed(), "Should be able to cleanup test objects")
			}

			if len(machineSets) > 0 {
				// Wait for all MachineSets and their Machines to be deleted.
				By("Waiting for MachineSets to be deleted...")
				framework.WaitForMachineSetsDeleted(ctx, client, machineSets...)
			}
		})

		var err error

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "StateGatherer should be able to be created")

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Controller-runtime client should be able to be created")

		platform, err = framework.GetPlatform(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Should be able to get Platform type")
		switch platform {
		case configv1.AWSPlatformType, configv1.AzurePlatformType:
			// Supported platforms, ok to continue.
		case configv1.GCPPlatformType:
			// TODO: GCP relies on the metadata IP for DNS.
			// This test prevents it from accessing the DNS, therefore
			// the termination handler cannot contact the API server
			// to mark the node as terminating.
			// Skip until we can come up with a way to workaround this.
			Skip("Platform GCP is not compatible with this test suite.")
		default:
			Skip(fmt.Sprintf("Platform %s does not support Spot, skipping.", platform))
		}

		By("Creating a Spot backed MachineSet", func() {
			machineSetReady := false
			machineSetParams := framework.BuildMachineSetParams(ctx, client, machinesCount)
			machineSetParamsList, err := framework.BuildAlternativeMachineSetParams(machineSetParams, platform)
			Expect(err).ToNot(HaveOccurred(), "Should be able to build list of MachineSet parameters")
			for i, machineSetParams := range machineSetParamsList {
				if i >= spotMachineSetMaxProvisioningRetryCount {
					// If there are many alternatives, only try the specified number of times
					break
				}
				Expect(setSpotOnProviderSpec(platform, machineSetParams, "")).To(Succeed(), "Should be able to set spot options on ProviderSpec")

				machineSet, err = framework.CreateMachineSet(client, machineSetParams)
				Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")
				delObjects[machineSet.Name] = machineSet

				err = framework.WaitForSpotMachineSet(ctx, client, machineSet.GetName())
				if errors.Is(err, framework.ErrMachineNotProvisionedInsufficientCloudCapacity) {
					By("Trying alternative machineSet because current one could not provision due to insufficient spot capacity")
					// If machineSet cannot scale up due to insufficient capacity, try again with different machineSetParams
					err = framework.DeleteMachineSets(client, machineSet)
					Expect(err).ToNot(HaveOccurred(), "MachineSet should be be able to be deleted")
					delete(delObjects, machineSet.Name)
					framework.WaitForMachineSetsDeleted(ctx, client, machineSet)

					continue
				}
				Expect(err).ToNot(HaveOccurred(), "Error while waiting for all spot MachineSet Machines to be ready")
				machineSetReady = true

				break // MachineSet created successfully
			}
			Expect(machineSetReady).To(BeTrue(), "Failed to create a spot backed MachineSet")
		})
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "StateGatherer should be able to gather resources")
		}
	})

	// Machines required for test: 1
	// Reason: We only deploy the termination simulator pod on one node. Machine draining is tested in other tests.
	It("should handle the spot instances", func() {
		By("should label the Machine specs as interruptible", func() {
			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(ctx, client, &selector)
			Expect(err).ToNot(HaveOccurred(), "Listing Machines should succeed")
			Expect(machines).To(HaveLen(machinesCount), "Should match the expected number of Machines")

			for _, machine := range machines {
				Expect(machine.Spec.ObjectMeta.Labels).To(HaveKeyWithValue(machinecontroller.MachineInterruptibleInstanceLabelName, ""), "Should have the expected spot label on the Machine")
			}
		})

		By("should deploy a termination handler pod to each instance", func() {
			nodes, err := framework.GetNodesFromMachineSet(ctx, client, machineSet)
			Expect(err).ToNot(HaveOccurred(), "Should be able to get Nodes linked to the MachineSet's Machines")
			Expect(nodes).To(HaveLen(machinesCount), "Nodes and Machines count should match")

			terminationLabels := map[string]string{
				"api":     "clusterapi",
				"k8s-app": "termination-handler",
			}

			for _, node := range nodes {
				By("Fetching termination Pods running on the Node")
				pods := []corev1.Pod{}
				Eventually(func() ([]corev1.Pod, error) {
					podList := &corev1.PodList{}

					if err := client.List(context.Background(), podList, runtimeclient.MatchingLabels(terminationLabels)); err != nil {
						return podList.Items, err
					}

					for _, pod := range podList.Items {
						if pod.Spec.NodeName == node.Name {
							pods = append(pods, pod)
						}
					}

					return pods, nil
				}, framework.WaitLong, framework.RetryMedium).ShouldNot(BeEmpty(), "Should find termination pod on Node")
				// Termination Pods run in a DaemonSet, should only be 1 per node
				Expect(pods).To(HaveLen(1), "There should only be one termination handler pod for this Node")
				podKey := runtimeclient.ObjectKey{Namespace: pods[0].Namespace, Name: pods[0].Name}

				By("Ensuring the termination Pod is running and the containers are ready")
				Eventually(func() (bool, error) {
					pod := &corev1.Pod{}

					if err := client.Get(context.Background(), podKey, pod); err != nil {
						return false, err
					}

					if pod.Status.Phase != corev1.PodRunning {
						return false, nil
					}

					// Ensure all containers are ready
					for _, condition := range pod.Status.Conditions {
						if condition.Type == corev1.ContainersReady {
							return condition.Status == corev1.ConditionTrue, nil
						}
					}

					return false, nil
				}, framework.WaitLong, framework.RetryMedium).Should(BeTrue(), "Should find the termination pod Ready")
			}
		})

		By("should terminate a Machine if a termination event is observed", func() {
			By("Deploying a mock metadata application", func() {
				configMap, err := getMetadataMockConfigMap()
				Expect(err).ToNot(HaveOccurred(), "Should load the desired metadata ConfigMap")
				Expect(client.Create(ctx, configMap)).To(Succeed(), "Should be able to create metadata ConfigMap")
				delObjects[configMap.Name] = configMap

				service := getMetadataMockService()
				Expect(client.Create(ctx, service)).To(Succeed(), "Should be able to create metadata Service")
				delObjects[service.Name] = service

				deployment := getMetadataMockDeployment(platform)
				Expect(client.Create(ctx, deployment)).To(Succeed(), "Should be able to create metadata Deployment")
				delObjects[deployment.Name] = deployment

				Expect(framework.IsDeploymentAvailable(ctx, client, deployment.Name, deployment.Namespace)).To(BeTrue(), "Should find an available the metadata Deployment")
			})

			var machine *machinev1.Machine
			By("Choosing a Machine to terminate", func() {
				machines, err := framework.GetMachinesFromMachineSet(ctx, client, machineSet)
				Expect(err).ToNot(HaveOccurred(), "Should be able to get Machines from MachineSet")
				Expect(len(machines)).To(BeNumerically(">", 0), "There should be at least one Machine")

				customRand := rand.New(rand.NewSource(time.Now().Unix()))
				machine = machines[customRand.Intn(len(machines))]
				Expect(machine.Status.NodeRef).ToNot(BeNil(), "Machine should have a linked Node")
			})

			By("Deploying a job to reroute metadata traffic to the mock", func() {
				serviceAccount := getTerminationSimulatorServiceAccount()
				Expect(client.Create(ctx, serviceAccount)).To(Succeed(), "Should be able to create termination simulator ServiceAccount")
				delObjects[serviceAccount.Name] = serviceAccount

				role := getTerminationSimulatorRole()
				Expect(client.Create(ctx, role)).To(Succeed(), "Should be able to create termination simulator Role")
				delObjects[role.Name] = role

				roleBinding := getTerminationSimulatorRoleBinding()
				Expect(client.Create(ctx, roleBinding)).To(Succeed(), "Should be able to create termination simulator RoleBinding")
				delObjects[roleBinding.Name] = roleBinding

				job := getTerminationSimulatorJob(machine.Status.NodeRef.Name)
				Expect(client.Create(ctx, job)).To(Succeed(), "Should be able to create termination simulator Job")
				delObjects[job.Name] = job
			})

			// If the job deploys correctly, the Machine will go away
			By(fmt.Sprintf("Waiting for machine %q to be deleted", machine.Name), func() {
				framework.WaitForMachinesDeleted(client, machine)
			})
		})
	})
})

func setSpotOnProviderSpec(platform configv1.PlatformType, params framework.MachineSetParams, maxPrice string) error {
	switch platform {
	case configv1.AWSPlatformType:
		return setSpotOnAWSProviderSpec(params, maxPrice)
	case configv1.AzurePlatformType:
		return setSpotOnAzureProviderSpec(params, maxPrice)
	case configv1.GCPPlatformType:
		return setSpotOnGCPProviderSpec(params)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

func setSpotOnAWSProviderSpec(params framework.MachineSetParams, maxPrice string) error {
	spec := machinev1.AWSMachineProviderConfig{}

	if err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec); err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %w", err)
	}

	spec.SpotMarketOptions = &machinev1.SpotMarketOptions{}
	if maxPrice != "" {
		spec.SpotMarketOptions.MaxPrice = &maxPrice
	}

	var err error

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %w", err)
	}

	return nil
}

func setSpotOnAzureProviderSpec(params framework.MachineSetParams, maxPrice string) error {
	spec := machinev1.AzureMachineProviderSpec{}

	if err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec); err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %w", err)
	}

	spec.SpotVMOptions = &machinev1.SpotVMOptions{}

	if maxPrice != "" {
		maxPriceQuantity := resource.MustParse(maxPrice)
		spec.SpotVMOptions.MaxPrice = &maxPriceQuantity
	}

	var err error

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %w", err)
	}

	return nil
}

func setSpotOnGCPProviderSpec(params framework.MachineSetParams) error {
	spec := machinev1.GCPMachineProviderSpec{}

	if err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec); err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %w", err)
	}

	spec.Preemptible = true

	var err error

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %w", err)
	}

	return nil
}

const (
	metadataServiceMockName          = "metadata-service-mock"
	metadataServiceMockServiceName   = metadataServiceMockName + "-service"
	metadataServiceMockConfigMapName = metadataServiceMockName + "-configmap"
	metadataServiceMockPort          = 8082
)

func getMetadataMockLabels() map[string]string {
	return map[string]string{
		"app": "metadata-mock",
	}
}

func getMetadataMockDeployment(platform configv1.PlatformType) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceMockName,
			Namespace: framework.MachineAPINamespace,
			Labels:    getMetadataMockLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To[int32](1),
			Selector: &metav1.LabelSelector{
				MatchLabels: getMetadataMockLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: getMetadataMockLabels(),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "metadata-mock",
							Image:   "golang:1.14",
							Command: []string{"/usr/local/go/bin/go"},
							Args: []string{
								"run",
								"/mock/metadata_mock.go",
								fmt.Sprintf("--provider=%s", platform),
								fmt.Sprintf("--listen-addr=0.0.0.0:%d", metadataServiceMockPort),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "GOCACHE",
									Value: "/go/.cache",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "mock-server",
									MountPath: "/mock",
								},
							},
						},
					},
					DNSPolicy: corev1.DNSClusterFirst,
					Volumes: []corev1.Volume{
						{
							Name: "mock-server",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: metadataServiceMockConfigMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getMetadataMockService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceMockServiceName,
			Namespace: framework.MachineAPINamespace,
			Labels:    getMetadataMockLabels(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       metadataServiceMockPort,
					Protocol:   "TCP",
					TargetPort: intstr.FromInt(metadataServiceMockPort),
				},
			},
			Selector:        getMetadataMockLabels(),
			SessionAffinity: corev1.ServiceAffinityNone,
			ClusterIP:       "None",
			Type:            corev1.ServiceTypeClusterIP,
		},
	}
}

func getMetadataMockConfigMap() (*corev1.ConfigMap, error) {
	// Load relative to the test execution directory
	data, err := os.ReadFile("./infra/mock/metadata_mock.go")
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metadataServiceMockConfigMapName,
			Namespace: framework.MachineAPINamespace,
			Labels:    getMetadataMockLabels(),
		},
		BinaryData: map[string][]byte{
			"metadata_mock.go": data,
		},
	}, nil
}

const (
	terminationSimulatorName               = "termination-simulator"
	terminationSimulatorServiceAccountName = terminationSimulatorName + "-service-account"
	terminationSimulatorRoleName           = terminationSimulatorName + "-role"
	terminationSimulatorRoleBindingName    = terminationSimulatorName + "-rolebinding"
)

func getTerminationSimulatorJob(nodeName string) *batchv1.Job {
	script := `apk update && apk add iptables bind-tools;
export SERVICE_IP=$(dig +short ${MOCK_SERVICE_NAME}.${NAMESPACE}.svc.cluster.local);
if [ -z ${SERVICE_IP} ]; then echo "No service IP"; exit 1; fi;
iptables-nft -t nat -A OUTPUT -p tcp -d 169.254.169.254 -j DNAT --to-destination ${SERVICE_IP}:${MOCK_SERVICE_PORT};
iptables-nft -t nat -A POSTROUTING -j MASQUERADE;
ifconfig lo:0 169.254.169.254 up;
echo "Redirected metadata service to ${SERVICE_IP}:${MOCK_SERVICE_PORT}";`

	fileOrCreate := corev1.HostPathFileOrCreate

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      terminationSimulatorName,
			Namespace: framework.MachineAPINamespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "iptables",
							Image:   "alpine:3.12",
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{script},
							Env: []corev1.EnvVar{
								{
									Name:  "NAMESPACE",
									Value: framework.MachineAPINamespace,
								},
								{
									Name:  "MOCK_SERVICE_NAME",
									Value: metadataServiceMockServiceName,
								},
								{
									Name:  "MOCK_SERVICE_PORT",
									Value: fmt.Sprintf("%d", metadataServiceMockPort),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: ptr.To[bool](true),
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN", "NET_RAW"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "xtables-lock",
									MountPath: "/run/xtables.lock",
									ReadOnly:  false,
								},
								{
									Name:      "lib-modules",
									MountPath: "/lib/modules",
									ReadOnly:  true,
								},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					HostNetwork:        true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					NodeName:           nodeName,
					ServiceAccountName: terminationSimulatorServiceAccountName,
					Volumes: []corev1.Volume{
						{
							Name: "xtables-lock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/run/xtables.lock",
									Type: &fileOrCreate,
								},
							},
						},
						{
							Name: "lib-modules",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getTerminationSimulatorServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      terminationSimulatorServiceAccountName,
			Namespace: framework.MachineAPINamespace,
		},
	}
}

func getTerminationSimulatorRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      terminationSimulatorRoleName,
			Namespace: framework.MachineAPINamespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"security.openshift.io"},
				ResourceNames: []string{"privileged"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func getTerminationSimulatorRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      terminationSimulatorRoleBindingName,
			Namespace: framework.MachineAPINamespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     terminationSimulatorRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      terminationSimulatorServiceAccountName,
				Namespace: framework.MachineAPINamespace,
			},
		},
	}
}
