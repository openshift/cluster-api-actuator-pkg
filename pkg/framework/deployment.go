package framework

import (
	"context"
	"fmt"
	"reflect"

	kappsapi "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment gets deployment object by name and namespace.
func GetDeployment(ctx context.Context, c client.Client, name, namespace string) (*kappsapi.Deployment, error) {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	d := &kappsapi.Deployment{}

	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, key, d); err != nil {
			klog.Errorf("Error querying api for Deployment object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting deployment %q: %w", name, err)
	}

	return d, nil
}

// DeleteDeployment deletes the specified deployment.
func DeleteDeployment(ctx context.Context, c client.Client, deployment *kappsapi.Deployment) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Delete(ctx, deployment); err != nil {
			klog.Errorf("error querying api for deployment object %q: %v, retrying...", deployment.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// UpdateDeployment updates the specified deployment.
func UpdateDeployment(ctx context.Context, c client.Client, name, namespace string, updated *kappsapi.Deployment) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitMedium, true, func(ctx context.Context) (bool, error) {
		d, err := GetDeployment(ctx, c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting deployment: %v", err)
			return false, nil
		}

		if err := c.Patch(ctx, d, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching deployment object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	})
}

// IsDeploymentAvailable returns true if the deployment has one or more available replicas.
func IsDeploymentAvailable(ctx context.Context, c client.Client, name, namespace string) bool {
	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitLong, true, func(ctx context.Context) (bool, error) {
		d, err := GetDeployment(ctx, c, name, namespace)
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

// IsDeploymentSynced returns true if provided deployment spec matched one found on cluster.
func IsDeploymentSynced(ctx context.Context, c client.Client, dep *kappsapi.Deployment, name, namespace string) bool {
	d, err := GetDeployment(ctx, c, name, namespace)
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

// DeploymentHasContainer returns true if the deployment has container with the specified name.
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
