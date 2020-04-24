package autoscaler

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	mapiv1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	autoscalingTestLabel                  = "test.autoscaling.label"
	clusterAutoscalerComponent            = "cluster-autoscaler"
	clusterAutoscalerObjectKind           = "ConfigMap"
	clusterAutoscalerScaledUpGroup        = "ScaledUpGroup"
	clusterAutoscalerScaleDownEmpty       = "ScaleDownEmpty"
	clusterAutoscalerMaxNodesTotalReached = "MaxNodesTotalReached"
	pollingInterval                       = 3 * time.Second
	autoscalerWorkerNodeRoleLabel         = "machine.openshift.io/autoscaler-e2e-worker"
	workloadJobName                       = "e2e-autoscaler-workload"
)

func newWorkLoad(njobs int32, memoryRequest resource.Quantity, nodeSelector string) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadJobName,
			Namespace: "default",
			Labels:    map[string]string{autoscalingTestLabel: ""},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  workloadJobName,
							Image: "busybox",
							Command: []string{
								"sleep",
								"86400", // 1 day
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"memory": memoryRequest,
									"cpu":    resource.MustParse("500m"),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicy("Never"),
					Tolerations: []corev1.Toleration{
						{
							Key:      "kubemark",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
			BackoffLimit: pointer.Int32Ptr(4),
			Completions:  pointer.Int32Ptr(njobs),
			Parallelism:  pointer.Int32Ptr(njobs),
		},
	}
	if nodeSelector != "" {
		job.Spec.Template.Spec.NodeSelector = map[string]string{
			nodeSelector: "",
		}
	}
	return job
}

