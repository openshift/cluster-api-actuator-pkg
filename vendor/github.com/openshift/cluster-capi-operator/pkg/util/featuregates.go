/*
Copyright 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package util

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/api/features"
	configv1client "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	featuregates "github.com/openshift/library-go/pkg/operator/configobserver/featuregates"
	"github.com/openshift/library-go/pkg/operator/events"
)

// GetFeatureGates returns feature gates for the current release version.
// Calling this function will additionally start two background goroutines which
// together watch cluster feature gates and call ctxCancel() if they change.
// These goroutines will be cancelled when the given context is cancelled.
func GetFeatureGates(ctx context.Context, log logr.Logger, componentName string, restConfig *rest.Config, ctxCancel context.CancelFunc) (featuregates.FeatureGate, error) {
	desiredVersion := GetReleaseVersion()
	missingVersion := "0.0.1-snapshot"

	configClient, err := configv1client.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create config client: %w", err)
	}

	configInformers := configinformers.NewSharedInformerFactory(configClient, 10*time.Minute)

	featureGateAccessor := featuregates.NewFeatureGateAccess(
		desiredVersion, missingVersion,
		configInformers.Config().V1().ClusterVersions(),
		configInformers.Config().V1().FeatureGates(),
		events.NewLoggingEventRecorder(componentName, clock.RealClock{}),
	)

	featureGateAccessor.SetChangeHandler(func(featureChange featuregates.FeatureChange) {
		// Do nothing if we are observing initialisation of the feature gates.
		if featureChange.Previous == nil {
			return
		}

		log.Info("Detected feature gates changed. Restarting manager.")
		ctxCancel()
	})

	go featureGateAccessor.Run(ctx)
	go configInformers.Start(ctx.Done())

	// Don't wait longer than 1 minute for the feature gates to be initialized.
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	select {
	case <-featureGateAccessor.InitialFeatureGatesObserved():
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for feature gates to be initialized: %w", ctx.Err())
	}

	currentFeatureGates, err := featureGateAccessor.CurrentFeatureGates()
	if err != nil {
		return nil, fmt.Errorf("failed to get current feature gates: %w", err)
	}

	log.Info("FeatureGates initialized", "features", currentFeatureGates.KnownFeatures())

	return currentFeatureGates, nil
}

// IsCAPIEnabledForPlatform returns true if CAPI support is enabled for the given
// platform in the given feature gates.
func IsCAPIEnabledForPlatform(currentFeatureGates featuregates.FeatureGate, platform configv1.PlatformType) bool {
	switch platform {
	case configv1.AWSPlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementAWS)
	case configv1.AzurePlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementAzure)
	case configv1.BareMetalPlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementBareMetal)
	case configv1.GCPPlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementGCP)
	case configv1.PowerVSPlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementPowerVS)
	case configv1.OpenStackPlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementOpenStack)
	case configv1.VSpherePlatformType:
		return currentFeatureGates.Enabled(features.FeatureGateClusterAPIMachineManagementVSphere)

	default:
		return false
	}
}
