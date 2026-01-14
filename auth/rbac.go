/*
Copyright © 2024 Red Hat, Inc.

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

package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/rs/zerolog/log"
)

// RBACClient defines the interface for an RBAC client.
type RBACClient interface {
	IsAuthorized(token string) bool
	IsEnforcing() bool
}

type rbacClientImpl struct {
	uri         string
	host        string
	client      *http.Client
	enforceAuth bool
}

// NewRBACClient create an RBACClient from the configuration
func NewRBACClient(conf *RBACConfig) (RBACClient, error) {
	url, host, err := constructRBACURL(conf)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	return &rbacClientImpl{
		url,
		host,
		client,
		conf.EnforceAuth,
	}, nil
}

// IsEnforcing returns wether requests should be denied if the requester does not
// have the correct permissions.
func (rc *rbacClientImpl) IsEnforcing() bool {
	return rc.enforceAuth
}

// IsAuthorized checks if an account has the correct permissions to access our resources
func (rc *rbacClientImpl) IsAuthorized(token string) bool {
	permissions := rc.getPermissions(token)
	log.Debug().Interface("permissions", permissions).Msg("Account ocp-advisor permissions")
	return permissions != nil
}

func (rc *rbacClientImpl) getPermissions(identityToken string) map[string][]string {
	acls := rc.requestAccess(rc.uri, identityToken)
	if len(acls) > 0 {
		log.Debug().Interface("acls", acls).Msg("Account all permissions")
		permissions := aggregatePermissions(acls)
		if len(permissions) > 0 {
			// as we just need a read permission, we accept any > 0 here
			log.Info().Any("RBAC openshift permissions", permissions).Send()
			return permissions
		}
		return nil
	}
	log.Debug().Msg("Account has no ACLs")
	return nil
}

// requestAccess handles the call(s) to RBAC taking into account that the response
// is paginated
func (rc *rbacClientImpl) requestAccess(url, identityToken string) []types.RbacData {
	//TODO Change return to (rbacData, err) and forward error?
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RBAC request")
		return nil
	}

	// Forward the x-rh-identity header directly
	req.Header.Set(XRHAuthTokenHeader, identityToken)

	resp, err := rc.client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to call RBAC API")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("RBAC API returned non-200 status: %d", resp.StatusCode)
		return nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing response body")
		}
	}()

	body, _ := io.ReadAll(resp.Body)
	response := types.RbacResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		log.Err(err).Str("URL", url).Msg("Unable to unmarshal response from RBAC server")
	}

	if response.Meta.Count == 0 {
		//TODO: Debug level, not info, but for now we need it
		log.Info().Msg("No RBAC data for this user")
		return nil
	}
	access := []types.RbacData{}

	access = append(access, response.Data...)
	if response.Links.Next != "" {
		nextURL := fmt.Sprintf("%s%s", rc.host, response.Links.Next)
		access = append(access, rc.requestAccess(nextURL, identityToken)...)
	}
	return access
}

// aggregatePermissions loop over all the permissions/roles/alcs of the user returned
// from RBAC and creates and return the map of permissions where key is
// resourceType (recommendation-results) and the values are the
// slice of resources (cluster names, node names, project names).
// We are interested in this ACLs: https://github.com/RedHatInsights/rbac-config/blob/master/configs/prod/permissions/ocp-advisor.json
func aggregatePermissions(acls []types.RbacData) map[string][]string {
	permissions := map[string][]string{}
	for _, acl := range acls {
		splits := strings.Split(acl.Permission, ":")
		if len(splits) < 3 {
			log.Warn().Str("ACL", acl.Permission).Msg("Unexpected RBAC response")
			continue
		}
		if splits[0] != "ocp-advisor" {
			// check the ACL is for ocp-advisor, not for other APIs
			continue
		}
		resourceType := splits[1]
		verb := splits[2]
		// ignore other kind of permissions, we just want recommendation-results
		if strings.Contains(resourceType, "recommendation-results") {
			if _, ok := permissions[resourceType]; !ok {
				// add the resource type to the permissions map if not already
				// there so that we can then add the permission verb
				permissions[resourceType] = []string{}
			}
			permissions[resourceType] = append(permissions[resourceType], verb)
		} else if resourceType == "*" {
			permissions["*"] = []string{}
		}
	}
	return permissions
}
