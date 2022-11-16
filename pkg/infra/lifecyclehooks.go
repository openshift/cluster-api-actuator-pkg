package infra

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	lifecycleHooksTestLabel           = "test.lifecyclehooks.label"
	lifecycleHooksPodLabel            = "e2e.lifecyclehooks.pod.label"
	lifecyclehooksWorkerNodeRoleLabel = "machine.openshift.io/lifecyclehooks-e2e-worker"
	lifecycleWorkloadJobName          = "e2e-lifecyclehooks-workload"
	pollingInterval                   = 3 * time.Second
)

var _ = Describe("Lifecycle Hooks should", framework.LabelMachines, func() {
	var client runtimeclient.Client
	var machineSet *machinev1.MachineSet
	var workload *batchv1.Job
	var pod corev1.Pod

	BeforeEach(func() {
		var err error

		By("Creating the machineset")
		client, err = framework.LoadClient()
		// Build machine set parameters
		expectedReplicas := 1
		machineSetParams := framework.BuildMachineSetParams(client, expectedReplicas)
		// Create a label for node and add to machine set parameters
		machineSetParams.Labels[lifecyclehooksWorkerNodeRoleLabel] = ""
		// Create machine set
		machineSet, err = framework.CreateMachineSet(client, machineSetParams)
		Expect(err).ToNot(HaveOccurred())
		// Wait for machine to be running
		framework.WaitForMachineSet(client, machineSet.GetName())

		By("Running a workload on the machine")
		// Run a pod on this machine
		workloadMemRequest := resource.MustParse("100m")
		workload = framework.NewWorkLoad(int32(expectedReplicas), workloadMemRequest,
			lifecycleWorkloadJobName, lifecycleHooksTestLabel, lifecyclehooksWorkerNodeRoleLabel, lifecycleHooksPodLabel)
		Expect(client.Create(context.Background(), workload)).To(Succeed(), "Could not create workload job")

		By("Waiting for job pod to start running on machine.")
		Eventually(func() (bool, error) {
			jobPodList, err := framework.GetPods(client, map[string]string{lifecycleHooksPodLabel: ""})
			if err != nil {
				return false, err
			}
			if len(jobPodList.Items) == expectedReplicas {
				pod = jobPodList.Items[0]
				return pod.Status.Phase == corev1.PodRunning, nil
			}
			return false, nil
		}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Pod did not start running on machine")
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() == true {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
		}

		By("Deleting the machineset")
		cascadeDelete := metav1.DeletePropagationForeground
		Expect(client.Delete(context.Background(), machineSet, &runtimeclient.DeleteOptions{
			PropagationPolicy: &cascadeDelete,
		})).To(Succeed())

		By("Waiting for the MachineSet to be deleted...")
		framework.WaitForMachineSetDelete(client, machineSet)

		By("Deleting workload job")
		Expect(client.Delete(context.Background(), workload, &runtimeclient.DeleteOptions{
			PropagationPolicy: &cascadeDelete,
		})).To(Succeed())
	})

	It("pause lifecycle actions when present", func() {
		machines, err := framework.GetMachinesFromMachineSet(client, machineSet)
		Expect(err).ToNot(HaveOccurred())
		Expect(machines).To(HaveLen(1), "There should be only one machine")
		machine := machines[0]
		podKey := types.NamespacedName{Namespace: pod.Namespace, Name: pod.Name}
		machineKey := types.NamespacedName{Namespace: machine.Namespace, Name: machine.Name}

		By("Setting lifecycle hooks on the machine")
		predrainHook := machinev1.LifecycleHook{
			Name:  "cluster-api-actuator-pkg/pre-drainHook",
			Owner: "cluster-api-actuator-pkg",
		}
		preterminateHook := machinev1.LifecycleHook{
			Name:  "cluster-api-actuator-pkg/pre-terminateHook",
			Owner: "cluster-api-actuator-pkg",
		}
		Eventually(func() (bool, error) {
			if err = client.Get(context.Background(), machineKey, machine); err != nil {
				return false, err
			}
			machine.Spec.LifecycleHooks.PreDrain = []machinev1.LifecycleHook{predrainHook}
			machine.Spec.LifecycleHooks.PreTerminate = []machinev1.LifecycleHook{preterminateHook}
			if err := client.Update(context.Background(), machine); err != nil {
				return false, err
			}
			return true, nil
		}, framework.WaitShort, pollingInterval).Should(BeTrue(),
			"Could not add lifecycle hooks to machine")

		By("Deleting the machine")
		// Delete the machine by scaling down the machineset to zero
		framework.ScaleMachineSet(machineSet.Name, 0)

		By("Checking that workload pod is running on machine")
		// pre-drain hook should prevent pod from being evicted
		Eventually(func() (bool, error) {
			if err := client.Get(context.Background(), podKey, &pod); err != nil {
				return false, err
			}
			if err := client.Get(context.Background(), machineKey, machine); err != nil {
				return false, err
			}
			// Check that machine drainable false condition is set
			for _, condition := range machine.Status.Conditions {
				if condition.Type == machinev1.MachineDrainable && condition.Status == corev1.ConditionFalse {
					return pod.Status.Phase == corev1.PodRunning, nil
				}
			}
			return false, nil
		}, framework.WaitMedium, pollingInterval).Should(BeTrue(),
			"Workload pod was evicted from the machine or drainable condition is not set")

		By("Removing pre-drain hook")
		Eventually(func() (bool, error) {
			if err := client.Get(context.Background(), machineKey, machine); err != nil {
				return false, err
			}
			machine.Spec.LifecycleHooks.PreDrain = []machinev1.LifecycleHook{}
			if err := client.Update(context.Background(), machine); err != nil {
				return false, err
			}
			return true, nil
		}, framework.WaitShort, pollingInterval).Should(BeTrue(), "Could not delete pre-drain hook")

		By("Checking that workload pod is evicted from the machine")
		// Check that pod is evicted, but machine is still present
		Eventually(func() bool {
			err = client.Get(context.Background(), podKey, &pod)
			return apierrors.IsNotFound(err)
		}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Pod was not evicted from machine")
		Eventually(func() (bool, error) {
			if err := client.Get(context.Background(), machineKey, machine); err != nil {
				return false, err
			}
			// Machine phase should be "Deleting"
			// but pre-terminate hook should prevent deletion and set terminable false condition
			for _, condition := range machine.Status.Conditions {
				if condition.Type == machinev1.MachineTerminable && condition.Status == corev1.ConditionFalse {
					return *machine.Status.Phase == "Deleting", nil
				}
			}
			return false, nil
		}, framework.WaitMedium, pollingInterval).Should(BeTrue(),
			"Machine was deleted or terminable condition is not set")

		By("Removing pre-terminate hook")
		Eventually(func() (bool, error) {
			if err := client.Get(context.Background(), machineKey, machine); err != nil {
				return false, err
			}
			machine.Spec.LifecycleHooks.PreTerminate = []machinev1.LifecycleHook{}
			if err = client.Update(context.Background(), machine); err != nil {
				return false, err
			}
			return true, nil
		}, framework.WaitShort, pollingInterval).Should(BeTrue(),
			"Could not delete pre-termiante hook")

		By("Checking that machine is deleted")
		Eventually(func() bool {
			err = client.Get(context.Background(), machineKey, machine)
			return apierrors.IsNotFound(err)
		}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Machine was not deleted")
	})
})
