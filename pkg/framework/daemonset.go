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

func daemonSetInfo(ds *kappsapi.DaemonSet) string {
	return fmt.Sprintf("(desired: %d, updated: %d, available: %d, unavailable: %d)",
		ds.Status.DesiredNumberScheduled, ds.Status.UpdatedNumberScheduled,
		ds.Status.NumberAvailable, ds.Status.NumberUnavailable)
}

// GetDaemonSet gets a daemonSet object by name and namespace.
func GetDaemonSet(c client.Client, name, namespace string) (*kappsapi.DaemonSet, error) {
	ds := &kappsapi.DaemonSet{}

	if err := wait.PollImmediate(RetryShort, WaitShort, func() (bool, error) {
		if err := c.Get(context.TODO(), types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}, ds); err != nil {
			klog.Errorf("Error querying object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting object %q: %v", name, err)
	}
	return ds, nil
}

// IsDaemonSetAvailable returns true if the daemonSet has no unavailable replicas
func IsDaemonSetAvailable(c client.Client, name, namespace string) bool {
	if err := wait.PollImmediate(RetryShort, WaitLong, func() (bool, error) {
		ds, err := GetDaemonSet(c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting daemonSet: %v", err)
			return false, nil
		}

		if !(ds.Generation <= ds.Status.ObservedGeneration &&
			ds.Status.UpdatedNumberScheduled == ds.Status.DesiredNumberScheduled &&
			ds.Status.NumberUnavailable == 0) {
			klog.Errorf("DaemonSet %q is not available. Status: %s",
				ds.Name, daemonSetInfo(ds))
			return false, nil
		}

		klog.Infof("DaemonSet %q is available. Status: %s", ds.Name, daemonSetInfo(ds))
		return true, nil
	}); err != nil {
		klog.Errorf("Error checking IsDaemonSetAvailable: %v", err)
		return false
	}
	return true
}

// DeleteDaemonSet deletes the specified daemonSet
func DeleteDaemonSet(c client.Client, ds *kappsapi.DaemonSet) error {
	return wait.PollImmediate(RetryShort, WaitShort, func() (bool, error) {
		if err := c.Delete(context.TODO(), ds); err != nil {
			klog.Errorf("error querying api object %q: %v, retrying...", ds.Name, err)
			return false, nil
		}
		return true, nil
	})
}

// IsDaemonSetSynced returns true if the provided daemonSet spec matches the one found on cluster
func IsDaemonSetSynced(c client.Client, expected *kappsapi.DaemonSet, name, namespace string) bool {
	got, err := GetDaemonSet(c, name, namespace)
	if err != nil {
		klog.Errorf("Error getting daemonSet: %v", err)
		return false
	}
	if !reflect.DeepEqual(expected.Spec, got.Spec) {
		klog.Errorf("DaemonSet %q is not updated. Spec is not equal to: %v",
			got.Name, expected.Spec)
		return false
	}
	klog.Infof("DaemonSet %q is updated. Spec matches the expected one", got.Name)
	return true
}

// UpdateDaemonSet updates the specified daemonSet
func UpdateDaemonSet(c client.Client, name, namespace string, updated *kappsapi.DaemonSet) error {
	return wait.PollImmediate(RetryShort, WaitMedium, func() (bool, error) {
		d, err := GetDaemonSet(c, name, namespace)
		if err != nil {
			klog.Errorf("Error getting daemonSet: %v", err)
			return false, nil
		}
		if err := c.Patch(context.TODO(), d, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching daemonSet object %q: %v, retrying...", name, err)
			return false, nil
		}
		return true, nil
	})
}
