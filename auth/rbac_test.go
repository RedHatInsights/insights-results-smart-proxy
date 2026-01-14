package auth

import (
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/stretchr/testify/assert"
)

func TestAggregatePermissions(t *testing.T) {
	type testCase struct {
		name        string
		permissions []string
		want        map[string][]string
	}

	testCases := []testCase{
		{
			name:        "no permissions",
			permissions: []string{},
			want:        map[string][]string{},
		},
		{
			name:        "all permissions",
			permissions: []string{"ocp-advisor:*:*"},
			want:        map[string][]string{"*": {}},
		},
		{
			name:        "all permissions for recommendations",
			permissions: []string{"ocp-advisor:recommendation-results:*"},
			want:        map[string][]string{"recommendation-results": {"*"}},
		},
		{
			name:        "read permissions for recommendations",
			permissions: []string{"ocp-advisor:recommendation-results:read"},
			want:        map[string][]string{"recommendation-results": {"read"}},
		},
		{
			name:        "recommendations permissions but not for ocp-advisor",
			permissions: []string{"other:recommendation-results:read"},
			want:        map[string][]string{},
		},
		{
			name:        "all permissions but not for ocp-advisor",
			permissions: []string{"other:recommendation-results:*"},
			want:        map[string][]string{},
		},
		{
			name:        "permissions on ocp-advisor but not for recommendations",
			permissions: []string{"ocp-advisor:other:*"},
			want:        map[string][]string{},
		},
		{
			name:        "all permissions for recommendations and other resources",
			permissions: []string{"other:other:*", "ocp-advisor:recommendation-results:*"},
			want:        map[string][]string{"recommendation-results": {"*"}},
		},
		{
			name:        "bad RBAC response (not enough elements)",
			permissions: []string{"ocp-advisor:recommendation-results"},
			want:        map[string][]string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			acls := []types.RbacData{}
			for _, permission := range tc.permissions {
				acls = append(acls, types.RbacData{Permission: permission})
			}
			got := aggregatePermissions(acls)
			assert.Equal(t, tc.want, got)
		})
	}
}
