// Package metrics defines Prometheus counters used by the smart-proxy.
package metrics

import (
	"github.com/RedHatInsights/insights-operator-utils/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	rbacIdentityTypeOps = prometheus.CounterOpts{
		Name: "rbac_identity_type",
		Help: "The total number of requesters by identity type",
	}

	rbacServiceAccountRejectedOps = prometheus.CounterOpts{
		Name: "rbac_service_accounts_rejected",
		Help: "The total number of service accounts that were rejected due to ACL",
	}
	apiEndpointsRequestsWithUserAgentOps = prometheus.CounterOpts{
		Name: "api_endpoints_user_agent",
		Help: "The total number of requests per endpoint with user agent information",
	}
)

// RBACIdentityType shows number of requesters by identity type. For example
// User, ServiceAccount...
var RBACIdentityType = promauto.NewCounterVec(rbacIdentityTypeOps, []string{"type"})

// RBACServiceAccountRejected shows number of SAs that were rejected due to
// their ACL and our RBAC policies.
var RBACServiceAccountRejected = promauto.NewCounter(rbacServiceAccountRejectedOps)

// APIEndpointsRequestsWithUserAgent shows the total number of requests per endpoint with user agent
var APIEndpointsRequestsWithUserAgent = promauto.NewCounterVec(apiEndpointsRequestsWithUserAgentOps, []string{"endpoint", "user_agent"})

// AddAPIMetricsWithNamespace registers API and RBAC metrics under the
// given Prometheus namespace.
func AddAPIMetricsWithNamespace(namespace string) {
	metrics.AddAPIMetricsWithNamespace(namespace)

	rbacIdentityTypeOps.Namespace = namespace
	RBACIdentityType = promauto.NewCounterVec(rbacIdentityTypeOps, []string{"type"})

	rbacServiceAccountRejectedOps.Namespace = namespace
	RBACServiceAccountRejected = promauto.NewCounter(rbacServiceAccountRejectedOps)

	apiEndpointsRequestsWithUserAgentOps.Namespace = namespace
	APIEndpointsRequestsWithUserAgent = promauto.NewCounterVec(apiEndpointsRequestsWithUserAgentOps, []string{"endpoint", "user_agent"})
}
