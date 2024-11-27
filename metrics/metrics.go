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
)

// RBACIdentityType shows number of requesters by identity type. For example
// User, ServiceAccount...
var RBACIdentityType = promauto.NewCounterVec(rbacIdentityTypeOps, []string{"type"})

// RBACServiceAccountRejected shows number of SAs that were rejected due to
// their ACL and our RBAC policies.
var RBACServiceAccountRejected = promauto.NewCounter(rbacServiceAccountRejectedOps)

func AddAPIMetricsWithNamespace(namespace string) {
	metrics.AddAPIMetricsWithNamespace(namespace)

	rbacIdentityTypeOps.Namespace = namespace
	RBACIdentityType = promauto.NewCounterVec(rbacIdentityTypeOps, []string{"type"})

	rbacServiceAccountRejectedOps.Namespace = namespace
	RBACServiceAccountRejected = promauto.NewCounter(rbacServiceAccountRejectedOps)
}
