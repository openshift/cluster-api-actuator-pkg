package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

const (
	amiIDMetadataEndpoint = "http://169.254.169.254/latest/meta-data/ami-id"
)

var _ = Describe("MetadataServiceOptions", framework.LabelCloudProviderSpecific, framework.LabelProviderAWS, func() {
	var client runtimeclient.Client
	var clientset *kubernetes.Clientset

	var gatherer *gatherer.StateGatherer
	var ctx context.Context

	toDelete := make([]*machinev1.MachineSet, 0, 3)

	BeforeEach(func() {
		var err error
		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Failed to load client")

		clientset, err = framework.LoadClientset()
		Expect(err).ToNot(HaveOccurred(), "Failed to load clientset")

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "Failed to load gatherer")

		ctx = framework.GetContext()

		platform, err := framework.GetPlatform(ctx, client)
		Expect(err).ToNot(HaveOccurred(), "Failed to get platform")
		if platform != configv1.AWSPlatformType {
			Skip(fmt.Sprintf("skipping AWS specific tests on %s", platform))
		}
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
		}

		Expect(framework.DeleteMachineSets(client, toDelete...)).To(Succeed())
		toDelete = make([]*machinev1.MachineSet, 0, 3)

		framework.WaitForMachineSetsDeleted(ctx, client, toDelete...)
	})

	createMachineSet := func(metadataAuth string) (*machinev1.MachineSet, error) {
		var err error

		By(fmt.Sprintf("Create machine with metadataServiceOptions.authentication %s", metadataAuth))
		machineSetParams := framework.BuildMachineSetParams(ctx, client, 1)
		spec := machinev1.AWSMachineProviderConfig{}
		Expect(json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, &spec)).To(Succeed())

		spec.MetadataServiceOptions.Authentication = machinev1.MetadataServiceAuthentication(metadataAuth)

		machineSetParams.ProviderSpec.Value.Raw, err = json.Marshal(spec)
		Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet parameters")

		mc, err := framework.CreateMachineSet(client, machineSetParams)
		if err != nil {
			return nil, err
		}
		toDelete = append(toDelete, mc)
		framework.WaitForMachineSet(ctx, client, mc.GetName())

		return mc, nil
	}

	assertIMDSavailability := func(machineset *machinev1.MachineSet, responseSubstring string) {
		By("Get node from machineset and spin a curl pod", func() {
			nodes, err := framework.GetNodesFromMachineSet(ctx, client, machineset)
			Expect(err).ToNot(HaveOccurred(), "Failed to get nodes from MachineSet")
			podSpec := corev1.PodSpec{
				HostNetwork: true,
				Containers: []corev1.Container{
					{
						Name:    "curl-metadata",
						Image:   "registry.access.redhat.com/ubi8/ubi-minimal:latest",
						Command: []string{"curl"},
						Args:    []string{"-w 'HTTP_CODE:%{http_code}\n'", "-o /dev/null", "-s", amiIDMetadataEndpoint},
					},
				},
			}
			pod, lastLog, cleanupPod, err := framework.RunPodOnNode(clientset, nodes[0], framework.MachineAPINamespace, podSpec)
			Expect(err).ToNot(HaveOccurred(), "Failed to run pod on node")
			defer func() {
				Expect(cleanupPod()).To(Succeed())
			}()

			By("Ensure curl pod is ready")
			Eventually(func() (bool, error) {
				if err := client.Get(context.Background(), runtimeclient.ObjectKeyFromObject(pod), pod); err != nil {
					return false, err
				}

				switch pod.Status.Phase {
				case corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed:
					return true, nil
				default:
					return false, nil
				}
			}, framework.WaitMedium, framework.RetryShort).Should(BeTrue())

			logs, err := lastLog("curl-metadata", 100, false)
			Expect(err).ToNot(HaveOccurred(), "Failed to get logs from curl pod")
			Expect(logs).Should(ContainSubstring(responseSubstring))
		})
	}

	// Machines required for test: 0
	// No machines are created, because the machineSet is rejected.
	It("should not allow to create machineset with incorrect metadataServiceOptions.authentication", func() {
		_, err := createMachineSet("fooobaar")
		Expect(err).To(HaveOccurred(), "Expected error, shouldn't be able to create machineSet with incorrect metadataServiceOptions.authentication")
		Expect(err.Error()).Should(ContainSubstring("Invalid value: \"fooobaar\": Allowed values are either 'Optional' or 'Required'"))
	})

	// Machines required for test: 1
	// Reason: Deploys a pod on the node, so it requires a machine to be running.
	It("should enforce auth on metadata service if metadataServiceOptions.authentication set to Required", func() {
		machineSet, err := createMachineSet(machinev1.MetadataServiceAuthenticationRequired)
		Expect(err).ToNot(HaveOccurred(), "metadataServiceOptions.authentication set to Required, authentication needed")
		assertIMDSavailability(machineSet, "HTTP_CODE:401")
	})

	// Machines required for test: 1
	// Reason: Deploys a pod on the node, so it requires a machine to be running.
	It("should allow unauthorized requests to metadata service if metadataServiceOptions.authentication is Optional", func() {
		machineSet, err := createMachineSet(machinev1.MetadataServiceAuthenticationOptional)
		Expect(err).ToNot(HaveOccurred(), "Failed to create unauthorized request to metadata service")
		assertIMDSavailability(machineSet, "HTTP_CODE:200")
	})
})
