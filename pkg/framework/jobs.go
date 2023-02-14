package framework

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func NewWorkLoad(njobs int32, memoryRequest resource.Quantity, workloadJobName string,
	testLabel string, nodeSelector string, podLabel string) *batchv1.Job {
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
			BackoffLimit: pointer.Int32(4),
			Completions:  pointer.Int32(njobs),
			Parallelism:  pointer.Int32(njobs),
		},
	}

	if nodeSelector != "" {
		job.Spec.Template.Spec.NodeSelector = map[string]string{
			nodeSelector: "",
		}
	}

	if podLabel != "" {
		job.Spec.Template.ObjectMeta.Labels = map[string]string{
			podLabel: "",
		}
	}

	return job
}
