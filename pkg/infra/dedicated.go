package infra

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	mapiv1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	awsproviderconfigv1 "sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1beta1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("[Feature:Machines] Running on Dedicated", func() {
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

		clusterInfra, err := framework.GetInfrastructure(client)
		Expect(err).NotTo(HaveOccurred())
		platform = clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not support Dedicated, skipping.", platform))
		}

		By("Creating a Dedicated backed MachineSet", func() {
			machineSetParams = framework.BuildMachineSetParams(client, 3)
			Expect(setDedicatedOnProviderSpec(platform, machineSetParams, "")).To(Succeed())

			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			delObjects[machineSet.Name] = machineSet

			framework.WaitForMachineSet(client, machineSet.GetName())
		})
	})

	AfterEach(func() {
		Expect(deleteObjects(client, delObjects)).To(Succeed())
	})

	It("should successfully create dedicated instances", func() {
		By("Creating a Dedicated backed MachineSet", func() {
			machineSetParams = framework.BuildMachineSetParams(client, 3)
			Expect(setDedicatedOnProviderSpec(platform, machineSetParams, "")).To(Succeed())

			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			delObjects[machineSet.Name] = machineSet

			framework.WaitForMachineSet(client, machineSet.GetName())
		})
	})
})

func setDedicatedOnProviderSpec(platform configv1.PlatformType, params framework.MachineSetParams, maxPrice string) error {
	switch platform {
	case configv1.AWSPlatformType:
		return setDedicatedOnAWSProviderSpec(params)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

func setDedicatedOnAWSProviderSpec(params framework.MachineSetParams) error {
	spec := awsproviderconfigv1.AWSMachineProviderConfig{}

	err := json.Unmarshal(params.ProviderSpec.Value.Raw, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshalling providerspec: %v", err)
	}

	spec.Placement.Tenancy = awsproviderconfigv1.DedicatedTenancy

	params.ProviderSpec.Value.Raw, err = json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("error marshalling providerspec: %v", err)
	}

	return nil
}
