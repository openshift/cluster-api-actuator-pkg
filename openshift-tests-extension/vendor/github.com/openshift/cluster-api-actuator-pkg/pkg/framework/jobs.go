package framework

import (
	"context"
	"fmt"
	"strings"

	. "github.com/onsi/gomega"
	machinev1 "github.com/openshift/api/machine/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/utils/ptr"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewWorkLoad(njobs int32, memoryRequest resource.Quantity, workloadJobName string,
	testLabel string, podLabel string, nodeSelectorReqs ...corev1.NodeSelectorRequirement) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workloadJobName,
			Namespace: MachineAPINamespace,
			Labels:    map[string]string{testLabel: ""},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  workloadJobName,
							Image: "registry.access.redhat.com/ubi8/ubi-minimal:latest",
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
						{
							Key:    ClusterAPIActuatorPkgTaint,
							Effect: corev1.TaintEffectPreferNoSchedule,
						},
					},
				},
			},
			BackoffLimit: ptr.To[int32](4),
			Completions:  ptr.To[int32](njobs),
			Parallelism:  ptr.To[int32](njobs),
		},
	}

	if len(nodeSelectorReqs) > 0 {
		// Create the empty node selector terms in the spec
		job.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: nodeSelectorReqs,
						},
					},
				},
			},
		}
	}

	if podLabel != "" {
		job.Spec.Template.ObjectMeta.Labels = map[string]string{
			podLabel: "",
		}
	}

	return job
}

// WaitForWorkload waits for a workload's pods to be scheduled and running on the given MachineSet.
func WaitForWorkload(ctx context.Context, c runtimeclient.Client, machineSet *machinev1.MachineSet, expectedReplicas int32, workloadName string) {
	WaitForWorkloadOverMachineSets(ctx, c, []*machinev1.MachineSet{machineSet}, expectedReplicas, workloadName)
}

// WaitForWorkloadOverMachineSets waits for a workload's pods to be scheduled and running across multiple MachineSets.
func WaitForWorkloadOverMachineSets(ctx context.Context, c runtimeclient.Client, machineSets []*machinev1.MachineSet, expectedReplicas int32, workloadName string) {
	job := &batchv1.Job{}
	key := runtimeclient.ObjectKey{Namespace: MachineAPINamespace, Name: workloadName}
	err := c.Get(ctx, key, job)
	Expect(err).ToNot(HaveOccurred(), "getting workload job should not error")

	Eventually(func() error {
		if err := c.Get(ctx, key, job); err != nil {
			return err
		}

		podList := &corev1.PodList{}
		listOpts := []runtimeclient.ListOption{
			runtimeclient.InNamespace(job.Namespace),
			runtimeclient.MatchingLabels(job.Spec.Template.ObjectMeta.Labels),
		}

		if err := c.List(ctx, podList, listOpts...); err != nil {
			return err
		}

		// check if there are the correct number of pods
		if len(podList.Items) != int(*job.Spec.Completions) {
			// there's a chance that some job pods may have completed, but realistically this should not happen
			// if so, just fail the test
			if job.Status.Succeeded > 0 || job.Status.Failed > 0 {
				return StopTrying(fmt.Sprintf("job %q with %d succeeded and %d failed pods", workloadName, job.Status.Succeeded, job.Status.Failed))
			}

			return fmt.Errorf("expected %d job pods, got %d", *job.Spec.Completions, len(podList.Items))
		}

		// build flattened list of machines from all machine sets
		allMachines := []*machinev1.Machine{}
		for _, machineSet := range machineSets {
			machines, err := GetMachinesFromMachineSet(ctx, c, machineSet)
			if err != nil {
				return err
			}
			// we need to check the nodeRef is set, otherwise try again until it is set
			for _, machine := range machines {
				if machine.Status.NodeRef == nil {
					return fmt.Errorf("machine %q has no nodeRef yet, try again", machine.Name)
				}
				allMachines = append(allMachines, machine)
			}
			klog.Infof("MachineSet %q, Machines and Nodes: %s", machineSet.Name, getMachinesAndNodesAsString(machines))
		}

		var runningPods int32 = 0
		for _, pod := range podList.Items {
			// make sure expected number of pods are running
			if pod.Status.Phase != corev1.PodRunning {
				conditionsInfo := []string{}
				for _, condition := range pod.Status.Conditions {
					if condition.Status != corev1.ConditionTrue && condition.Reason != "" {
						conditionsInfo = append(conditionsInfo, fmt.Sprintf("%s=%s (%s)", condition.Type, condition.Status, condition.Reason))
					}
				}
				klog.Warningf("Pod %q not running. phase: [%s], conditions: [%s]", pod.Name, pod.Status.Phase, strings.Join(conditionsInfo, ", "))

				continue
			}

			// make sure pods are running on the any nodes associated with the MachineSet(s)
			if !isPodRunningOnMachineSet(&pod, allMachines) {
				klog.Warningf("pod %q is not running on any MachineSet node", pod.Name)
				continue
			}

			klog.Infof("Pod %q is running on Node %q, as expected", pod.Name, pod.Spec.NodeName)
			runningPods++
		}

		if runningPods != expectedReplicas {
			return fmt.Errorf("expected %d running job pods, got %d", expectedReplicas, runningPods)
		}
		klog.Infof("Got %d %q workload Pods, as expected", runningPods, corev1.PodRunning)

		return nil
	}, WaitLong, RetryMedium).ShouldNot(HaveOccurred())
}

// isPodRunningOnMachineSet checks if the pod is running on any of the nodes as part of the MachineSet.
func isPodRunningOnMachineSet(pod *corev1.Pod, machineList []*machinev1.Machine) bool {
	for _, machine := range machineList {
		if machine == nil || machine.Status.NodeRef == nil {
			continue
		}

		if machine.Status.NodeRef.Name == pod.Spec.NodeName {
			return true
		}
	}

	return false
}

func getMachinesAndNodesAsString(machineList []*machinev1.Machine) string {
	machineNames := []string{}
	for _, machine := range machineList {
		machineNames = append(machineNames, fmt.Sprintf("[%q: %q]", machine.Name, machine.Status.NodeRef.Name))
	}

	return strings.Join(machineNames, ", ")
}
