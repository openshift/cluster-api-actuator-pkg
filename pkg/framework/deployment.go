package framework

import (
	"context"
	"fmt"
	"reflect"
	"time"

	kappsapi "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment gets deployment object by name and namespace with custom timeout
func GetDeployment(c client.Client, name, namespace string, timeout time.Duration) (*kappsapi.Deployment, error) {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	d := &kappsapi.Deployment{}

	if err := wait.PollImmediate(RetryShort, timeout, func() (bool, error) {
		if err := c.Get(context.TODO(), key, d); err != nil {
			klog.Errorf("Error querying api for Deployment object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting deployment %q: %v", name, err)
	}
	return d, nil
}

// DeleteDeployment deletes the specified deployment with custom timeout
func DeleteDeployment(c client.Client, deployment *kappsapi.Deployment, timeout time.Duration) error {
	return wait.PollImmediate(RetryShort, timeout, func() (bool, error) {
		if err := c.Delete(context.TODO(), deployment); err != nil {
			klog.Errorf("error querying api for deployment object %q: %v, retrying...", deployment.Name, err)
			return false, nil
		}
		return true, nil
	})
}

// UpdateDeployment updates the specified deployment with custom timeout
func UpdateDeployment(c client.Client, name, namespace string,
	updated *kappsapi.Deployment, timeout time.Duration) error {
	return wait.PollImmediate(RetryShort, timeout, func() (bool, error) {
		d, err := GetDeployment(c, name, namespace, WaitShort)
		if err != nil {
			klog.Errorf("Error getting deployment: %v", err)
			return false, nil
		}
		if err := c.Patch(context.TODO(), d, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching deployment object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	})
}

// IsDeploymentAvailable returns true if the deployment has one or more availabe replicas with custom timeout
func IsDeploymentAvailable(c client.Client, name, namespace string, timeout time.Duration) bool {
	if err := wait.PollImmediate(RetryShort, timeout, func() (bool, error) {
		d, err := GetDeployment(c, name, namespace, WaitShort)
		if err != nil {
			klog.Errorf("Error getting deployment: %v", err)
			return false, nil
		}
		if d.Status.AvailableReplicas < 1 {
			klog.Errorf("Deployment %q is not available. Status: %s",
				d.Name, deploymentInfo(d))
			return false, nil
		}
		klog.Infof("Deployment %q is available. Status: %s",
			d.Name, deploymentInfo(d))
		return true, nil
	}); err != nil {
		klog.Errorf("Error checking isDeploymentAvailable: %v", err)
		return false
	}
	return true
}

// IsDeploymentSynced returns true if provided deployment spec
// matched one found on cluster with custom timeout
func IsDeploymentSynced(c client.Client, dep *kappsapi.Deployment,
	name, namespace string, timeout time.Duration) bool {
	d, err := GetDeployment(c, name, namespace, timeout)
	if err != nil {
		klog.Errorf("Error getting deployment: %v", err)
		return false
	}
	if !reflect.DeepEqual(d.Spec, dep.Spec) {
		klog.Errorf("Deployment %q is not updated. Spec is not equal to: %v",
			d.Name, dep.Spec)
		return false
	}
	klog.Infof("Deployment %q is updated. Spec is matched", d.Name)
	return true
}

// DeploymentHasContainer returns true if the deployment has container with the specified name
func DeploymentHasContainer(deployment *kappsapi.Deployment, containerName string) bool {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == containerName {
			return true
		}
	}
	return false
}

func deploymentInfo(d *kappsapi.Deployment) string {
	return fmt.Sprintf("(replicas: %d, updated: %d, ready: %d, available: %d, unavailable: %d)",
		d.Status.Replicas, d.Status.UpdatedReplicas, d.Status.ReadyReplicas,
		d.Status.AvailableReplicas, d.Status.UnavailableReplicas)
}
