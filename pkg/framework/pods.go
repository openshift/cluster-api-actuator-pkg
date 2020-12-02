package framework

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPods returns a list of pods matching the provided selector
func GetPods(client runtimeclient.Client, selector map[string]string) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	err := client.List(context.TODO(), pods, runtimeclient.MatchingLabels(selector))
	return pods, err
}
