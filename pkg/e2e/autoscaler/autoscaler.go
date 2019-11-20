package autoscaler

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	g "github.com/onsi/ginkgo"
	o "github.com/onsi/gomega"
	e2e "github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/e2e/infra"
	mapiv1beta1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	caov1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1"
	caov1beta1 "github.com/openshift/cluster-autoscaler-operator/pkg/apis/autoscaling/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
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

func newWorkLoad(njobs int32, memoryRequest resource.Quantity) *batchv1.Job {
	return &batchv1.Job{
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
					NodeSelector: map[string]string{
						autoscalerWorkerNodeRoleLabel: "",
					},
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
			Namespace: e2e.MachineAPINamespace,
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
			Namespace:    e2e.MachineAPINamespace,
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

func dumpClusterAutoscalerLogs(client runtimeclient.Client, restClient *rest.RESTClient) {
	pods := corev1.PodList{}
	caLabels := map[string]string{
		"app": "cluster-autoscaler",
	}
	if err := client.List(context.TODO(), &pods, runtimeclient.MatchingLabels(caLabels)); err != nil {
		glog.Errorf("Error querying api for clusterAutoscaler pod object: %v", err)
		return
	}
	// We're only expecting one pod but let's log from all that
	// are found. If we see more than one that's indicative of
	// some unexpected problem and we may as well dump its logs.
	for i, pod := range pods.Items {
		req := restClient.Get().Namespace(e2e.MachineAPINamespace).Resource("pods").Name(pod.Name).SubResource("log")
		res := req.Do()
		raw, err := res.Raw()
		if err != nil {
			glog.Errorf("Unable to get pod logs: %v", err)
			continue
		}
		glog.Infof("\n\nDumping pod logs: %d/%d, logs from %q:\n%v", i, len(pods.Items), pod.Name, string(raw))
	}
}

var _ = g.Describe("[Feature:Machines] Autoscaler should", func() {
	cascadeDelete := metav1.DeletePropagationForeground

	g.It("scale up and down", func() {
		defer g.GinkgoRecover()

		clientset, err := e2e.LoadClientset()
		o.Expect(err).NotTo(o.HaveOccurred())

		var client runtimeclient.Client
		client, err = e2e.LoadClient()
		o.Expect(err).NotTo(o.HaveOccurred())

		var restClient *rest.RESTClient
		restClient, err = e2e.LoadRestClient()
		o.Expect(err).NotTo(o.HaveOccurred())

		deleteObject := func(name string, obj runtime.Object) error {
			glog.Infof("[cleanup] %q (%T)", name, obj)
			switch obj.(type) {
			case *caov1.ClusterAutoscaler:
				dumpClusterAutoscalerLogs(client, restClient)
			}
			return client.Delete(context.TODO(), obj, &runtimeclient.DeleteOptions{
				PropagationPolicy: &cascadeDelete,
			})
		}

		// Anything we create we must cleanup
		cleanupObjects := map[string]runtime.Object{}

		defer func() {
			for name, obj := range cleanupObjects {
				if err := deleteObject(name, obj); err != nil {
					glog.Infof("[cleanup] error deleting object %q (%T): %v", name, obj, err)
				}
			}
		}()

		g.By("Getting existing machinesets")
		existingMachineSets, err := e2e.GetMachineSets(client)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(existingMachineSets)).To(o.BeNumerically(">=", 1))

		g.By("Getting existing machines")
		existingMachines, err := e2e.GetMachines(client)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(existingMachines)).To(o.BeNumerically(">=", 1))

		g.By("Getting existing nodes")
		existingNodes, err := e2e.GetNodes(client)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(existingNodes)).To(o.BeNumerically(">=", 1))

		glog.Infof("Have %v existing machinesets", len(existingMachineSets))
		glog.Infof("Have %v existing machines", len(existingMachines))
		glog.Infof("Have %v existing nodes", len(existingNodes))
		o.Expect(len(existingNodes) == len(existingMachines)).To(o.BeTrue())

		// The remainder of the logic in this test requires 3
		// machinesets.
		var machineSets [3]*mapiv1beta1.MachineSet

		randomUUID := string(uuid.NewUUID())
		for i := 0; i < len(machineSets); i++ {
			targetMachineSet := existingMachineSets[i%len(existingMachineSets)]
			machineSetName := fmt.Sprintf("e2e-%s-w-%d", randomUUID[:5], i)
			machineSets[i] = e2e.NewMachineSet(targetMachineSet.Labels[e2e.ClusterKey],
				targetMachineSet.Namespace,
				machineSetName,
				targetMachineSet.Spec.Selector.MatchLabels,
				targetMachineSet.Spec.Template.ObjectMeta.Labels,
				&targetMachineSet.Spec.Template.Spec.ProviderSpec,
				1) // one replica
			machineSets[i].Spec.Template.Spec.ObjectMeta.Labels = map[string]string{
				autoscalerWorkerNodeRoleLabel: "",
			}
			o.Expect(client.Create(context.TODO(), machineSets[i])).Should(o.Succeed())
			cleanupObjects[machineSets[i].Name] = runtime.Object(machineSets[i])
		}

		g.By(fmt.Sprintf("Creating %v transient machinesets", len(machineSets)))
		testDuration := time.Now().Add(time.Duration(e2e.WaitLong))
		o.Eventually(func() bool {
			g.By(fmt.Sprintf("[%s remaining] Waiting for nodes to be Ready in %v transient machinesets",
				remaining(testDuration), len(machineSets)))
			var allNewNodes []*corev1.Node
			for i := 0; i < len(machineSets); i++ {
				nodes, err := infra.GetNodesFromMachineSet(client, *machineSets[i])
				if err != nil {
					return false
				}
				allNewNodes = append(allNewNodes, nodes...)
			}
			return len(allNewNodes) == len(machineSets) && infra.NodesAreReady(allNewNodes)
		}, e2e.WaitLong, pollingInterval).Should(o.BeTrue())

		// Now that we have created some transient machinesets
		// take stock of the number of nodes we now have.
		g.By("Getting nodes")
		nodes, err := e2e.GetNodes(client)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(nodes)).To(o.BeNumerically(">=", 1))

		g.By(fmt.Sprintf("Creating %v machineautoscalers", len(machineSets)))
		var clusterExpansionSize int
		for i := range machineSets {
			clusterExpansionSize += 1
			glog.Infof("Create MachineAutoscaler backed by MachineSet %s/%s - min:%v, max:%v", machineSets[i].Namespace, machineSets[i].Name, 1, 2)
			asr := machineAutoscalerResource(machineSets[i], 1, 2)
			o.Expect(client.Create(context.TODO(), asr)).Should(o.Succeed())
			cleanupObjects[asr.Name] = runtime.Object(asr)
		}
		o.Expect(clusterExpansionSize).To(o.BeNumerically(">", 1))

		// The total size of our cluster is
		// len(existingMachineSets) + clusterExpansionSize. We
		// cap that to $max-1 because we want to test that the
		// maxNodesTotal flag is respected by the
		// cluster-autoscaler
		maxNodesTotal := len(nodes) + clusterExpansionSize - 1

		eventWatcher := newEventWatcher(clientset)
		o.Expect(eventWatcher.run()).Should(o.BeTrue())
		defer eventWatcher.stop()

		// Log cluster-autoscaler events
		eventWatcher.onEvent(matchAnyEvent, func(e *corev1.Event) {
			if e.Source.Component == clusterAutoscalerComponent {
				glog.Infof("%s: %s", e.InvolvedObject.Name, e.Message)
			}
		}).enable()

		g.By(fmt.Sprintf("Creating ClusterAutoscaler configured with maxNodesTotal:%v", maxNodesTotal))
		clusterAutoscaler := clusterAutoscalerResource(maxNodesTotal)
		o.Expect(client.Create(context.TODO(), clusterAutoscaler)).Should(o.Succeed())
		cleanupObjects[clusterAutoscaler.Name] = runtime.Object(clusterAutoscaler)

		g.By(fmt.Sprintf("Deriving Memory capacity from machine %q", existingMachineSets[0].Name))
		workerNodes, err := e2e.GetWorkerNodes(client)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(workerNodes)).To(o.BeNumerically(">=", 1))
		memCapacity := workerNodes[0].Status.Capacity[corev1.ResourceMemory]
		o.Expect(memCapacity).ShouldNot(o.BeNil())
		o.Expect(memCapacity.String()).ShouldNot(o.BeEmpty())
		glog.Infof("Memory capacity of worker node %q is %s", workerNodes[0].Name, memCapacity.String())
		bytes, ok := memCapacity.AsInt64()
		o.Expect(ok).Should(o.BeTrue())
		// 70% - enough that the existing and new nodes will
		// be used, not enough to have more than 1 pod per
		// node.
		workloadMemRequest := resource.MustParse(fmt.Sprintf("%v", 0.7*float32(bytes)))

		g.By(fmt.Sprintf("Creating scale-out workload: jobs: %v, memory: %s", maxNodesTotal+1, workloadMemRequest.String()))
		scaledGroups := map[string]bool{}
		for i := range machineSets {
			scaledGroups[path.Join(machineSets[i].Namespace, machineSets[i].Name)] = false
		}
		scaleUpCounter := newScaleUpCounter(eventWatcher, 0, scaledGroups)
		maxNodesTotalReachedCounter := newMaxNodesTotalReachedCounter(eventWatcher, 0)
		// +1 to continuously generate the MaxNodesTotalReached
		workload := newWorkLoad(int32(maxNodesTotal+1), workloadMemRequest)
		o.Expect(client.Create(context.TODO(), workload)).Should(o.Succeed())
		cleanupObjects[workload.Name] = runtime.Object(workload)
		testDuration = time.Now().Add(time.Duration(e2e.WaitLong))
		o.Eventually(func() bool {
			v := scaleUpCounter.get()
			glog.Infof("[%s remaining] Expecting %v %q events; observed %v",
				remaining(testDuration), clusterExpansionSize-1, clusterAutoscalerScaledUpGroup, v)
			return v == uint32(clusterExpansionSize-1)
		}, e2e.WaitLong, pollingInterval).Should(o.BeTrue())

		// The cluster-autoscaler can keep on generating
		// ScaledUpGroup events but in this scenario we are
		// expecting no more as we explicitly capped the
		// cluster size with maxNodesTotal (i.e.,
		// clusterExpansionSize -1). We run for a period of
		// time asserting that the cluster does not exceed the
		// capped size.
		testDuration = time.Now().Add(time.Duration(e2e.WaitShort))
		o.Eventually(func() uint32 {
			v := maxNodesTotalReachedCounter.get()
			glog.Infof("[%s remaining] Waiting for %s to generate a %q event; observed %v",
				remaining(testDuration), clusterAutoscalerComponent, clusterAutoscalerMaxNodesTotalReached, v)
			return v
		}, e2e.WaitShort, pollingInterval).Should(o.BeNumerically(">=", 1))

		testDuration = time.Now().Add(time.Duration(e2e.WaitShort))
		o.Consistently(func() bool {
			v := scaleUpCounter.get()
			glog.Infof("[%s remaining] At max cluster size and expecting no more %q events; currently have %v, max=%v",
				remaining(testDuration), clusterAutoscalerScaledUpGroup, v, clusterExpansionSize-1)
			return v == uint32(clusterExpansionSize-1)
		}, e2e.WaitShort, pollingInterval).Should(o.BeTrue())

		g.By("Deleting workload")
		scaleDownCounter := newScaleDownCounter(eventWatcher, uint32(clusterExpansionSize-1))
		o.Expect(deleteObject(workload.Name, cleanupObjects[workload.Name])).Should(o.Succeed())
		delete(cleanupObjects, workload.Name)
		testDuration = time.Now().Add(time.Duration(e2e.WaitLong))
		o.Eventually(func() bool {
			v := scaleDownCounter.get()
			glog.Infof("[%s remaining] Expecting %v %q events; observed %v",
				remaining(testDuration), clusterExpansionSize-1, clusterAutoscalerScaleDownEmpty, v)
			return v == uint32(clusterExpansionSize-1)
		}, e2e.WaitLong, pollingInterval).Should(o.BeTrue())

		o.Eventually(func() bool {
			podList := corev1.PodList{}
			err = client.List(context.TODO(), &podList, runtimeclient.InNamespace(workload.Namespace))
			o.Expect(err).NotTo(o.HaveOccurred())
			for i := range podList.Items {
				if strings.Contains(podList.Items[i].Name, workloadJobName) {
					glog.Infof("still have workload POD: %q", podList.Items[i].Name)
					return false
				}
			}
			return true
		}, e2e.WaitMedium, pollingInterval).Should(o.BeZero())

		o.Expect(deleteObject(clusterAutoscaler.Name, cleanupObjects[clusterAutoscaler.Name])).Should(o.Succeed())
		delete(cleanupObjects, clusterAutoscaler.Name)
		o.Eventually(func() bool {
			podList := corev1.PodList{}
			err = client.List(context.TODO(), &podList, runtimeclient.InNamespace(""))
			o.Expect(err).NotTo(o.HaveOccurred())
			for i := range podList.Items {
				// This needs to disappear before we
				// start scaling the machinesets to
				// zero because we don't want it
				// scaling anything up as we are
				// trying to scale down.
				if strings.Contains(podList.Items[i].Name, "cluster-autoscaler-default") {
					glog.Infof("Waiting for cluster-autoscaler POD %q to disappear", podList.Items[i].Name)
					return true
				}
			}
			return false
		}, e2e.WaitMedium, pollingInterval).Should(o.BeTrue())

		g.By("Scaling transient machinesets to zero")
		for i := 0; i < len(machineSets); i++ {
			glog.Infof("Scaling transient machineset %q to zero", machineSets[i].Name)
			var freshMachineSet *mapiv1beta1.MachineSet
			err := retry.RetryOnConflict(retry.DefaultRetry, func() (err error) {
				freshMachineSet, err = e2e.GetMachineSet(client, machineSets[i].Name)
				return
			})
			o.Expect(err).NotTo(o.HaveOccurred())
			freshMachineSet.Spec.Replicas = pointer.Int32Ptr(0)
			err = client.Update(context.TODO(), freshMachineSet)
			o.Expect(err).NotTo(o.HaveOccurred())
		}

		g.By("Waiting for scaled up nodes to be deleted")
		testDuration = time.Now().Add(time.Duration(e2e.WaitLong))
		o.Eventually(func() int {
			currentNodes, err := e2e.GetNodes(client)
			o.Expect(err).NotTo(o.HaveOccurred())
			glog.Infof("[%s remaining] Waiting for cluster to reach original node count of %v; currently have %v",
				remaining(testDuration), len(existingNodes), len(currentNodes))
			return len(currentNodes)
		}, e2e.WaitLong, pollingInterval).Should(o.Equal(len(existingNodes)))

		g.By("Waiting for scaled up machines to be deleted")
		testDuration = time.Now().Add(time.Duration(e2e.WaitLong))
		o.Eventually(func() int {
			currentMachines, err := e2e.GetMachines(client)
			o.Expect(err).NotTo(o.HaveOccurred())
			glog.Infof("[%s remaining] Waiting for cluster to reach original machine count of %v; currently have %v",
				remaining(testDuration), len(existingMachines), len(currentMachines))
			return len(currentMachines)
		}, e2e.WaitLong, pollingInterval).Should(o.Equal(len(existingMachines)))
	})
})
