package framework

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1alpha1 "github.com/openshift/api/operator/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	revision2      = "revision-2"
	currentVersion = "current"
)

func TestClusterAPIReconciled(t *testing.T) {
	tests := []struct {
		name       string
		generation int64
		observed   int64
		desired    operatorv1alpha1.RevisionName
		current    operatorv1alpha1.RevisionName
		wantError  bool
	}{
		{name: "ready", generation: 2, observed: 2, desired: revision2, current: revision2},
		{name: "generation not observed", generation: 2, observed: 1, desired: revision2, current: revision2, wantError: true},
		{name: "revision not applied", generation: 2, observed: 2, desired: revision2, current: "revision-1", wantError: true},
		{name: "empty revisions", generation: 2, observed: 2, wantError: true},
		{name: "uninitialized", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &operatorv1alpha1.ClusterAPI{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster", Generation: tt.generation},
				Status: operatorv1alpha1.ClusterAPIStatus{
					ObservedRevisionGeneration: tt.observed,
					DesiredRevision:            tt.desired,
					CurrentRevision:            tt.current,
				},
			}
			c := &proxyTestClient{clusterAPI: obj}

			err := CheckClusterAPIReconciled(context.Background(), c)
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

func TestClusterAPIMachineManagementEnabled(t *testing.T) {
	tests := []struct {
		name        string
		details     []configv1.FeatureGateDetails
		wantEnabled bool
		wantError   bool
	}{
		{name: "enabled", details: []configv1.FeatureGateDetails{{Version: currentVersion, Enabled: []configv1.FeatureGateAttributes{{Name: clusterAPIMachineManagementFeatureGate}}}}, wantEnabled: true},
		{name: "disabled", details: []configv1.FeatureGateDetails{{Version: currentVersion, Disabled: []configv1.FeatureGateAttributes{{Name: clusterAPIMachineManagementFeatureGate}}}}},
		{name: "old version does not enable current version", details: []configv1.FeatureGateDetails{
			{Version: "old", Enabled: []configv1.FeatureGateAttributes{{Name: clusterAPIMachineManagementFeatureGate}}},
			{Version: currentVersion, Disabled: []configv1.FeatureGateAttributes{{Name: clusterAPIMachineManagementFeatureGate}}},
		}},
		{name: "missing status", wantError: true},
		{name: "missing current version", details: []configv1.FeatureGateDetails{{Version: "old"}}, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version := &configv1.ClusterVersion{ObjectMeta: metav1.ObjectMeta{Name: "version"}, Status: configv1.ClusterVersionStatus{Desired: configv1.Release{Version: currentVersion}}}
			gate := &configv1.FeatureGate{ObjectMeta: metav1.ObjectMeta{Name: "cluster"}, Status: configv1.FeatureGateStatus{FeatureGates: tt.details}}
			c := &proxyTestClient{version: version, gate: gate}

			enabled, err := ClusterAPIMachineManagementEnabled(context.Background(), c)
			if enabled != tt.wantEnabled || (err != nil) != tt.wantError {
				t.Fatalf("enabled = %v, error = %v; want enabled = %v, error = %v", enabled, err, tt.wantEnabled, tt.wantError)
			}
		})
	}

	c := &proxyTestClient{}
	if _, err := ClusterAPIMachineManagementEnabled(context.Background(), c); err == nil {
		t.Fatal("missing ClusterVersion must return an error")
	}

	c.version = &configv1.ClusterVersion{}
	if _, err := ClusterAPIMachineManagementEnabled(context.Background(), c); err == nil {
		t.Fatal("missing FeatureGate must return an error")
	}
}
