package infra

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

var nodeDrainLabels = map[string]string{
	framework.WorkerNodeRoleLabel: "",
	"node-draining-test":          string(uuid.NewUUID()),
}

func replicationControllerWorkload(namespace string) *corev1.ReplicationController {
	var replicas int32 = 20

	return &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pdb-workload",
			Namespace: namespace,
		},
		Spec: corev1.ReplicationControllerSpec{
			Replicas: &replicas,
			Selector: map[string]string{
				"app": "nginx",
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "nginx",
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "work",
							Image:   "registry.ci.openshift.org/openshift/origin-v4.0:base",
							Command: []string{"sleep", "10h"},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("40m"),
									"memory": resource.MustParse("50Mi"),
								},
							},
						},
					},
					NodeSelector: nodeDrainLabels,
					Tolerations: []corev1.Toleration{
						{
							Key:      "kubemark",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}
}

func podDisruptionBudget(namespace string) *policyv1.PodDisruptionBudget {
	maxUnavailable := intstr.FromInt(1)

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pdb",
			Namespace: namespace,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			MaxUnavailable: &maxUnavailable,
		},
	}
}

func invalidMachinesetWithEmptyProviderConfig() *machinev1.MachineSet {
	var oneReplicas int32 = 1

	return &machinev1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-machineset",
			Namespace: framework.MachineAPINamespace,
		},
		Spec: machinev1.MachineSetSpec{
			Replicas: &oneReplicas,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"little-kitty": "i-am-little-kitty",
				},
			},
			Template: machinev1.MachineTemplateSpec{
				ObjectMeta: machinev1.ObjectMeta{
					Labels: map[string]string{
						"big-kitty": "i-am-bit-kitty",
					},
				},
				Spec: machinev1.MachineSpec{
					// Empty providerSpec!!! we don't want to provision real instances.
					// Just to observe how many machine replicas get created.
					ProviderSpec: machinev1.ProviderSpec{},
				},
			},
		},
	}
}

func deleteObject(client runtimeclient.Client, obj runtimeclient.Object) error {
	cascadeDelete := metav1.DeletePropagationForeground

	return client.Delete(context.TODO(), obj, &runtimeclient.DeleteOptions{
		PropagationPolicy: &cascadeDelete,
	})
}

func deleteObjects(client runtimeclient.Client, delObjects map[string]runtimeclient.Object) error {
	// Remove resources
	for _, obj := range delObjects {
		if err := deleteObject(client, obj); err != nil {
			klog.Errorf("[cleanup] error deleting object: %v", err)
			return err
		}
	}

	return nil
}

