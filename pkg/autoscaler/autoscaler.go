package autoscaler

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"

	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework/gatherer"
)

const (
	autoscalingTestLabel          = "test.autoscaling.label"
	clusterAutoscalerComponent    = "cluster-autoscaler"
	pollingInterval               = 3 * time.Second
	autoscalerWorkerNodeRoleLabel = "machine.openshift.io/autoscaler-e2e-worker"
	workloadJobName               = "e2e-autoscaler-workload"
	machineDeleteAnnotationKey    = "machine.openshift.io/cluster-api-delete-machine"
	deletionCandidateTaintKey     = "DeletionCandidateOfClusterAutoscaler"
	toBeDeletedTaintKey           = "ToBeDeletedByClusterAutoscaler"
)

// Build default CA resource to allow fast scaling up and down
func clusterAutoscalerResource(maxNodesTotal int) *caov1.ClusterAutoscaler {
	tenSecondString := "10s"

	// Choose a time that is at least twice as the sync period
	// and that has high least common multiple to avoid a case
	// when a node is considered to be empty even if there are
	// pods already scheduled and running on the node.
	unneededTimeString := "60s"
	return &caov1.ClusterAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: framework.MachineAPINamespace,
			Labels: map[string]string{
				autoscalingTestLabel: "",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterAutoscaler",
			APIVersion: "autoscaling.openshift.io/v1",
		},
		Spec: caov1.ClusterAutoscalerSpec{
			ScaleDown: &caov1.ScaleDownConfig{
				Enabled:           true,
				DelayAfterAdd:     &tenSecondString,
				DelayAfterDelete:  &tenSecondString,
				DelayAfterFailure: &tenSecondString,
				UnneededTime:      &unneededTimeString,
			},
			ResourceLimits: &caov1.ResourceLimits{
				MaxNodesTotal: pointer.Int32Ptr(int32(maxNodesTotal)),
			},
		},
	}
}

// Build MA resource from targeted machineset
func machineAutoscalerResource(targetMachineSet *machinev1.MachineSet, minReplicas, maxReplicas int32) *caov1beta1.MachineAutoscaler {
	return &caov1beta1.MachineAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("autoscale-%s", targetMachineSet.Name),
			Namespace:    framework.MachineAPINamespace,
			Labels: map[string]string{
				autoscalingTestLabel: "",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "MachineAutoscaler",
			APIVersion: "autoscaling.openshift.io/v1beta1",
		},
		Spec: caov1beta1.MachineAutoscalerSpec{
			MaxReplicas: maxReplicas,
			MinReplicas: minReplicas,
			ScaleTargetRef: caov1beta1.CrossVersionObjectReference{
				Name:       targetMachineSet.Name,
				Kind:       "MachineSet",
				APIVersion: "machine.openshift.io/v1beta1",
			},
		},
	}
}

