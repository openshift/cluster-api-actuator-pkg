package infra

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	gcproviderconfigv1 "github.com/openshift/cluster-api-provider-gcp/pkg/apis/gcpprovider/v1beta1"
	mapiv1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	machinecontroller "github.com/openshift/machine-api-operator/pkg/controller/machine"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	awsproviderconfigv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	azureproviderconfigv1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:Machines] Running on Spot", func() {
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
		case configv1.AWSPlatformType, configv1.GCPPlatformType, configv1.AzurePlatformType:
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

})

func setSpotOnProviderSpec(platform configv1.PlatformType, params framework.MachineSetParams, maxPrice string) error {
	switch platform {
	case configv1.AWSPlatformType:
		return setSpotOnAWSProviderSpec(params, maxPrice)
	case configv1.GCPPlatformType:
		return setSpotOnGCPProviderSpec(params)
	case configv1.AzurePlatformType:
		return setSpotOnAzureProviderSpec(params, maxPrice)
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

func setSpotOnAzureProviderSpec(params framework.MachineSetParams, maxPrice string) error {
	spec := azureproviderconfigv1.AzureMachineProviderSpec{}

	err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %v", err)
	}

	spec.SpotVMOptions = &azureproviderconfigv1.SpotVMOptions{
		MaxPrice: &maxPrice,
	}

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %v", err)
	}

	return nil
}
