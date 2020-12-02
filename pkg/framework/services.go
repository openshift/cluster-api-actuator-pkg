package framework

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServices returns a list of services matching the provided selector
func GetServices(client runtimeclient.Client, selector map[string]string) (*corev1.ServiceList, error) {
	services := &corev1.ServiceList{}
	err := client.List(context.TODO(), services, runtimeclient.MatchingLabels(selector))
	if err != nil {
		return nil, fmt.Errorf("error getting Services %v", err)
	}
	return services, err
}

// GetService gets service object by name and namespace.
func GetService(c runtimeclient.Client, name, namespace string) (*corev1.Service, error) {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	s := &corev1.Service{}

	if err := wait.PollImmediate(RetryMedium, WaitShort, func() (bool, error) {
		if err := c.Get(context.TODO(), key, s); err != nil {
			klog.Errorf("Error querying api for Service object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting Service %q: %v", name, err)
	}
	return s, nil
}

// IsServiceAvailable returns true if the service exists
func IsServiceAvailable(c runtimeclient.Client, name, namespace string) bool {
	if err := wait.PollImmediate(RetryMedium, WaitLong, func() (bool, error) {
		s, err := GetService(c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting Service: %v", err)
			return false, nil
		}
		if s.Spec.ClusterIP == "" {
			klog.Errorf("Service doesn't have a clusterIP: %v", err)
			return false, nil
		}
		klog.Infof("Service %q is available. Status: %s",
			s.Name, serviceInfo(s))
		return true, nil
	}); err != nil {
		klog.Errorf("Error checking IsServiceAvailable: %v", err)
		return false
	}
	return true
}

func serviceInfo(s *corev1.Service) string {
	return fmt.Sprintf("(ClusterIP: %s)",
		s.Spec.ClusterIP)
}
