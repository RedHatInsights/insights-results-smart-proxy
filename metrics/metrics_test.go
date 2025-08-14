/*
Copyright © 2020 Red Hat, Inc.

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

package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
)

func TestAddAPIMetricsWithNamespace(t *testing.T) {
	// Test that the function can be called without panicking
	testNamespace := "test_namespace"
	
	// This should not panic
	metrics.AddAPIMetricsWithNamespace(testNamespace)
	
	// Verify that metrics are accessible after registration
	assert.NotNil(t, metrics.RBACIdentityType)
	assert.NotNil(t, metrics.RBACServiceAccountRejected)
}

func TestRBACIdentityTypeMetric(t *testing.T) {
	// Test that we can increment the RBAC identity type counter without panicking
	assert.NotPanics(t, func() {
		metrics.RBACIdentityType.WithLabelValues("user").Inc()
	}, "Incrementing RBACIdentityType should not panic")
	
	assert.NotPanics(t, func() {
		metrics.RBACIdentityType.WithLabelValues("serviceaccount").Inc()
	}, "Incrementing RBACIdentityType with different label should not panic")
}

func TestRBACServiceAccountRejectedMetric(t *testing.T) {
	// Test that we can increment the service account rejected counter without panicking
	assert.NotPanics(t, func() {
		metrics.RBACServiceAccountRejected.Inc()
	}, "Incrementing RBACServiceAccountRejected should not panic")
}

func TestMetricsInitialization(t *testing.T) {
	// Test that metrics are properly initialized
	assert.NotNil(t, metrics.RBACIdentityType, "RBACIdentityType should be initialized")
	assert.NotNil(t, metrics.RBACServiceAccountRejected, "RBACServiceAccountRejected should be initialized")
}

func TestAddAPIMetricsWithNamespaceMultipleCalls(t *testing.T) {
	// Test that multiple calls don't panic
	assert.NotPanics(t, func() {
		metrics.AddAPIMetricsWithNamespace("test1")
		metrics.AddAPIMetricsWithNamespace("test2")
	}, "Multiple calls to AddAPIMetricsWithNamespace should not panic")
}