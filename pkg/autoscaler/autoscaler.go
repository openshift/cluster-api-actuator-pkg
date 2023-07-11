package autoscaler

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"

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
	caMinSizeAnnotation           = "machine.openshift.io/cluster-api-autoscaler-node-group-min-size"
	caMaxSizeAnnotation           = "machine.openshift.io/cluster-api-autoscaler-node-group-max-size"
)

// Build default CA resource to allow fast scaling up and down.
func clusterAutoscalerResource(maxNodesTotal int) *caov1.ClusterAutoscaler {
	tenSecondString := "10s"

	// Choose a time that is at least twice as the sync period
	// and that has high least common multiple to avoid a case
	// when a node is considered to be empty even if there are
	// pods already scheduled and running on the node.
	unneededTimeString := "60s"

	// set the logging verbosity high enough that we can get more debugging information
	var logverbosity int32 = 4

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
				MaxNodesTotal: pointer.Int32(int32(maxNodesTotal)),
			},
			LogVerbosity: pointer.Int32(logverbosity),
		},
	}
}

// Build MA resource from targeted machineset.
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
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client for test")

		komega.SetClient(client)

		workerNodes, err := framework.GetWorkerNodes(client)
		Expect(err).NotTo(HaveOccurred(), "Failed to get worker Node objects")
		Expect(len(workerNodes)).To(BeNumerically(">=", 1), "Expected >= 1 worker node, observed %d", len(workerNodes))

		memCapacity := workerNodes[0].Status.Allocatable[corev1.ResourceMemory]
		Expect(memCapacity).ShouldNot(BeNil(), "First worker node does not advertise an allocatable memory capacity")
		Expect(memCapacity.String()).ShouldNot(BeEmpty(), "First worker node has an empty allocatable memory capacity")
		klog.Infof("Allocatable memory capacity of worker node %q is %s", workerNodes[0].Name, memCapacity.String())

		bytes, ok := memCapacity.AsInt64()
		Expect(ok).Should(BeTrue(), "Failed to convert allocatable memory capacity into byte count as Int64, capacity is %v", memCapacity)

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

			Expect(deleteObject(name, obj)).To(Succeed(), "Failed to delete object %v", name)
		}

		if len(machineSets) > 0 {
			// Wait for all MachineSets and their Machines to be deleted.
			By("Waiting for MachineSets to be deleted...")
			framework.WaitForMachineSetsDeleted(ctx, client, machineSets...)
		}
	})

	Context("use a ClusterAutoscaler that has 100 maximum total nodes count", framework.LabelPeriodic, func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler
		var caEventWatcher *eventWatcher

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred())

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(100)
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed(), "Failed to create ClusterAutoscaler resource")
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler

			By("Starting Cluster Autoscaler event watcher")
			clientset, err := framework.LoadClientset()
			Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes Clientset")
			caEventWatcher, err = newEventWatcher(clientset)
			Expect(err).NotTo(HaveOccurred(), "Failed to create event watcher")
			Expect(caEventWatcher.run()).Should(BeTrue(), "Failed to start event watcher informer")
			// Log cluster-autoscaler events
			caEventWatcher.onEvent(matchAnyEvent, func(e *corev1.Event) {
				if e.Source.Component == clusterAutoscalerComponent {
					klog.Infof("%s: %s", e.InvolvedObject.Name, e.Message)
				}
			}).enable()
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to gather spec report")
			}

			By("Stopping Cluster Autoscaler event watcher")
			caEventWatcher.stop()

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed(), "Failed to delete ClusterAutoscaler")
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Failed to cleanup Cluster Autoscaler before timeout")
		})

		// Machines required for test: 2
		// Reason: This tests checks that autoscaler is able to scale from zero. It requires 2 machines to ensure it scales to the correct number of nodes based on the workload size.
		It("It scales from/to zero", func() {
			// Only run in platforms which support autoscaling from/to zero.
			clusterInfra, err := framework.GetInfrastructure(ctx, client)
			Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")

			platform := clusterInfra.Status.PlatformStatus.Type
			switch platform {
			case configv1.AWSPlatformType, configv1.GCPPlatformType, configv1.AzurePlatformType, configv1.OpenStackPlatformType, configv1.VSpherePlatformType, configv1.NutanixPlatformType:
				klog.Infof("Platform is %v", platform)
			default:
				Skip(fmt.Sprintf("Platform %v does not support autoscaling from/to zero, skipping.", platform))
			}

			By("Creating a new MachineSet with 0 replicas")
			machineSetParams := framework.BuildMachineSetParams(ctx, client, 0)
			targetedNodeLabel := fmt.Sprintf("%v-scale-from-zero", autoscalerWorkerNodeRoleLabel)
			machineSetParams.Labels[targetedNodeLabel] = ""

			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet with 0 replicas")
			cleanupObjects[machineSet.GetName()] = machineSet

			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			expectedReplicas := int32(2)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v",
				machineSet.GetNamespace(), machineSet.GetName(), 0, expectedReplicas))
			asr := machineAutoscalerResource(machineSet, 0, expectedReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed(), "Failed to create MachineAutoscaler with min 0/max 2 replicas")
			cleanupObjects[asr.GetName()] = asr

			uniqueJobName := fmt.Sprintf("%s-scale-from-zero", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s", uniqueJobName, expectedReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(expectedReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, "", corev1.NodeSelectorRequirement{
				Key:      targetedNodeLabel,
				Operator: corev1.NodeSelectorOpExists,
			})
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed(), "Failed to create scale-out workload %s", workloadJobName)

			Eventually(func() bool {
				ms, err := framework.GetMachineSet(ctx, client, machineSet.GetName())
				Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet %s", machineSet.GetName())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale out. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "MachineSet %s failed to scale out to %d replicas", machineSet.GetName(), expectedReplicas)

			By("Waiting for the machineSet replicas to become nodes")
			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			expectedReplicas = 0
			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed(), "Failed to delete scale-out workload %s", workload.Name)
			delete(cleanupObjects, workload.Name)
			Eventually(func() bool {
				ms, err := framework.GetMachineSet(ctx, client, machineSet.GetName())
				Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet %s", machineSet.GetName())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale in. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "MachineSet %s failed to scale in to 0 replicas", machineSet.GetName())
		})

		// Machines required for test: 1
		// Reason: This tests checks that autoscaler is able to scale from zero when a workload requires specific architecture in the node affinity fields.
		// Moreover, this test gives a better signal when multiple architectures are available in the cluster or the cluster is not amd64,
		// and the workload requires to be scheduled on an architecture different from amd64.
		It("It scales from/to zero a machine set with the architecture requested by the workload", func() {
			// Only run in platforms which support arch-aware autoscaling from/to zero.
			clusterInfra, err := framework.GetInfrastructure(ctx, client)
			Expect(err).NotTo(HaveOccurred(), "Failed to get cluster infrastructure object")

			platform := clusterInfra.Status.PlatformStatus.Type
			switch platform {
			case configv1.AWSPlatformType, configv1.AzurePlatformType:
				klog.Infof("Platform is %v", platform)
			default:
				Skip(fmt.Sprintf("Platform %v does not support arch-aware autoscaling from/to zero, skipping.", platform))
			}

			By("Creating a new MachineSet with 0 replicas for each machineset of a different architecture found in the cluster")
			machineSetParamsList := framework.BuildPerArchMachineSetParamsSet(ctx, client, 0)
			targetedNodeLabel := fmt.Sprintf("%v-scale-from-zero", autoscalerWorkerNodeRoleLabel)
			workloadArch := framework.Amd64
			expectedReplicas := int32(1)
			var expectedScaledMachineSet *machinev1.MachineSet
			var machineSets = make([]*machinev1.MachineSet, 0)

			for _, machineSetParams := range machineSetParamsList {
				machineSetParams.Labels[targetedNodeLabel] = ""
				By(fmt.Sprintf("Deploying the MachineSet with 0 replicas (architecture: %s)",
					machineSetParams.Labels[framework.ArchLabel]))
				machineSet, err := framework.CreateMachineSet(client, machineSetParams)
				Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet with 0 replicas")
				machineSets = append(machineSets, machineSet)
				cleanupObjects[machineSet.GetName()] = machineSet

				framework.WaitForMachineSet(ctx, client, machineSet.GetName())

				By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v",
					machineSet.GetNamespace(), machineSet.GetName(), 0, expectedReplicas))
				asr := machineAutoscalerResource(machineSet, 0, expectedReplicas)
				Expect(client.Create(ctx, asr)).Should(Succeed(), fmt.Sprintf("Failed to create MachineAutoscaler with min 0/max %v replicas", expectedReplicas))
				cleanupObjects[asr.GetName()] = asr
				arch := machineSetParams.Labels[framework.ArchLabel]
				if arch != framework.Amd64 {
					workloadArch = arch
					expectedScaledMachineSet = machineSet
				}
			}
			if workloadArch == "" {
				// If workloadArch is empty, it means that exactly one MachineSet is in the list, and it is amd64.
				workloadArch = framework.Amd64
				expectedScaledMachineSet = machineSets[0]
			}
			uniqueJobName := fmt.Sprintf("%s-scale-from-zero", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s", uniqueJobName,
				expectedReplicas+1, workloadMemRequest.String()))
			// Executing one additional replica to ensure that the workload does not triggers other scale-out events with
			// other MachineSets not having the requested architecture.
			workload := framework.NewWorkLoad(expectedReplicas+1, workloadMemRequest, uniqueJobName, autoscalingTestLabel, "",
				corev1.NodeSelectorRequirement{
					Key:      targetedNodeLabel,
					Operator: corev1.NodeSelectorOpExists,
				},
				corev1.NodeSelectorRequirement{
					Key:      "kubernetes.io/arch",
					Operator: corev1.NodeSelectorOpIn,
					Values:   []string{workloadArch},
				})
			// Setting the command to sleep 30 seconds to ensure that the workload can be completed and the additional
			// replica (job completion) is run on a node with the requested architecture before the scale-in event is triggered.
			workload.Spec.Template.Spec.Containers[0].Command = []string{"sleep", "30"}

			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed(), "Failed to create scale-out workload %s", workloadJobName)

			Consistently(func() bool {
				condition := true
				for _, machineSet := range machineSets {
					if machineSet.GetName() == expectedScaledMachineSet.GetName() {
						// Ignore the MachineSet with the architecture requested by the workload.
						continue
					}
					ms, err := framework.GetMachineSet(ctx, client, machineSet.GetName())
					Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet %s", machineSet.GetName())

					By(fmt.Sprintf("Current replicas for %s are %d, expected %d.", machineSet.GetName(),
						*ms.Spec.Replicas, 0))

					condition = condition && *ms.Spec.Replicas == 0
				}

				return condition
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Some MachineSets scaled out unexpectedly.")
			Eventually(func() bool {
				ms, err := framework.GetMachineSet(ctx, client, expectedScaledMachineSet.GetName())
				Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet %s", expectedScaledMachineSet.GetName())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale out. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "MachineSet %s failed to scale out to %d replicas",
				expectedScaledMachineSet.GetName(), expectedReplicas)

			By("Waiting for the machineSet replicas to become nodes")
			framework.WaitForMachineSet(ctx, client, expectedScaledMachineSet.GetName())
			job := &v1.Job{}
			Eventually(func() bool {
				err = client.Get(ctx, types.NamespacedName{Name: workload.GetName(), Namespace: workload.GetNamespace()}, job)
				if err != nil {
					By(fmt.Sprintf("Failed to get job %s: %v", workload.GetName(), err))
					return false
				}
				By(fmt.Sprintf("Waiting for the workload to complete. Current completions are %d, expected %d.",
					job.Status.Succeeded, expectedReplicas+1))

				return job.Status.Succeeded == expectedReplicas+1
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Workload %s failed to complete", workload.GetName())
			expectedReplicas = 0
			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed(), "Failed to delete scale-out workload %s", workload.Name)
			delete(cleanupObjects, workload.Name)
			Eventually(func() bool {
				ms, err := framework.GetMachineSet(ctx, client, expectedScaledMachineSet.GetName())
				Expect(err).ToNot(HaveOccurred(), "Failed to get MachineSet %s", expectedScaledMachineSet.GetName())

				By(fmt.Sprintf("Waiting for machineSet replicas to scale in. Current replicas are %v, expected %v.",
					*ms.Spec.Replicas, expectedReplicas))

				return *ms.Spec.Replicas == expectedReplicas
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "MachineSet %s failed to scale in to 0 replicas", expectedScaledMachineSet.GetName())
		})

		// Machines required for test: 2
		// Reason: Needs to scale down to minReplicas = 1. Scales 1 -> 2 -> 1.
		It("cleanup deletion information after scale down [Slow]", func() {
			By("Creating MachineSet with 1 replica")
			targetedNodeLabel := fmt.Sprintf("%v-delete-cleanup", autoscalerWorkerNodeRoleLabel)
			machineSetParams := framework.BuildMachineSetParams(ctx, client, 1)
			machineSetParams.Labels[targetedNodeLabel] = ""
			machineSet, err := framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet with 1 replica")
			cleanupObjects[machineSet.GetName()] = machineSet

			By("Waiting for the machineSet to enter Running phase")
			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			expectedReplicas := int32(2)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
				machineSet.GetName(), expectedReplicas))
			asr := machineAutoscalerResource(machineSet, 1, expectedReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed(), "Failed to create MachineAutoscaler with min 1/max %d replicas", expectedReplicas)
			cleanupObjects[asr.GetName()] = asr

			jobReplicas := expectedReplicas
			uniqueJobName := fmt.Sprintf("%s-cleanup-after-scale-down", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, "", corev1.NodeSelectorRequirement{
				Key:      targetedNodeLabel,
				Operator: corev1.NodeSelectorOpExists,
			})
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed(), "Failed to create scale-out workload %s", uniqueJobName)

			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale out", machineSet.GetName()))
			Eventually(func() (int32, error) {
				current, err := framework.GetMachineSet(ctx, client, machineSet.GetName())
				if err != nil {
					return 0, err
				}

				return *current.Spec.Replicas, nil
			}, framework.WaitMedium, pollingInterval).Should(BeEquivalentTo(expectedReplicas), "MachineSet %s failed to scale out to %d replicas", machineSet.GetName(), expectedReplicas)

			By("Waiting for all Machines in the MachineSet to enter Running phase")
			framework.WaitForMachineSet(ctx, client, machineSet.GetName())

			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed(), "Failed to delete workload object %s", workload.Name)
			delete(cleanupObjects, workload.Name)

			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale in", machineSet.GetName()))
			expectedLength := 1
			var machines []*machinev1.Machine
			Eventually(func() (int, error) {
				machines, err = framework.GetMachinesFromMachineSet(ctx, client, machineSet)
				if err != nil {
					return 0, err
				}

				return len(machines), nil
			}, framework.WaitLong, pollingInterval).Should(BeEquivalentTo(expectedLength), "MachineSet %s failed to scale in to %d replicas", machineSet.GetName(), expectedLength)

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
				}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Machine %s did not receive a deletion annotation", machine.Name)
			}

			for _, machine := range machines {
				if machine.Status.NodeRef == nil {
					continue
				}
				By(fmt.Sprintf("Checking Node %s for %s and %s taints", machine.Status.NodeRef.Name, deletionCandidateTaintKey, toBeDeletedTaintKey))
				Eventually(func() (bool, error) {
					n, err := framework.GetNodeForMachine(ctx, client, machine)
					if err != nil {
						return false, err
					}
					for _, t := range n.Spec.Taints {
						if t.Key == deletionCandidateTaintKey || t.Key == toBeDeletedTaintKey {
							return false, nil
						}
					}

					return true, nil
				}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Node %s did not receive a deletion candidate taint", machine.Status.NodeRef.Name)
			}
		})
	})

	Context("use a ClusterAutoscaler that has balance similar nodes enabled and 100 maximum total nodes", func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to create gatherer")

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(100)
			clusterAutoscaler.Spec.BalanceSimilarNodeGroups = pointer.Bool(true)
			// Ignore this label to make test nodes similar
			clusterAutoscaler.Spec.BalancingIgnoredLabels = []string{
				"e2e.openshift.io",
			}
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed(), "Failed to create ClusterAutoscaler")
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to gather spec report")
			}

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed(), "Failed to delete ClusterAutoscaler")
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Failed to cleanup Cluster Autoscaler before timeout")
		})

		// Machines required for test: 4
		// Reason: This test starts with 2 machinesets, each with 1 replica to avoid scaling from zero.
		// Then it autoscales both machinesets to 2 replicas.
		// Does not start with replicas=0 machineset to avoid scaling from 0.
		It("places nodes evenly across node groups [Slow]", func() {
			By("Creating 2 MachineSets each with 1 replica")
			var transientMachineSets [2]*machinev1.MachineSet
			targetedNodeLabel := fmt.Sprintf("%v-balance-nodes", autoscalerWorkerNodeRoleLabel)
			for i := range transientMachineSets {
				machineSetParams := framework.BuildMachineSetParams(ctx, client, 1)
				machineSetParams.Labels[targetedNodeLabel] = ""
				machineSet, err := framework.CreateMachineSet(client, machineSetParams)
				Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet %d of %d", i, len(transientMachineSets))
				cleanupObjects[machineSet.GetName()] = machineSet
				transientMachineSets[i] = machineSet
			}

			// balance similar nodes requires that all the participating Nodes have the same
			// instance types and labels. while the instance types should be the same, we want to
			// ensure that no extra labels have been added.
			// TODO it would be nice to check instance types as well, this will require adding some deserialization code for the machine specs.
			By("Ensuring both MachineSets have the same .spec.template.spec.labels")
			// Ignore e2e.openshift.io in the comparison to test BalancingIgnoredLabels feature
			labelsMachineSetA := transientMachineSets[0].Spec.Template.Spec.Labels
			delete(labelsMachineSetA, "e2e.openshift.io")
			labelsMachineSetB := transientMachineSets[1].Spec.Template.Spec.Labels
			delete(labelsMachineSetB, "e2e.openshift.io")
			Expect(labelsMachineSetA).To(Equal(labelsMachineSetB), "Failed to match MachineSet labels for balancing similar nodes")

			By("Waiting for all Machines in MachineSets to enter Running phase")
			framework.WaitForMachineSet(ctx, client, transientMachineSets[0].GetName())
			framework.WaitForMachineSet(ctx, client, transientMachineSets[1].GetName())

			maxMachineSetReplicas := int32(3)
			for _, machineSet := range transientMachineSets {
				By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
					machineSet.GetName(), maxMachineSetReplicas))
				asr := machineAutoscalerResource(machineSet, 1, maxMachineSetReplicas)
				Expect(client.Create(ctx, asr)).Should(Succeed(), "Failed to create MachineAutoscaler with min 1/max %d replicas", maxMachineSetReplicas)
				cleanupObjects[asr.GetName()] = asr
			}

			for _, machineSet := range transientMachineSets {
				By(fmt.Sprintf("Waiting for MachineSet %s to acquire autoscaling annotations", machineSet.GetName()))
				Eventually(komega.Object(machineSet), framework.WaitShort, pollingInterval).Should(HaveField("GetAnnotations()", SatisfyAll(
					HaveKey(caMinSizeAnnotation),
					HaveKey(caMaxSizeAnnotation),
				)), "MachineSet %s did not acquire autoscaling annotation", machineSet.GetName())
			}

			// on some cloud providers, we have experienced the new nodes having resources added to their status.capacity
			// after they have been initialized. this has been seen with `hugepages-1Gi` and `hugepages-2Mi`. so we
			// wait until the nodes have both entries before moving on.
			By("Waiting for the new Nodes to have hugepages-1Gi and hugepages-2Mi capacity")
			Eventually(func() ([]*corev1.Node, error) {
				nodes := []*corev1.Node{}

				for _, machineSet := range transientMachineSets {
					if n, err := framework.GetNodesFromMachineSet(ctx, client, machineSet); err != nil {
						return nodes, err
					} else if len(n) != 1 {
						return nodes, fmt.Errorf("expected 1 node for MachineSet %s, found %d", machineSet.GetName(), len(n))
					} else {
						nodes = append(nodes, n[0])
					}
				}

				return nodes, nil
			}, framework.WaitMedium, pollingInterval).Should(HaveEach(
				HaveField("Status.Capacity", SatisfyAll(
					HaveKey(BeEquivalentTo(corev1.ResourceHugePagesPrefix+"1Gi")),
					HaveKey(BeEquivalentTo(corev1.ResourceHugePagesPrefix+"2Mi")),
				)),
			), "Node capacity resources did not contain hugepages-1Gi and hugepages-2Mi")

			// wait until the new nodes have the same resource keys before progressing, otherwise the balance will not work.
			By("Waiting for the new Nodes to have similar resources")
			Eventually(func() (map[corev1.ResourceName]int, error) {
				nodes := []*corev1.Node{}

				for _, machineSet := range transientMachineSets {
					if n, err := framework.GetNodesFromMachineSet(ctx, client, machineSet); err != nil {
						return nil, err
					} else if len(n) != 1 {
						return nil, fmt.Errorf("expected 1 node for MachineSet %s, found %d", machineSet.GetName(), len(n))
					} else {
						nodes = append(nodes, n[0])
					}
				}

				// add the resources from each node's capacity, counting the number of each
				resources := map[corev1.ResourceName]int{}
				for _, n := range nodes {
					for k := range n.Status.Capacity {
						resources[k] += 1
					}
				}

				return resources, nil
			}, framework.WaitMedium, pollingInterval).Should(HaveEach(2), "Node capacity resources did not match")

			// 4 job replicas are being chosen here to force the cluster to
			// expand its size by 2 nodes. the cluster autoscaler should
			// place 1 node in each of the 2 MachineSets created.
			jobReplicas := int32(4)
			uniqueJobName := fmt.Sprintf("%s-balance-nodegroups", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, "", corev1.NodeSelectorRequirement{
				Key:      targetedNodeLabel,
				Operator: corev1.NodeSelectorOpExists,
			})
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed(), "Failed to create scale-out workload %s", uniqueJobName)

			expectedReplicas := int(2)
			By("Waiting for transient MachineSets replicas to scale out")
			Eventually(func() (map[string]int, error) {
				allreplicas := map[string]int{}

				for _, machineSet := range transientMachineSets {
					ms, err := framework.GetMachineSet(ctx, client, machineSet.GetName())
					if err != nil {
						return allreplicas, err
					}

					replicas := int(pointer.Int32Deref(ms.Spec.Replicas, 0))
					allreplicas[ms.GetName()] = replicas

					if replicas > expectedReplicas {
						return allreplicas, StopTrying(fmt.Sprintf(
							"observed %d replicas in MachineSet %s, which exceeds the expected replicas of %d",
							replicas, ms.GetName(), expectedReplicas))
					}
				}

				return allreplicas, nil
			}, framework.WaitOverMedium, pollingInterval).Should(HaveEach(expectedReplicas), "Failed to balance properly")
		})
	})

	Context("use a ClusterAutoscaler that has 8 maximum total nodes", framework.LabelPeriodic, func() {
		var clusterAutoscaler *caov1.ClusterAutoscaler
		caMaxNodesTotal := 8

		BeforeEach(func() {
			gatherer, err = framework.NewGatherer()
			Expect(err).ToNot(HaveOccurred(), "Failed to create gatherer")

			By("Creating ClusterAutoscaler")
			clusterAutoscaler = clusterAutoscalerResource(caMaxNodesTotal)
			Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed(), "Failed to create ClusterAutoscaler")
			cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler
		})

		AfterEach(func() {
			specReport := CurrentSpecReport()
			if specReport.Failed() {
				Expect(gatherer.WithSpecReport(specReport).GatherAll()).To(Succeed(), "Failed to gather spec report")
			}

			// explicitly delete the ClusterAutoscaler
			// this is needed due to the autoscaler tests requiring singleton
			// deployments of the ClusterAutoscaler.
			By("Waiting for ClusterAutoscaler to delete.")
			caName := clusterAutoscaler.GetName()
			Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed(), "Failed to delete ClusterAutoscaler")
			delete(cleanupObjects, caName)
			Eventually(func() (bool, error) {
				_, err := framework.GetClusterAutoscaler(client, caName)
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				// Return the error so that failures print additional errors
				return false, err
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "Failed to cleanup Cluster Autoscaler before timeout")
		})

		// Machines required for test: 2
		// Reason: This test starts with 1 replica machineSet. Then it creates a workload that would require 3 replicas,
		// but it only scales up to 2 replicas because the cluster is at maximum size of 8 machines. (3 masters and 3 other worker machines; 2 workers from this test)
		// Does not start with replicas=0 machineset to avoid scaling from 0.
		It("scales up and down while respecting MaxNodesTotal [Slow][Serial]", func() {
			// This test requires to have exactly 6 machines in the cluster at the beginning and to run serially.
			By("Ensuring there are 6 machines in the cluster")
			Eventually(func() (int, error) {
				machines, err := framework.GetMachines(ctx, client)
				if err != nil {
					return 0, err
				}

				return len(machines), nil
			}, framework.WaitLong, pollingInterval).Should(BeEquivalentTo(6), "Expected to have 6 machines in the cluster")

			By("Creating 1 MachineSet with 1 replica")
			var transientMachineSet *machinev1.MachineSet
			targetedNodeLabel := fmt.Sprintf("%v-scale-updown", autoscalerWorkerNodeRoleLabel)
			machineSetParams := framework.BuildMachineSetParams(ctx, client, 1)
			machineSetParams.Labels[targetedNodeLabel] = ""
			transientMachineSet, err = framework.CreateMachineSet(client, machineSetParams)
			Expect(err).ToNot(HaveOccurred(), "Failed to create MachineSet with 1 replica")
			cleanupObjects[transientMachineSet.GetName()] = transientMachineSet

			By("Waiting for all Machines in the MachineSet to enter Running phase")
			framework.WaitForMachineSet(ctx, client, transientMachineSet.GetName())

			// To exercise the MaxNodesTotal mechanism we want to make sure
			// that the MachineSet can grow large enough to reach the boundary.
			// A simple way to test this is by setting the max scale size to
			// the MaxNodesTotal+1, since we will not be able to reach this limit
			// due to the original install master/worker nodes.
			maxMachineSetReplicas := int32(caMaxNodesTotal + 1)
			By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s - min: 1, max: %d",
				transientMachineSet.GetName(), maxMachineSetReplicas))
			asr := machineAutoscalerResource(transientMachineSet, 1, maxMachineSetReplicas)
			Expect(client.Create(ctx, asr)).Should(Succeed(), "Failed to create MachineAutoscaler with min 1/max %d replicas", maxMachineSetReplicas)
			cleanupObjects[asr.GetName()] = asr

			// We want to create a workload that would cause the autoscaler to
			// grow the cluster beyond the MaxNodesTotal. If we set the replicas
			// to the maximum MachineSet size this will create enough demand to
			// grow the cluster to maximum size.
			jobReplicas := maxMachineSetReplicas
			uniqueJobName := fmt.Sprintf("%s-scale-to-maxnodestotal", workloadJobName)
			By(fmt.Sprintf("Creating scale-out workload %s: jobs: %v, memory: %s",
				uniqueJobName, jobReplicas, workloadMemRequest.String()))
			workload := framework.NewWorkLoad(jobReplicas, workloadMemRequest, uniqueJobName, autoscalingTestLabel, "", corev1.NodeSelectorRequirement{
				Key:      targetedNodeLabel,
				Operator: corev1.NodeSelectorOpExists,
			})
			cleanupObjects[workload.GetName()] = workload
			Expect(client.Create(ctx, workload)).Should(Succeed(), "Failed to create scale-out workload %s", uniqueJobName)

			// At this point the autoscaler should be growing the cluster, we
			// wait until the cluster has grown to reach MaxNodesTotal size.
			// Because the autoscaler will ignore nodes that are not ready or unschedulable,
			// we need to check against the number of ready nodes in the cluster since
			// previous tests might have left nodes that are not ready or unschedulable.
			By(fmt.Sprintf("Waiting for cluster to scale up to %d nodes", caMaxNodesTotal))
			Eventually(func() (bool, error) {
				nodes, err := framework.GetReadyAndSchedulableNodes(client)
				return len(nodes) == caMaxNodesTotal, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Cluster failed to reach %d nodes", caMaxNodesTotal)

			// Wait for all nodes to become ready, we wait here to help ensure
			// that the cluster has reached a steady state and no more machines
			// are in the process of being added.
			By("Waiting for all Machines in MachineSet to enter Running phase")
			framework.WaitForMachineSet(ctx, client, transientMachineSet.GetName())

			// Now that the cluster has reached maximum size, we want to ensure
			// that it doesn't try to grow larger.
			// Because the autoscaler will ignore nodes that are not ready or unschedulable,
			// we need to check against the number of ready nodes in the cluster since
			// previous tests might have left nodes that are not ready or unschedulable.
			By("Watching Cluster node count to ensure it remains consistent")
			Consistently(func() (int, error) {
				nodes, err := framework.GetReadyAndSchedulableNodes(client)
				return len(nodes), err
			}, framework.WaitShort, pollingInterval).Should(Equal(caMaxNodesTotal), "Cluster failed to stay consistent at %d nodes", caMaxNodesTotal)

			By("Deleting the workload")
			Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed(), "Failed to delete scale-out workload %s", workload.Name)
			delete(cleanupObjects, workload.Name)

			// With the workload gone, the MachineSet should scale back down to
			// its minimum size of 1.
			By(fmt.Sprintf("Waiting for MachineSet %s replicas to scale down", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				machineSet, err := framework.GetMachineSet(ctx, client, transientMachineSet.Name)
				if err != nil {
					return false, err
				}

				return pointer.Int32Deref(machineSet.Spec.Replicas, -1) == 1, nil
			}, framework.WaitMedium, pollingInterval).Should(BeTrue(), "MachineSet %s failed to scale down to 1 replica", transientMachineSet.GetName())
			By(fmt.Sprintf("Waiting for Deleted MachineSet %s nodes to go away", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				nodes, err := framework.GetNodesFromMachineSet(ctx, client, transientMachineSet)
				return len(nodes) == 1, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Nodes failed to scale down to 1 node")
			By(fmt.Sprintf("Waiting for Deleted MachineSet %s machines to go away", transientMachineSet.GetName()))
			Eventually(func() (bool, error) {
				machines, err := framework.GetMachinesFromMachineSet(ctx, client, transientMachineSet)
				return len(machines) == 1, err
			}, framework.WaitLong, pollingInterval).Should(BeTrue(), "Machines failed to scale down to 1 machine")
		})
	})
})