// Build default CA resource to allow fast scaling up and down
func clusterAutoscalerResource(maxNodesTotal int) *caov1.ClusterAutoscaler {
	tenSecondString := "10s"

	// Choose a time that is at least twice as the sync period
	// and that has high least common multiple to avoid a case
	// when a node is considered to be empty even if there are
	// pods already scheduled and running on the node.
	unneededTimeString := "23s"
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
func machineAutoscalerResource(targetMachineSet *mapiv1beta1.MachineSet, minReplicas, maxReplicas int32) *caov1beta1.MachineAutoscaler {
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

func newScaleUpCounter(w *eventWatcher, v uint32, scaledGroups map[string]bool) *eventCounter {
	isAutoscalerScaleUpEvent := func(event *corev1.Event) bool {
		return event.Source.Component == clusterAutoscalerComponent &&
			event.Reason == clusterAutoscalerScaledUpGroup &&
			event.InvolvedObject.Kind == clusterAutoscalerObjectKind &&
			strings.HasPrefix(event.Message, "Scale-up: setting group")
	}

	matchGroup := func(event *corev1.Event) bool {
		if !isAutoscalerScaleUpEvent(event) {
			return false
		}
		for k := range scaledGroups {
			if !scaledGroups[k] && strings.HasPrefix(event.Message, fmt.Sprintf("Scale-up: group %s size set to", k)) {
				scaledGroups[k] = true
			}
		}
		return true
	}

	c := newEventCounter(w, matchGroup, v, increment)
	c.enable()

	return c
}

func newScaleDownCounter(w *eventWatcher, v uint32) *eventCounter {
	isAutoscalerScaleDownEvent := func(event *corev1.Event) bool {
		return event.Source.Component == clusterAutoscalerComponent &&
			event.Reason == clusterAutoscalerScaleDownEmpty &&
			event.InvolvedObject.Kind == clusterAutoscalerObjectKind &&
			strings.HasPrefix(event.Message, "Scale-down: empty node")
	}

	c := newEventCounter(w, isAutoscalerScaleDownEvent, v, increment)
	c.enable()
	return c
}

func newMaxNodesTotalReachedCounter(w *eventWatcher, v uint32) *eventCounter {
	isAutoscalerMaxNodesTotalEvent := func(event *corev1.Event) bool {
		return event.Source.Component == clusterAutoscalerComponent &&
			event.Reason == clusterAutoscalerMaxNodesTotalReached &&
			event.InvolvedObject.Kind == clusterAutoscalerObjectKind &&
			strings.HasPrefix(event.Message, "Max total nodes in cluster reached")
	}

	c := newEventCounter(w, isAutoscalerMaxNodesTotalEvent, v, increment)
	c.enable()
	return c
}

func remaining(t time.Time) time.Duration {
	return t.Sub(time.Now()).Round(time.Second)
}

var _ = Describe("[Feature:Machines] Autoscaler should", func() {

	var workloadMemRequest resource.Quantity
	var client runtimeclient.Client
	var err error
	var cleanupObjects map[string]runtime.Object

	ctx := context.Background()
	cascadeDelete := metav1.DeletePropagationForeground
	deleteObject := func(name string, obj runtime.Object) error {
		glog.Infof("[cleanup] %q (%T)", name, obj)
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

		memCapacity := workerNodes[0].Status.Capacity[corev1.ResourceMemory]
		Expect(memCapacity).ShouldNot(BeNil())
		Expect(memCapacity.String()).ShouldNot(BeEmpty())
		glog.Infof("Memory capacity of worker node %q is %s", workerNodes[0].Name, memCapacity.String())

		bytes, ok := memCapacity.AsInt64()
		Expect(ok).Should(BeTrue())

		// 70% - enough that the existing and new nodes will
		// be used, not enough to have more than 1 pod per
		// node.
		workloadMemRequest = resource.MustParse(fmt.Sprintf("%v", 0.7*float32(bytes)))

		// Anything we create we must cleanup
		cleanupObjects = make(map[string]runtime.Object)
	})

	AfterEach(func() {
		for name, obj := range cleanupObjects {
			Expect(deleteObject(name, obj)).To(Succeed())
		}
	})

	It("scale up and down", func() {
		clientset, err := framework.LoadClientset()
		Expect(err).NotTo(HaveOccurred())

		By("Getting existing machinesets")
		existingMachineSets, err := framework.GetMachineSets(client)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(existingMachineSets)).To(BeNumerically(">=", 1))

		By("Getting existing machines")
		existingMachines, err := framework.GetMachines(client)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(existingMachines)).To(BeNumerically(">=", 1))

		By("Getting existing nodes")
		existingNodes, err := framework.GetNodes(client)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(existingNodes)).To(BeNumerically(">=", 1))

		glog.Infof("Have %v existing machinesets", len(existingMachineSets))
		glog.Infof("Have %v existing machines", len(existingMachines))
		glog.Infof("Have %v existing nodes", len(existingNodes))
		Expect(len(existingNodes) == len(existingMachines)).To(BeTrue())

		// The remainder of the logic in this test requires 3
		// machinesets.
		var machineSets [3]*mapiv1beta1.MachineSet

		randomUUID := string(uuid.NewUUID())
		for i := 0; i < len(machineSets); i++ {
			targetMachineSet := existingMachineSets[i%len(existingMachineSets)]
			machineSetName := fmt.Sprintf("e2e-%s-w-%d", randomUUID[:5], i)
			machineSets[i] = framework.NewMachineSet(targetMachineSet.Labels[framework.ClusterKey],
				targetMachineSet.Namespace,
				machineSetName,
				targetMachineSet.Spec.Selector.MatchLabels,
				targetMachineSet.Spec.Template.ObjectMeta.Labels,
				&targetMachineSet.Spec.Template.Spec.ProviderSpec,
				1) // one replica
			machineSets[i].Spec.Template.Spec.Labels = map[string]string{
				autoscalerWorkerNodeRoleLabel: "",
			}
			Expect(client.Create(ctx, machineSets[i])).Should(Succeed())
			cleanupObjects[machineSets[i].Name] = machineSets[i]
		}

		By(fmt.Sprintf("Creating %v transient machinesets", len(machineSets)))
		testDuration := time.Now().Add(time.Duration(framework.WaitLong))
		Eventually(func() bool {
			By(fmt.Sprintf("[%s remaining] Waiting for nodes to be Ready in %v transient machinesets",
				remaining(testDuration), len(machineSets)))
			var allNewNodes []*corev1.Node
			for i := 0; i < len(machineSets); i++ {
				nodes, err := framework.GetNodesFromMachineSet(client, machineSets[i])
				if err != nil {
					return false
				}
				allNewNodes = append(allNewNodes, nodes...)
			}
			return len(allNewNodes) == len(machineSets) && framework.NodesAreReady(allNewNodes)
		}, framework.WaitLong, pollingInterval).Should(BeTrue())

		// Now that we have created some transient machinesets
		// take stock of the number of nodes we now have.
		By("Getting nodes")
		nodes, err := framework.GetNodes(client)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(nodes)).To(BeNumerically(">=", 1))

		var machineAutoscalers []*caov1beta1.MachineAutoscaler

		By(fmt.Sprintf("Creating %v machineautoscalers", len(machineSets)))
		var clusterExpansionSize int
		for i := range machineSets {
			clusterExpansionSize++
			glog.Infof("Create MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v", machineSets[i].Namespace, machineSets[i].Name, 1, 2)
			asr := machineAutoscalerResource(machineSets[i], 1, 2)
			Expect(client.Create(ctx, asr)).Should(Succeed())
			machineAutoscalers = append(machineAutoscalers, asr)
			cleanupObjects[asr.Name] = asr
		}
		Expect(clusterExpansionSize).To(BeNumerically(">", 1))

		// The total size of our cluster is
		// len(existingMachineSets) + clusterExpansionSize. We
		// cap that to $max-1 because we want to test that the
		// maxNodesTotal flag is respected by the
		// cluster-autoscaler
		maxNodesTotal := len(nodes) + clusterExpansionSize - 1

		eventWatcher := newEventWatcher(clientset)
		Expect(eventWatcher.run()).Should(BeTrue())
		defer eventWatcher.stop()

		// Log cluster-autoscaler events
		eventWatcher.onEvent(matchAnyEvent, func(e *corev1.Event) {
			if e.Source.Component == clusterAutoscalerComponent {
				glog.Infof("%s: %s", e.InvolvedObject.Name, e.Message)
			}
		}).enable()

		By(fmt.Sprintf("Creating ClusterAutoscaler configured with maxNodesTotal:%v", maxNodesTotal))
		clusterAutoscaler := clusterAutoscalerResource(maxNodesTotal)
		Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed())
		cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler

		By(fmt.Sprintf("Deriving Memory capacity from machine %q", existingMachineSets[0].Name))

		By(fmt.Sprintf("Creating scale-out workload: jobs: %v, memory: %s", maxNodesTotal+1, workloadMemRequest.String()))
		scaledGroups := map[string]bool{}
		for i := range machineSets {
			scaledGroups[path.Join(machineSets[i].Namespace, machineSets[i].Name)] = false
		}
		scaleUpCounter := newScaleUpCounter(eventWatcher, 0, scaledGroups)
		maxNodesTotalReachedCounter := newMaxNodesTotalReachedCounter(eventWatcher, 0)
		// +1 to continuously generate the MaxNodesTotalReached
		workload := newWorkLoad(int32(maxNodesTotal+1), workloadMemRequest, autoscalerWorkerNodeRoleLabel)
		Expect(client.Create(ctx, workload)).Should(Succeed())
		cleanupObjects[workload.Name] = workload
		testDuration = time.Now().Add(time.Duration(framework.WaitLong))
		Eventually(func() bool {
			v := scaleUpCounter.get()
			glog.Infof("[%s remaining] Expecting %v %q events; observed %v",
				remaining(testDuration), clusterExpansionSize-1, clusterAutoscalerScaledUpGroup, v)
			return v == uint32(clusterExpansionSize-1)
		}, framework.WaitLong, pollingInterval).Should(BeTrue())

		// The cluster-autoscaler can keep on generating
		// ScaledUpGroup events but in this scenario we are
		// expecting no more as we explicitly capped the
		// cluster size with maxNodesTotal (i.e.,
		// clusterExpansionSize -1). We run for a period of
		// time asserting that the cluster does not exceed the
		// capped size.
		testDuration = time.Now().Add(time.Duration(framework.WaitShort))
		Eventually(func() uint32 {
			v := maxNodesTotalReachedCounter.get()
			glog.Infof("[%s remaining] Waiting for %s to generate a %q event; observed %v",
				remaining(testDuration), clusterAutoscalerComponent, clusterAutoscalerMaxNodesTotalReached, v)
			return v
		}, framework.WaitShort, pollingInterval).Should(BeNumerically(">=", 1))

		testDuration = time.Now().Add(time.Duration(framework.WaitShort))
		Consistently(func() bool {
			v := scaleUpCounter.get()
			glog.Infof("[%s remaining] At max cluster size and expecting no more %q events; currently have %v, max=%v",
				remaining(testDuration), clusterAutoscalerScaledUpGroup, v, clusterExpansionSize-1)
			return v == uint32(clusterExpansionSize-1)
		}, framework.WaitShort, pollingInterval).Should(BeTrue())

		By("Deleting workload")
		scaleDownCounter := newScaleDownCounter(eventWatcher, uint32(clusterExpansionSize-1))
		Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(Succeed())
		delete(cleanupObjects, workload.Name)
		testDuration = time.Now().Add(time.Duration(framework.WaitLong))
		Eventually(func() bool {
			v := scaleDownCounter.get()
			glog.Infof("[%s remaining] Expecting %v %q events; observed %v",
				remaining(testDuration), clusterExpansionSize-1, clusterAutoscalerScaleDownEmpty, v)
			return v == uint32(clusterExpansionSize-1)
		}, framework.WaitLong, pollingInterval).Should(BeTrue())

		Eventually(func() bool {
			podList := corev1.PodList{}
			err = client.List(ctx, &podList, runtimeclient.InNamespace(workload.Namespace))
			Expect(err).NotTo(HaveOccurred())
			for i := range podList.Items {
				if strings.Contains(podList.Items[i].Name, workloadJobName) {
					glog.Infof("still have workload POD: %q", podList.Items[i].Name)
					return false
				}
			}
			return true
		}, framework.WaitMedium, pollingInterval).Should(BeZero())

		// Delete MachineAutoscalers to prevent scaling while we manually
		// scale-down the recently created MachineSets.
		for _, ma := range machineAutoscalers {
			err := deleteObject(ma.Name, ma)
			Expect(err).NotTo(HaveOccurred())
			delete(cleanupObjects, ma.Name)
		}

		// Delete the transient MachinSets.
		for _, ms := range machineSets {
			err := deleteObject(ms.Name, ms)
			Expect(err).NotTo(HaveOccurred())

			delete(cleanupObjects, ms.Name)

			framework.WaitForMachineSetDelete(client, ms)
		}

		// explicitly delete the ClusterAutoscaler
		// this is needed due to the autoscaler tests requiring singleton
		// deployments of the ClusterAutoscaler.
		caName := clusterAutoscaler.GetName()
		Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed())
		delete(cleanupObjects, caName)
		Eventually(func() bool {
			By(fmt.Sprintf("Waiting for ClusterAutoscaler to delete."))
			if ca, err := framework.GetClusterAutoscaler(client, caName); ca == nil && err != nil {
				return true
			}
			return false
		}, framework.WaitLong, pollingInterval).Should(BeTrue())
	})

	It("It scales from/to zero", func() {

		// Only run in platforms which support autoscaling from/to zero.
		clusterInfra, err := framework.GetInfrastructure(client)
		Expect(err).NotTo(HaveOccurred())

		platform := clusterInfra.Status.PlatformStatus.Type
		switch platform {
		case configv1.AWSPlatformType, configv1.GCPPlatformType, configv1.AzurePlatformType:
			glog.Infof("Platform is %v", platform)
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

		By(fmt.Sprintf("Creating ClusterAutoscaler "))
		clusterAutoscaler := clusterAutoscalerResource(100)
		Expect(client.Create(ctx, clusterAutoscaler)).Should(Succeed())
		cleanupObjects[clusterAutoscaler.GetName()] = clusterAutoscaler

		expectedReplicas := int32(3)
		By(fmt.Sprintf("Creating a MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v",
			machineSet.GetNamespace(), machineSet.GetName(), 0, expectedReplicas))
		asr := machineAutoscalerResource(machineSet, 0, expectedReplicas)
		Expect(client.Create(ctx, asr)).Should(Succeed())
		cleanupObjects[asr.GetName()] = asr

		By(fmt.Sprintf("Creating scale-out workload: jobs: %v, memory: %s", expectedReplicas, workloadMemRequest.String()))
		workload := newWorkLoad(expectedReplicas, workloadMemRequest, targetedNodeLabel)
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

		// explicitly delete the ClusterAutoscaler
		// this is needed due to the autoscaler tests requiring singleton
		// deployments of the ClusterAutoscaler.
		caName := clusterAutoscaler.GetName()
		Expect(deleteObject(caName, cleanupObjects[caName])).Should(Succeed())
		delete(cleanupObjects, caName)
		Eventually(func() bool {
			By(fmt.Sprintf("Waiting for ClusterAutoscaler to delete."))
			if ca, err := framework.GetClusterAutoscaler(client, caName); ca == nil && err != nil {
				return true
			}
			return false
		}, framework.WaitLong, pollingInterval).Should(BeTrue())
	})
})