var _ = Describe("Autoscaler should", framework.LabelAutoscaler, Serial, func() {

	var workloadMemRequest resource.Quantity
	var client runtimeclient.Client
	var gatherer *gatherer.StateGatherer
	var err error
	var cleanupObjects map[string]runtimeclient.Object

	ctx := context.Background()
	cascadeDelete := metav1.DeletePropagationForeground
	deleteObject := func(name string, obj runtimeclient.Object) error {
		klog.Infof("[cleanup] %q (%T)", name, obj)
		return client.Delete(ctx, obj, &runtimeclient.DeleteOptions{
			PropagationPolicy: &cascadeDelete,
		})
	}

	BeforeEach(func() {
		client, err = framework.LoadClient()
		Expect(err).NotTo(HaveOccurred())

		workerNodes, err := framework.GetWorkerNodes(client)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(workerNodes)).To(BeNumerically(">=", 1))

		memCapacity := workerNodes[0].Status.Allocatable[corev1.ResourceMemory]
		Expect(memCapacity).ShouldNot(BeNil())
		Expect(memCapacity.String()).ShouldNot(BeEmpty())
		klog.Infof("Allocatable memory capacity of worker node %q is %s", workerNodes[0].Name, memCapacity.String())

		bytes, ok := memCapacity.AsInt64()
		Expect(ok).Should(BeTrue())

		// 70% - enough that the existing and new nodes will
		// be used, not enough to have more than 1 pod per
		// node.
		workloadMemRequest = resource.MustParse(fmt.Sprintf("%v", 0.7*float32(bytes)))

		// Anything we create we must cleanup
		cleanupObjects = make(map[string]runtimeclient.Object)
	})

	AfterEach(func() {
		var machineSets []*machinev1.MachineSet

		for name, obj := range cleanupObjects {
			if machineSet, ok := obj.(*machinev1.MachineSet); ok {
				// Once we delete a MachineSet we should make sure that the
				// all of its machines are deleted as well.
				// Collect MachineSets to wait for.
				machineSets = append(machineSets, machineSet)
			}

			Expect(deleteObject(name, obj)).To(Succeed())
		}

		if len(machineSets) > 0 {
			// Wait for all MachineSets and their Machines to be deleted.
			By("Waiting for MachineSets to be deleted...")
			framework.WaitForMachineSetsDeleted(client, machineSets...)
		}
	})

	Context("use a ClusterAutoscaler that has 100 maximum total nodes count", func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler
		var caEventWatcher *eventWatcher

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(100)
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed())
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler

			By("Starting Cluster Autoscaler event watcher")
			clientset, err := framework.LoadClientset()
			Expect(err).NotTo(HaveOccurred())
			caEventWatcher = newEventWatcher(clientset)
			Expect(caEventWatcher.run()).Should(BeTrue())
			// Log cluster-autoscaler events
			caEventWatcher.onEvent(matchAnyEvent, func(e *corev1.Event) {
				if e.Source.Component == clusterAutoscalerComponent {
					klog.Infof("%s: %s", e.InvolvedObject.Name, e.Message)
				}
			}).enable()
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() == true {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}

			By("Stopping Cluster Autoscaler event watcher")
			caEventWatcher.stop()

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed())
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue())
		})

		// Machines required for test: 2
		// Reason: This tests checks that autoscaler is able to scale from zero. It requires 2 machines to ensure it scales to the correct number of nodes based on the workload size.
		It("It scales from/to zero", func() {
			// Only run in platforms which support autoscaling from/to zero.
			clusterInfra, err := framework.GetInfrastructure(client)
			Expect(err).NotTo(HaveOccurred())

			platform := clusterInfra.Status.PlatformStatus.Type
			switch platform {
			case configv1.AWSPlatformType, configv1.GCPPlatformType, configv1.AzurePlatformType, configv1.OpenStackPlatformType, configv1.VSpherePlatformType, configv1.NutanixPlatformType:
				klog.Infof("Platform is %v", platform)
			default:
				Skip(fmt.Sprintf("Platform %v does not support autoscaling from/to zero, skipping.", platform))
			}

			By("Creating a new MachineSet with 0 replicas")
			machineSetParams := framework.BuildMachineSetParams(client, 0)
			targetedNodeLabel := fmt.Sprintf("%v-scale-from-zero", autoscalerWorkerNodeRoleLabel)
			machineSetParams.Labels[targetedNodeLabel] = ""

			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			cleanupObjects[machineSet.GetName()] = machineSet

			framework.WaitForMachineSet(client, machineSet.GetName())

			expectedReplicas := int32(2)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v",
				machineSet.GetNamespace(), machineSet.GetName(), 0, expectedReplicas))
			asr := machineAutoscalerResource(machineSet, 0, expectedReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed())
			cleanupObjects[asr.GetName()] = asr

			uniqueJobName := fmt.Sprintf("%s-scale-from-zero", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s", uniqueJobName, expectedReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(expectedReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, targetedNodeLabel, "")
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed())

			Eventually(func() bool {
				ms, err := framework.GetMachineSet(client, machineSet.GetName())
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale out. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitMedium, pollingInterval).Should(BeTrue())

			By("Waiting for the machineSet replicas to become nodes")
			framework.WaitForMachineSet(client, machineSet.GetName())

			expectedReplicas = 0
			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed())
			delete(cleanupObjects, workload.Name)
			Eventually(func() bool {
				ms, err := framework.GetMachineSet(client, machineSet.GetName())
				Expect(err).ToNot(HaveOccurred())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale in. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitLong, pollingInterval).Should(BeTrue())
		})

		// Machines required for test: 2
		// Reason: Needs to scale down to minReplicas = 1. Scales 1 -> 2 -> 1.
		It("cleanup deletion information after scale down [Slow]", func() {
			By("Creating MachineSet with 1 replica")
			targetedNodeLabel := fmt.Sprintf("%v-delete-cleanup", autoscalerWorkerNodeRoleLabel)
			machineSetParams := framework.BuildMachineSetParams(client, 1)
			machineSetParams.Labels[targetedNodeLabel] = ""
			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			cleanupObjects[machineSet.GetName()] = machineSet

			By("Waiting for the machineSet to enter Running phase")
			framework.WaitForMachineSet(client, machineSet.GetName())

			expectedReplicas := int32(2)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
				machineSet.GetName(), expectedReplicas))
			asr := machineAutoscalerResource(machineSet, 1, expectedReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed())
			cleanupObjects[asr.GetName()] = asr

			jobReplicas := expectedReplicas
			uniqueJobName := fmt.Sprintf("%s-cleanup-after-scale-down", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, targetedNodeLabel, "")
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed())

			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale out", machineSet.GetName()))
			Eventually(func() (int32, error) {
				current, err := framework.GetMachineSet(client, machineSet.GetName())
				if err != nil {
					return 0, err
				}
				return *current.Spec.Replicas, nil
			}, framework.WaitMedium, pollingInterval).Should(BeEquivalentTo(expectedReplicas))

			By("Waiting for all Machines in the MachineSet to enter Running phase")
			framework.WaitForMachineSet(client, machineSet.GetName())

			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed())
			delete(cleanupObjects, workload.Name)

			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale in", machineSet.GetName()))
			expectedLength := 1
			var machines []*machinev1.Machine
			Eventually(func() (int, error) {
				machines, err = framework.GetMachinesFromMachineSet(client, machineSet)
				if err != nil {
					return 0, err
				}
				return len(machines), nil
			}, framework.WaitLong, pollingInterval).Should(BeEquivalentTo(expectedLength))

			for _, machine := range machines {
				By(fmt.Sprintf("Checking Machine %s for %s annotation", machine.Name, machineDeleteAnnotationKey))
				Eventually(func() (bool, error) {
					m, err := framework.GetMachine(client, machine.Name)
					if err != nil {
						return false, err
					}
					if m.ObjectMeta.Annotations == nil {
						return true, nil
					}
					if _, exists := m.ObjectMeta.Annotations[machineDeleteAnnotationKey]; exists {
						return false, nil
					}
					return true, nil
				}, framework.WaitMedium, pollingInterval).Should(BeTrue())
			}

			for _, machine := range machines {
				if machine.Status.NodeRef == nil {
					continue
				}
				By(fmt.Sprintf("Checking Node %s for %s and %s taints", machine.Status.NodeRef.Name, deletionCandidateTaintKey, toBeDeletedTaintKey))
				Eventually(func() (bool, error) {
					n, err := framework.GetNodeForMachine(client, machine)
					if err != nil {
						return false, err
					}
					for _, t := range n.Spec.Taints {
						if t.Key == deletionCandidateTaintKey || t.Key == toBeDeletedTaintKey {
							return false, nil
						}
					}
					return true, nil
				}, framework.WaitMedium, pollingInterval).Should(BeTrue())
			}
		})
	})

	Context("use a ClusterAutoscaler that has balance similar nodes enabled and 100 maximum total nodes", func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(100)
			clusterAutoscaler.Spec.BalanceSimilarNodeGroups = pointer.BoolPtr(true)
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed())
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed())
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue())
		})

		// Machines required for test: 4
		// Reason: This test starts with 2 machinesets, each with 1 replica to avoid scaling from zero.
		// Then it autoscales both machinesets to 2 replicas.
		// Does not start with replicas=0 machineset to avoid scaling from 0.
		It("places nodes evenly across node groups [Slow]", func() {
			By("Creating 2 MachineSets each with 1 replica")
			var transientMachineSets [2]*machinev1.MachineSet
			targetedNodeLabel := fmt.Sprintf("%v-balance-nodes", autoscalerWorkerNodeRoleLabel)
			for i, machineSet := range transientMachineSets {
				machineSetParams := framework.BuildMachineSetParams(client, 1)
				machineSetParams.Labels[targetedNodeLabel] = ""
				// remove this label to make the MachineSets similar, see below for more details
				delete(machineSetParams.Labels, "e2e.openshift.io")
				machineSet, err = framework.CreateMachineSet(client, machineSetParams)
				Expect(err).ToNot(HaveOccurred())
				cleanupObjects[machineSet.GetName()] = machineSet
				transientMachineSets[i] = machineSet
			}

			// balance similar nodes requires that all the participating MachineSets have the same
			// instance types and labels. while the instance types should be the same, we want to
			// ensure that no extra labels have been added.
			// TODO it would be nice to check instance types as well, this will require adding some deserialization code for the machine specs.
			By("Ensuring both MachineSets have the same labels")
			Expect(reflect.DeepEqual(transientMachineSets[0].Labels, transientMachineSets[1].Labels)).Should(BeTrue())

			By("Waiting for all Machines in MachineSets to enter Running phase")
			framework.WaitForMachineSet(client, transientMachineSets[0].GetName())
			framework.WaitForMachineSet(client, transientMachineSets[1].GetName())

			maxMachineSetReplicas := int32(3)
			for _, machineSet := range transientMachineSets {
				By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
					machineSet.GetName(), maxMachineSetReplicas))
				asr := machineAutoscalerResource(machineSet, 1, maxMachineSetReplicas)
				Expect(client.Create(ctx, asr)).Should(Succeed())
				cleanupObjects[asr.GetName()] = asr
			}

			// 4 job replicas are being chosen here to force the cluster to
			// expand its size by 2 nodes. the cluster autoscaler should
			// place 1 node in each of the 2 MachineSets created.
			jobReplicas := int32(4)
			uniqueJobName := fmt.Sprintf("%s-balance-nodegroups", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, targetedNodeLabel, "")
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed())

			expectedReplicas := int32(2)
			for _, machineSet := range transientMachineSets {
				By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale out", machineSet.GetName()))
				Eventually(func() (int, error) {
					current, err := framework.GetMachineSet(client, machineSet.GetName())
					if err != nil {
						return 0, err
					}
					return int(pointer.Int32Deref(current.Spec.Replicas, 0)), nil
				}, framework.WaitOverMedium, pollingInterval).Should(BeEquivalentTo(expectedReplicas))
			}
		})
	})

	Context("use a ClusterAutoscaler that has 8 maximum total nodes", func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler
		caMaxNodesTotal := 8

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(caMaxNodesTotal)
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed())
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed())
			}

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed())
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue())
		})

		// Machines required for test: 2
		// Reason: This test starts with 1 replica machineSet. Then it creates a workload that would require 3 replicas,
		// but it only scales up to 2 replicas because the cluster is at maximum size of 8 machines. (3 masters and 3 other worker machines; 2 workers from this test)
		// Does not start with replicas=0 machineset to avoid scaling from 0.
		It("scales up and down while respecting MaxNodesTotal [Slow][Serial]", func() {
			// This test requires to have exactly 6 machines in the cluster at the begining and to run serially.
			By("Ensuring there are 6 machines in the cluster")
			Eventually(func() (int, error) {
				machines, err := framework.GetMachines(client)
				if err != nil {
					return 0, err
				}
				return len(machines), nil
			}, framework.WaitLong, pollingInterval).Should(BeEquivalentTo(6), "Expected to have 6 machines in the cluster")

			By("Creating 1 MachineSet with 1 replica")
			var transientMachineSet *machinev1.MachineSet
			targetedNodeLabel := fmt.Sprintf("%v-scale-updown", autoscalerWorkerNodeRoleLabel)
			machineSetParams := framework.BuildMachineSetParams(client, 1)
			machineSetParams.Labels[targetedNodeLabel] = ""
			transientMachineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred())
			cleanupObjects[transientMachineSet.GetName()] = transientMachineSet

			By("Waiting for all Machines in the MachineSet to enter Running phase")
			framework.WaitForMachineSet(client, transientMachineSet.GetName())

			// To exercise the MaxNodesTotal mechanism we want to make sure
			// that the MachineSet can grow large enough to reach the boundary.
			// A simple way to test this is by setting the max scale size to
			// the MaxNodesTotal+1, since we will not be able to reach this limit
			// due to the original install master/worker nodes.
			maxMachineSetReplicas := int32(caMaxNodesTotal + 1)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
				transientMachineSet.GetName(), maxMachineSetReplicas))
			asr := machineAutoscalerResource(transientMachineSet, 1, maxMachineSetReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed())
			cleanupObjects[asr.GetName()] = asr

			// We want to create a workload that would cause the autoscaler to
			// grow the cluster beyond the MaxNodesTotal. If we set the replicas
			// to the maximum MachineSet size this will create enough demand to
			// grow the cluster to maximum size.
			jobReplicas := maxMachineSetReplicas
			uniqueJobName := fmt.Sprintf("%s-scale-to-maxnodestotal", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, targetedNodeLabel, "")
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed())

			// At this point the autoscaler should be growing the cluster, we
			// wait until the cluster has grown to reach MaxNodesTotal size.
			// Because the autoscaler will ignore nodes that are not ready or unschedulable,
			// we need to check against the number of ready nodes in the cluster since
			// previous tests might have left nodes that are not ready or unschedulable.
			By(fmt.Sprintf("Waiting for cluster to scale up to %d nodes", caMaxNodesTotal))
			Eventually(func() (bool, error) {
				nodes, err := framework.GetReadyAndSchedulableNodes(client)
				return len(nodes) == caMaxNodesTotal, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue())

			// Wait for all nodes to become ready, we wait here to help ensure
			// that the cluster has reached a steady state and no more machines
			// are in the process of being added.
			By("Waiting for all Machines in MachineSet to enter Running phase")
			framework.WaitForMachineSet(client, transientMachineSet.GetName())

			// Now that the cluster has reached maximum size, we want to ensure
			// that it doesn't try to grow larger.
			// Because the autoscaler will ignore nodes that are not ready or unschedulable,
			// we need to check against the number of ready nodes in the cluster since
			// previous tests might have left nodes that are not ready or unschedulable.
			By("Watching Cluster node count to ensure it remains consistent")
			Consistently(func() (bool, error) {
				nodes, err := framework.GetReadyAndSchedulableNodes(client)
				return len(nodes) == caMaxNodesTotal, err
			}, framework.WaitShort, pollingInterval).Should(BeTrue())

			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed())
			delete(cleanupObjects, workload.Name)

			// With the workload gone, the MachineSet should scale back down to
			// its minimum size of 1.
			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale down", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				machineSet, err := framework.GetMachineSet(client, transientMachineSet.Name)
				if err != nil {
					return false, err
				}
				return pointer.Int32Deref(machineSet.Spec.Replicas, -1) == 1, nil
			}, framework.WaitMedium, pollingInterval).Should(BeTrue())
			By(fmt.Sprintf("Waiting for Deleted MachineSet %s nodes to go away", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				nodes, err := framework.GetNodesFromMachineSet(client, transientMachineSet)
				return len(nodes) == 1, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue())
			By(fmt.Sprintf("Waiting for Deleted MachineSet %s machines to go away", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				machines, err := framework.GetMachinesFromMachineSet(client, transientMachineSet)
				return len(machines) == 1, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue())
		})
	})
})
