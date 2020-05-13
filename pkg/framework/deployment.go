package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	kappsapi "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployment gets deployment object by name and namespace.
func GetDeployment(c client.Client, name, namespace string) (*kappsapi.Deployment, error) {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	d := &kappsapi.Deployment{}

	if err := wait.PollImmediate(1*time.Second, WaitShort, func() (bool, error) {
		if err := c.Get(context.TODO(), key, d); err != nil {
			glog.Errorf("Error querying api for Deployment object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting deployment %q: %v", name, err)
	}
	return d, nil
}

// DeleteDeployment deletes the specified deployment
func DeleteDeployment(c client.Client, deployment *kappsapi.Deployment) error {
	return wait.PollImmediate(1*time.Second, WaitShort, func() (bool, error) {
		if err := c.Delete(context.TODO(), deployment); err != nil {
			glog.Errorf("error querying api for deployment object %q: %v, retrying...", deployment.Name, err)
			return false, nil
		}
		return true, nil
	})
}

// IsDeploymentAvailable returns true if the deployment has one or more availabe replicas
func IsDeploymentAvailable(c client.Client, name, namespace string) bool {
	if err := wait.PollImmediate(1*time.Second, WaitLong, func() (bool, error) {
		d, err := GetDeployment(c, name, namespace)
		if err != nil {
			glog.Errorf("Error getting deployment: %v", err)
			return false, nil
		}
		if d.Status.AvailableReplicas < 1 {
			glog.Errorf("Deployment %q is not available. Status: %s",
				d.Name, deploymentInfo(d))
			return false, nil
		}
		glog.Infof("Deployment %q is available. Status: %s",
			d.Name, deploymentInfo(d))
		return true, nil
	}); err != nil {
		glog.Errorf("Error checking isDeploymentAvailable: %v", err)
		return false
	}
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
