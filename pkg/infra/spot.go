package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	gcproviderconfigv1 "github.com/openshift/cluster-api-provider-gcp/pkg/apis/gcpprovider/v1beta1"
	mapiv1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	awsproviderconfigv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:Machines] Running on Spot", func() {
	var ctx = context.Background()

	var client runtimeclient.Client
	var machineSet *mapiv1.MachineSet
	var machineSetParams framework.MachineSetParams
	var platform configv1.PlatformType

	var delObjects map[string]runtime.Object

	BeforeEach(func() {
		delObjects = make(map[string]runtime.Object)

		var err error
		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred())
		// Only run on AWS
		clusterInfra, err := framework.GetInfrastructure(client)
		Expect(err).NotTo(HaveOccurred())
		platform = clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.GCPPlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not support Spot, skipping.", platform))
		}

		By("Creating a Spot backed MachineSet", func() {
			machineSetParams = framework.BuildMachineSetParams(client, 3)
			Expect(setSpotOnProviderSpec(platform, machineSetParams, "")).To(Succeed())

			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			delObjects[machineSet.Name] = machineSet

			framework.WaitForMachineSet(client, machineSet.GetName())
		})
	})

	AfterEach(func() {
		Expect(deleteObjects(client, delObjects)).To(Succeed())
	})

	It("should label the Machine specs as interruptible", func() {
		selector := machineSet.Spec.Selector
		machines, err := framework.GetMachines(client, &selector)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).To(HaveLen(3))

		for _, machine := range machines {
			Expect(machine.Spec.ObjectMeta.Labels).To(HaveKeyWithValue(machinecontroller.MachineInterruptibleInstanceLabelName, ""))
		}
	})

	It("should deploy a termination handler pod to each instance", func() {
		nodes, err := framework.GetNodesFromMachineSet(client, machineSet)
		Expect(err).ToNot(HaveOccurred())
		Expect(nodes).To(HaveLen(3))

		terminationLabels := map[string]string{
			"api":     "clusterapi",
			"k8s-app": "termination-handler",
		}

		for _, node := range nodes {
			By("Fetching termination Pods running on the Node")
			pods := []corev1.Pod{}
			Eventually(func() ([]corev1.Pod, error) {
				podList := &corev1.PodList{}
				err := client.List(context.Background(), podList, runtimeclient.MatchingLabels(terminationLabels))
				if err != nil {
					return podList.Items, err
				}
				for _, pod := range podList.Items {
					if pod.Spec.NodeName == node.Name {
						pods = append(pods, pod)
					}
				}
				return pods, nil
			}, framework.WaitLong, framework.RetryMedium).ShouldNot(BeEmpty())
			// Termination Pods run in a DaemonSet, should only be 1 per node
			Expect(pods).To(HaveLen(1))
			podKey := runtimeclient.ObjectKey{Namespace: pods[0].Namespace, Name: pods[0].Name}

			By("Ensuring the termination Pod is running and the containers are ready")
			Eventually(func() (bool, error) {
				pod := &corev1.Pod{}
				err := client.Get(context.Background(), podKey, pod)
				if err != nil {
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
			}, framework.WaitLong, framework.RetryMedium).Should(BeTrue())
		}
	})

	It("should terminate a Machine if a termination event is observed", func() {
		By("Deploying a mock metadata application", func() {
			configMap, err := getMetadataMockConfigMap()
			Expect(err).ToNot(HaveOccurred())
			Expect(client.Create(ctx, configMap)).To(Succeed())
			delObjects[configMap.Name] = configMap

			service := getMetadataMockService()
			Expect(client.Create(ctx, service)).To(Succeed())
			delObjects[service.Name] = service

			deployment := getMetadataMockDeployment(platform)
			Expect(client.Create(ctx, deployment)).To(Succeed())
			delObjects[deployment.Name] = deployment

			Expect(framework.IsDeploymentAvailable(client, deployment.Name, deployment.Namespace)).To(BeTrue())
		})

		var machine *mapiv1.Machine
		By("Choosing a Machine to terminate", func() {
			machines, err := framework.GetMachinesFromMachineSet(client, machineSet)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(machines)).To(BeNumerically(">", 0))

			rand.Seed(time.Now().Unix())
			machine = machines[rand.Intn(len(machines))]
			Expect(machine.Status.NodeRef).ToNot(BeNil())
		})

		By("Deploying a job to reroute metadata traffic to the mock", func() {
			serviceAccount := getTerminationSimulatorServiceAccount()
			Expect(client.Create(ctx, serviceAccount)).To(Succeed())
			delObjects[serviceAccount.Name] = serviceAccount

			role := getTerminationSimulatorRole()
			Expect(client.Create(ctx, role)).To(Succeed())
			delObjects[role.Name] = role

			roleBinding := getTerminationSimulatorRoleBinding()
			Expect(client.Create(ctx, roleBinding)).To(Succeed())
			delObjects[roleBinding.Name] = roleBinding

			job := getTerminationSimulatorJob(machine.Status.NodeRef.Name)
			Expect(client.Create(ctx, job)).To(Succeed())
			delObjects[job.Name] = job
		})

		// If the job deploys correctly, the Machine will go away
		framework.WaitForMachinesDeleted(client, machine)
	})
})

func setSpotOnProviderSpec(platform configv1.PlatformType, params framework.MachineSetParams, maxPrice string) error {
	switch platform {
	case configv1.AWSPlatformType:
		return setSpotOnAWSProviderSpec(params, maxPrice)
	case configv1.GCPPlatformType:
		return setSpotOnGCPProviderSpec(params)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

func setSpotOnAWSProviderSpec(params framework.MachineSetParams, maxPrice string) error {
	spec := awsproviderconfigv1.AWSMachineProviderConfig{}

	err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %v", err)
	}

	spec.SpotMarketOptions = &awsproviderconfigv1.SpotMarketOptions{
		MaxPrice: &maxPrice,
	}

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %v", err)
	}

	return nil
}

func setSpotOnGCPProviderSpec(params framework.MachineSetParams) error {
	spec := gcproviderconfigv1.GCPMachineProviderSpec{}

	err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %v", err)
	}

	spec.Preemptible = true

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %v", err)
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
			Replicas: pointer.Int32Ptr(1),
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
	data, err := ioutil.ReadFile("./infra/mock/metadata_mock.go")
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
								Privileged: pointer.BoolPtr(true),
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
