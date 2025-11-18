package framework

import (
	"bytes"
	"context"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPods returns a list of pods matching the provided selector.
func GetPods(client runtimeclient.Client, selector map[string]string) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	err := client.List(context.TODO(), pods, runtimeclient.MatchingLabels(selector))

	return pods, err
}

type PodCleanupFunc func() error

type PodLastLogFunc func(container string, lines int, previous bool) (string, error)

// RunPodOnNode runs a pod according passed spec on particular node.
// returns created pod object, function for retrieve last logs, cleanup function and error if occurred.
func RunPodOnNode(clientset *kubernetes.Clientset, node *corev1.Node, namespace string, podSpec corev1.PodSpec) (*corev1.Pod, PodLastLogFunc, PodCleanupFunc, error) {
	var err error

	podSpec.NodeName = node.Name

	pod := &corev1.Pod{
		Spec: podSpec,
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "machine-api-e2e-",
		},
	}

	pod, err = clientset.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, nil, nil, err
	}

	cleanup := func() error {
		return clientset.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	}

	lastLog := func(container string, lines int, previous bool) (string, error) {
		tailLines := int64(lines)
		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: container,
			Follow:    true,
			Previous:  previous,
			TailLines: &tailLines,
		})

		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			return "", err
		}

		defer podLogs.Close()

		buf := new(bytes.Buffer)

		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return "", err
		}

		logs := buf.String()

		return logs, nil
	}

	return pod, lastLog, cleanup, err
}
