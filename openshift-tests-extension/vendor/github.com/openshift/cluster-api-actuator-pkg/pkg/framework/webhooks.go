package framework

import (
	"context"
	"fmt"

	"github.com/openshift/machine-api-operator/pkg/webhooks"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DefaultValidatingWebhookConfiguration is a default validating webhook configuration resource provided by MAO.
var DefaultValidatingWebhookConfiguration = webhooks.NewMachineValidatingWebhookConfiguration()

// DefaultMutatingWebhookConfiguration is a default mutating webhook configuration resource provided by MAO.
var DefaultMutatingWebhookConfiguration = webhooks.NewMachineMutatingWebhookConfiguration()

// GetMutatingWebhookConfiguration gets MutatingWebhookConfiguration object by name.
func GetMutatingWebhookConfiguration(ctx context.Context, c client.Client, name string) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	key := client.ObjectKey{Name: name}
	existing := &admissionregistrationv1.MutatingWebhookConfiguration{}

	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, key, existing); err != nil {
			klog.Errorf("Error querying api for MutatingWebhookConfiguration object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting MutatingWebhookConfiguration %q: %w", name, err)
	}

	return existing, nil
}

// GetValidatingWebhookConfiguration gets ValidatingWebhookConfiguration object by name.
func GetValidatingWebhookConfiguration(ctx context.Context, c client.Client, name string) (*admissionregistrationv1.ValidatingWebhookConfiguration, error) {
	key := client.ObjectKey{Name: name}
	existing := &admissionregistrationv1.ValidatingWebhookConfiguration{}

	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Get(ctx, key, existing); err != nil {
			klog.Errorf("Error querying api for ValidatingWebhookConfiguration object %q: %v, retrying...", name, err)
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("error getting ValidatingWebhookConfiguration %q: %w", name, err)
	}

	return existing, nil
}

// DeleteValidatingWebhookConfiguration deletes the specified ValidatingWebhookConfiguration object.
func DeleteValidatingWebhookConfiguration(ctx context.Context, c client.Client, webhookConfiguraiton *admissionregistrationv1.ValidatingWebhookConfiguration) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Delete(ctx, webhookConfiguraiton); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			klog.Errorf("error querying api for ValidatingWebhookConfiguration object %q: %v, retrying...", webhookConfiguraiton.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// DeleteMutatingWebhookConfiguration deletes the specified MutatingWebhookConfiguration object.
func DeleteMutatingWebhookConfiguration(ctx context.Context, c client.Client, webhookConfiguraiton *admissionregistrationv1.MutatingWebhookConfiguration) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		if err := c.Delete(ctx, webhookConfiguraiton); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			klog.Errorf("error querying api for MutatingWebhookConfiguration object %q: %v, retrying...", webhookConfiguraiton.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// UpdateMutatingWebhookConfiguration updates the specified mutating webhook configuration.
func UpdateMutatingWebhookConfiguration(ctx context.Context, c client.Client, updated *admissionregistrationv1.MutatingWebhookConfiguration) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		existing, err := GetMutatingWebhookConfiguration(ctx, c, updated.Name)
		if err != nil {
			klog.Errorf("Error getting MutatingWebhookConfiguration: %v", err)
			return false, nil
		}

		if err := c.Patch(ctx, existing, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching MutatingWebhookConfiguration object %q: %v, retrying...", updated.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// UpdateValidatingWebhookConfiguration updates the specified mutating webhook configuration.
func UpdateValidatingWebhookConfiguration(ctx context.Context, c client.Client, updated *admissionregistrationv1.ValidatingWebhookConfiguration) error {
	return wait.PollUntilContextTimeout(ctx, RetryShort, WaitShort, true, func(ctx context.Context) (bool, error) {
		existing, err := GetValidatingWebhookConfiguration(ctx, c, updated.Name)
		if err != nil {
			klog.Errorf("Error getting ValidatingWebhookConfiguration: %v", err)
			return false, nil
		}

		if err := c.Patch(ctx, existing, client.MergeFrom(updated)); err != nil {
			klog.Errorf("error patching ValidatingWebhookConfiguration object %q: %v, retrying...", updated.Name, err)
			return false, nil
		}

		return true, nil
	})
}

// IsMutatingWebhookConfigurationSynced expects a matching MutatingWebhookConfiguration to be present in the cluster.
func IsMutatingWebhookConfigurationSynced(ctx context.Context, c client.Client) bool {
	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitMedium, true, func(ctx context.Context) (bool, error) {
		existing, err := GetMutatingWebhookConfiguration(ctx, c, DefaultMutatingWebhookConfiguration.Name)
		if err != nil {
			klog.Errorf("Error getting MutatingWebhookConfiguration: %v", err)
			return false, nil
		}

		// Due to caBundle injection by service-ca-operator, we have to use DeepDerivative,
		// which will ignore change in spec.webhooks[x].serviceReference.caBundle in comparison
		// to empty value, as the default webhook configuration does not have this field set.
		equal := equality.Semantic.DeepDerivative(DefaultMutatingWebhookConfiguration.Webhooks, existing.Webhooks)
		if !equal {
			klog.Infof("MutatingWebhookConfiguration is not yet equal, retrying...")
		}

		return equal, nil
	}); err != nil {
		klog.Errorf("Error waiting for match with expected MutatingWebhookConfigurationMatched: %v", err)
		return false
	}

	return true
}

// IsValidatingWebhookConfigurationSynced expects a matching MutatingWebhookConfiguration to be present in the cluster.
func IsValidatingWebhookConfigurationSynced(ctx context.Context, c client.Client) bool {
	if err := wait.PollUntilContextTimeout(ctx, RetryShort, WaitMedium, true, func(ctx context.Context) (bool, error) {
		existing, err := GetValidatingWebhookConfiguration(ctx, c, DefaultValidatingWebhookConfiguration.Name)
		if err != nil {
			klog.Errorf("Error getting MutatingWebhookConfiguration: %v", err)
			return false, nil
		}

		// Due to caBundle injection by service-ca-operator, we have to use DeepDerivative,
		// which will ignore change in spec.webhooks[x].serviceReference.caBundle in comparison
		// to empty value, as the default webhook configuration does not have this field set.
		equal := equality.Semantic.DeepDerivative(DefaultValidatingWebhookConfiguration.Webhooks, existing.Webhooks)
		if !equal {
			klog.Infof("ValidatingWebhookConfiguration is not yet equal, retrying...")
		}

		return equal, nil
	}); err != nil {
		klog.Errorf("Error waiting for match with expected ValidatingWebhookConfigurationMatched: %v", err)
		return false
	}

	return true
}