var _ = Describe("Managed cluster should", framework.LabelMachines, func() {
	var client runtimeclient.Client
	var ctx context.Context
	var machineSet *machinev1.MachineSet
	var machineSetParams framework.MachineSetParams

	var gatherer *gatherer.StateGatherer

	BeforeEach(func() {
		var err error

		ctx = framework.GetContext()

		gatherer, err = framework.NewGatherer()
		Expect(err).ToNot(HaveOccurred(), "StateGatherer should be able to be created")

		client, err = framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Controller-runtime client should be able to be created")

		// Reset the machineSet between each test
		machineSet = nil

		// Make sure to clean up the resources we created
		DeferCleanup(func() {
			if machineSet != nil {
				By("Deleting the new MachineSet")
				Expect(client.Delete(ctx, machineSet)).To(Succeed(), "MachineSet should be able to be deleted")
				framework.WaitForMachineSetsDeleted(ctx, client, machineSet)
			}
		})
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "StateGatherer should be able to gather resources")
		}
	})

	When("machineset has one replica", func() {
		BeforeEach(func() {
			var err error
			machineSetParams = framework.BuildMachineSetParams(ctx, client, 1)

			By("Creating a new MachineSet")
			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")

			framework.WaitForMachineSet(ctx, client, machineSet.GetName())
		})

		// Machines required for test: 1
		// Reason: This test works on a single machine and its node.
		It("have ability to additively reconcile taints from machine to nodes", func() {
			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(ctx, client, &selector)
			Expect(err).ToNot(HaveOccurred(), "Listing Machines should succeed")
			Expect(machines).ToNot(BeEmpty(), "The list of Machines should not be empty")

			machine := machines[0]
			By(fmt.Sprintf("getting machine %q", machine.Name))

			node, err := framework.GetNodeForMachine(ctx, client, machine)
			Expect(err).NotTo(HaveOccurred(), "Should be able to retrieve Node from its Machine")
			By(fmt.Sprintf("getting the backed node %q", node.Name))

			nodeTaint := corev1.Taint{
				Key:    "not-from-machine",
				Value:  "true",
				Effect: corev1.TaintEffectNoSchedule,
			}
			By(fmt.Sprintf("updating node %q with taint: %v", node.Name, nodeTaint))
			for {
				node.Spec.Taints = append(node.Spec.Taints, nodeTaint)
				err = client.Update(ctx, node)
				if !apierrors.IsConflict(err) {
					break
				}
			}
			Expect(err).NotTo(HaveOccurred(), "Node update should succeed")

			machineTaint := corev1.Taint{
				Key:    fmt.Sprintf("from-machine-%v", string(uuid.NewUUID())),
				Value:  "true",
				Effect: corev1.TaintEffectNoSchedule,
			}
			By(fmt.Sprintf("updating machine %q with taint: %v", machine.Name, machineTaint))
			for {
				machine.Spec.Taints = append(machine.Spec.Taints, machineTaint)
				err = client.Update(ctx, machine)
				if !apierrors.IsConflict(err) {
					break
				}
			}
			Expect(err).NotTo(HaveOccurred(), "Machine update should succeed")

			var expectedTaints = sets.NewString("not-from-machine", machineTaint.Key)
			Eventually(func() bool {
				klog.Info("Getting node from machine again for verification of taints")
				node, err := framework.GetNodeForMachine(ctx, client, machine)
				if err != nil {
					return false
				}
				var observedTaints = sets.NewString()
				for _, taint := range node.Spec.Taints {
					observedTaints.Insert(taint.Key)
				}
				if !expectedTaints.Difference(observedTaints).HasAny("not-from-machine", machineTaint.Key) {
					klog.Infof("Expected : %v, observed %v , difference %v, ", expectedTaints, observedTaints, expectedTaints.Difference(observedTaints))
					return true
				}
				klog.Infof("Did not find all expected taints on the node. Missing: %v", expectedTaints.Difference(observedTaints))

				return false
			}, framework.WaitMedium, 5*time.Second).Should(BeTrue(), "Should find all the expected taints on the Node")
		})

	})

	When("machineset has 2 replicas", func() {
		BeforeEach(func() {
			var err error
			machineSetParams = framework.BuildMachineSetParams(ctx, client, 2)

			By("Creating a new MachineSet")
			machineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "MachineSet creation should succeed")

			framework.WaitForMachineSet(ctx, client, machineSet.GetName())
		})

		// Machines required for test: 2
		// Reason: We want to test that all machines get replaced when we delete them.
		It("recover from deleted worker machines", func() {
			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(ctx, client, &selector)
			Expect(err).ToNot(HaveOccurred(), "Listing Machines should succeed")
			Expect(machines).ToNot(BeEmpty(), "The list of Machines should not be empty")

			By("deleting all machines")
			Expect(framework.DeleteMachines(ctx, client, machines...)).To(Succeed(), "Should be able to delete all Machines")
			framework.WaitForMachinesDeleted(client, machines...)

			framework.WaitForMachineSet(ctx, client, machineSet.GetName())
		})

		// Machines required for test: 4
		// Reason: MachineSet scales 2->0 and MachineSet2 scales 0->2. Changing to scaling 1->0 and 0->1 might not test this thoroughly.
		It("grow and decrease when scaling different machineSets simultaneously", framework.LabelPeriodic, func() {
			By("Creating a second MachineSet") // Machineset 1 can start with 1 replica
			machineSetParams := framework.BuildMachineSetParams(ctx, client, 0)
			machineSet2, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "Should be able to create MachineSet")

			// Make sure second machineset gets deleted anyway
			defer func() {
				By("Deleting the second MachineSet")
				Expect(deleteObject(client, machineSet2)).To(Succeed(), "Should be able to delete MachineSet")
				framework.WaitForMachineSetsDeleted(ctx, client, machineSet2)
			}()

			framework.WaitForMachineSet(ctx, client, machineSet2.GetName())

			Expect(framework.ScaleMachineSet(machineSet.GetName(), 0)).To(Succeed(), "Should be able to scale down MachineSet")
			Expect(framework.ScaleMachineSet(machineSet2.GetName(), 1)).To(Succeed(), "Should be able to scale MachineSet")

			framework.WaitForMachineSet(ctx, client, machineSet.GetName())
			framework.WaitForMachineSet(ctx, client, machineSet2.GetName())
		})

		// Machines required for test: 2 (3 but it gets deleted without waiting for it to be ready)
		// Reason: Pods are spread across both machines. After one is deleted, the pods are rescheduled onto the other machine.
		It("drain node before removing machine resource", func() {
			By("Create a machine for node about to be drained")

			selector := machineSet.Spec.Selector
			machines, err := framework.GetMachines(ctx, client, &selector)
			Expect(err).ToNot(HaveOccurred(), "Should be able to List Machines")
			Expect(len(machines)).To(BeNumerically(">=", 2), "Should have found at least 2 Machines")

			// Add node draining labels to params
			for k, v := range nodeDrainLabels {
				machineSetParams.Labels[k] = v
			}

			machines[0].Spec.ObjectMeta.Labels = machineSetParams.Labels
			machines[1].Spec.ObjectMeta.Labels = machineSetParams.Labels

			Expect(client.Update(context.TODO(), machines[0])).To(Succeed(), "Should be able to update Machine")

			Expect(client.Update(context.TODO(), machines[1])).To(Succeed(), "Should be able to update Machine")

			// Make sure RC and PDB get deleted anyway
			delObjects := make(map[string]runtimeclient.Object)

			defer func() {
				Expect(deleteObjects(client, delObjects)).To(Succeed(), "Should be able to cleanup test objects")
			}()

			By("Creating RC with workload")

			// Use the openshift-machine-api namespace as it is excluded from
			// Pod security admission checks.
			namespace := framework.MachineAPINamespace

			rc := replicationControllerWorkload(namespace)
			Expect(client.Create(context.TODO(), rc)).To(Succeed(), "Should be able to create ReplicationController")
			delObjects["rc"] = rc

			By("Creating PDB for RC")
			pdb := podDisruptionBudget(namespace)
			Expect(client.Create(context.TODO(), pdb)).To(Succeed(), "Should be able to create PodDisruptionBudget")
			delObjects["pdb"] = pdb

			By("Wait until all replicas are ready")
			Expect(framework.WaitUntilAllRCPodsAreReady(ctx, client, rc)).To(Succeed(), "Should wait until all Pod replicas are ready")

			// TODO(jchaloup): delete machine that has at least half of the RC pods

			// All pods are distributed evenly among all nodes so it's fine to drain
			// random node and observe reconciliation of pods on the other one.
			By("Delete machine to trigger node draining")
			Expect(client.Delete(context.TODO(), machines[0])).To(Succeed(), "Should be able to Delete Machine")

			// We still should be able to list the machine as until rc.replicas-1 are running on the other node
			By("Observing and verifying node draining")
			drainedNodeName, err := framework.VerifyNodeDraining(ctx, client, machines[0], rc)
			Expect(err).NotTo(HaveOccurred(), "Should verify Node was drained")

			By("Validating the machine is deleted")
			framework.WaitForMachinesDeleted(client, machines[0])

			By("Validate underlying node corresponding to machine1 is removed as well")
			Expect(framework.WaitUntilNodeDoesNotExists(ctx, client, drainedNodeName)).To(Succeed(), "Should wait until Node does not exit")
		})
	})

	// Machines required for test: 0
	// Reason: The machineSet creation is rejected by the webhook.
	It("reject invalid machinesets", func() {
		client, err := framework.LoadClient()
		Expect(err).ToNot(HaveOccurred(), "Controller-runtime client should be able to be created")
		// Only run on platforms that have webhooks
		clusterInfra, err := framework.GetInfrastructure(ctx, client)
		Expect(err).NotTo(HaveOccurred(), "Should be able to get Infrastructure")
		platform := clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.AzurePlatformType, configv1.GCPPlatformType, configv1.VSpherePlatformType, configv1.PowerVSPlatformType, configv1.NutanixPlatformType:
			// Do Nothing
		default:
			Skip(fmt.Sprintf("Platform %s does not have webhooks, skipping.", platform))
		}

		By("Creating invalid machineset")
		invalidMachineSet := invalidMachinesetWithEmptyProviderConfig()
		expectedAdmissionWebhookErr := "admission webhook \"default.machineset.machine.openshift.io\" denied the request: providerSpec.value: Required value: a value must be provided"

		Expect(client.Create(context.TODO(), invalidMachineSet)).To(MatchError(expectedAdmissionWebhookErr), "Should fail to create invalid MachineSet")
	})
})
