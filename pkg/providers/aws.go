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
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	amiIdMetadataEndpoint = "http://169.254.169.254/latest/meta-data/ami-id"
)

var _ = Describe("[Feature:Machines] [AWS] MetadataServiceOptions", func() {
	client, err := framework.LoadClient()
	Expect(err).ToNot(HaveOccurred())

	clientset, err := framework.LoadClientset()
	Expect(err).ToNot(HaveOccurred())

	toDelete := make([]*machinev1.MachineSet, 0, 3)

	BeforeEach(func() {
		platform, err := framework.GetPlatform(client)
		Expect(err).ToNot(HaveOccurred())
		if platform != configv1.AWSPlatformType {
			Skip(fmt.Sprintf("skipping AWS specific tests on %s", platform))
		}
	})

	AfterEach(func() {
		Expect(framework.DeleteMachineSets(client, toDelete...)).To(Succeed())
		toDelete = make([]*machinev1.MachineSet, 0, 3)

		framework.WaitForMachineSetsDeleted(client, toDelete...)
	})

	createMachineSet := func(metadataAuth string) (*machinev1.MachineSet, error) {
		By(fmt.Sprintf("Create machine with metadataServiceOptions.authentication %s", metadataAuth))
		machineSetParams := framework.BuildMachineSetParams(client, 1)
		spec := machinev1.AWSMachineProviderConfig{}
		err := json.Unmarshal(machineSetParams.ProviderSpec.Value.Raw, &spec)
		Expect(err).ToNot(HaveOccurred())

		spec.MetadataServiceOptions.Authentication = machinev1.MetadataServiceAuthentication(metadataAuth)

		machineSetParams.ProviderSpec.Value.Raw, err = json.Marshal(spec)
		Expect(err).ToNot(HaveOccurred())

		mc, err := framework.CreateMachineSet(client, machineSetParams)
		if err != nil {
			return nil, err
		}
		toDelete = append(toDelete, mc)
		framework.WaitForMachineSet(client, mc.GetName())
		return mc, nil
	}

	assertIMDSavailability := func(machineset *machinev1.MachineSet, responseSubstring string) {
		By("Get node from machineset and spin a curl pod", func() {
			nodes, err := framework.GetNodesFromMachineSet(client, machineset)
			Expect(err).ToNot(HaveOccurred())
			podSpec := corev1.PodSpec{
				HostNetwork: true,
				Containers: []corev1.Container{
					{
						Name:    "curl-metadata",
						Image:   "registry.access.redhat.com/ubi8/ubi-minimal:latest",
						Command: []string{"curl"},
						Args:    []string{"-v", amiIdMetadataEndpoint},
					},
				},
			}
			pod, lastLog, cleanupPod, err := framework.RunPodOnNode(clientset, nodes[0], framework.MachineAPINamespace, podSpec)
			Expect(err).ToNot(HaveOccurred())
			defer cleanupPod()

			By("Ensure curl pod is ready")
			Eventually(func() (bool, error) {
				err := client.Get(context.Background(), runtimeclient.ObjectKeyFromObject(pod), pod)
				if err != nil {
					return false, err
				}
				switch pod.Status.Phase {
				case corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed:
					return true, nil
				default:
					return false, nil
				}
			}, framework.WaitShort, framework.RetryShort).Should(BeTrue())

			logs, err := lastLog("curl-metadata", 100, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(logs).Should(ContainSubstring(responseSubstring))
		})
	}

	It("should not allow to create machineset with incorrect metadataServiceOptions.authentication", func() {
		_, err := createMachineSet("fooobaar")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("Invalid value: \"fooobaar\": Allowed values are either 'Optional' or 'Required'"))
	})

	It("should enforce auth on metadata service if metadataServiceOptions.authentication set to Required", func() {
		machineSet, err := createMachineSet(machinev1.MetadataServiceAuthenticationRequired)
		Expect(err).ToNot(HaveOccurred())
		assertIMDSavailability(machineSet, "HTTP/1.1 401 Unauthorized")
	})

	It("should allow unauthorized requests to metadata service if metadataServiceOptions.authentication is Optional", func() {
		machineSet, err := createMachineSet(machinev1.MetadataServiceAuthenticationOptional)
		Expect(err).ToNot(HaveOccurred())
		assertIMDSavailability(machineSet, "HTTP/1.1 200 OK")
	})
})
