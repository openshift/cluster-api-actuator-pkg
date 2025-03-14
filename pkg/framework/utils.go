/*
Copyright 2022 Red Hat, Inc.

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

package framework

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errContextCancelled = errors.New("context cancelled")
)

// GomegaAssertions is a subset of the gomega.Gomega interface.
// It is the set allowed for checks and conditions in the RunCheckUntil
// helper function.
type GomegaAssertions interface {
	Ω(actual interface{}, extra ...interface{}) gomega.Assertion //nolint:asciicheck
	Expect(actual interface{}, extra ...interface{}) gomega.Assertion
	ExpectWithOffset(offset int, actual interface{}, extra ...interface{}) gomega.Assertion
}

// RunCheckUntil runs the check function until the condition succeeds or the context is cancelled.
// If the check fails before the condition succeeds, the test will fail.
// The check and condition functions must use the passed Gomega for any assertions so that we can handle failures
// within the functions appropriately.
func RunCheckUntil(ctx context.Context, check, condition func(context.Context, GomegaAssertions) bool) bool {
	return gomega.Eventually(func() error {
		checkErr := runAssertion(ctx, check)
		conditionErr := runAssertion(ctx, condition)

		switch {
		case conditionErr == nil:
			// The until finally succeeded.
			return nil
		case errors.Is(conditionErr, errContextCancelled) || errors.Is(checkErr, errContextCancelled):
			// The context was cancelled.
			// Return the context cancelled error so that the Eventually will fail with a consistent error.
			return errContextCancelled
		case checkErr != nil:
			// The check failed but the until has not completed.
			// Abort the check.
			return gomega.StopTrying("Check failed before condition succeeded").Wrap(checkErr)
		default:
			return conditionErr
		}
	}).WithContext(ctx).Should(gomega.Succeed(), "check failed or condition did not succeed before the context was cancelled")
}

// runAssertion runs the assertion function and returns an error if the assertion failed.
func runAssertion(ctx context.Context, assertion func(context.Context, GomegaAssertions) bool) error {
	select {
	case <-ctx.Done():
		return errContextCancelled
	default:
	}

	var err error

	g := gomega.NewGomega(func(message string, callerSkip ...int) {
		err = errors.New(message) //nolint:goerr113
	})

	if !assertion(ctx, g) {
		return err
	}

	return nil
}

func GetControlPlaneHostAndPort(ctx context.Context, cl client.Client) (string, int32, error) {
	var infraCluster configv1.Infrastructure

	namespacedName := client.ObjectKey{
		Namespace: ClusterAPINamespace,
		Name:      "cluster",
	}

	if err := cl.Get(ctx, namespacedName, &infraCluster); err != nil {
		return "", 0, fmt.Errorf("failed to get the infrastructure object: %w", err)
	}

	apiURL, err := url.Parse(infraCluster.Status.APIServerURL)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse the API server URL: %w", err)
	}

	port, err := strconv.ParseInt(apiURL.Port(), 10, 32)
	if err != nil {
		return apiURL.Hostname(), 0, fmt.Errorf("failed to parse port: %w", err)
	}

	return apiURL.Hostname(), int32(port), nil
}
