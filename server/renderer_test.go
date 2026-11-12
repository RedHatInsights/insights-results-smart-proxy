// Copyright 2026 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rendererResponseItem mirrors a single item of the renderer service response
type rendererResponseItem struct {
	Rendered map[string]string `json:"rendered"`
	Errors   map[string]string `json:"errors"`
}

type rendererResponse struct {
	Items []rendererResponseItem `json:"items"`
}

// startRendererMock starts an HTTP server that acts as the template renderer
// service. The handler receives the request body and returns items computed
// by the given callback, indexed the same as the request items.
func startRendererMock(
	t *testing.T,
	respond func(items []map[string]interface{}) rendererResponse,
) *httptest.Server {
	t.Helper()

	rendererServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v2/rendered_reports", request.URL.Path)

		var requestBody struct {
			Items []struct {
				Template map[string]string      `json:"template"`
				Data     map[string]interface{} `json:"data"`
			} `json:"items"`
		}
		require.NoError(t, json.NewDecoder(request.Body).Decode(&requestBody))

		items := make([]map[string]interface{}, 0, len(requestBody.Items))
		for _, item := range requestBody.Items {
			items = append(items, map[string]interface{}{
				"template": item.Template,
				"data":     item.Data,
			})
		}

		response := respond(items)
		writer.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(writer).Encode(response))
	}))
	t.Cleanup(rendererServer.Close)

	return rendererServer
}

// rendererTestServer creates a smart proxy HTTPServer configured to use the
// given renderer service URL
func rendererTestServer(rendererURL string) *server.HTTPServer {
	servicesConfig := helpers.DefaultServicesConfig
	servicesConfig.RendererBaseEndpoint = rendererURL + "/"

	return server.New(
		helpers.DefaultServerConfig,
		servicesConfig,
		nil, // AMS client
		nil, // Redis client
		nil, // groups channel
		nil, // error found channel
		nil, // error channel
		nil, // RBAC client
	)
}

func TestRenderRuleWithContent(t *testing.T) {
	rendererServer := startRendererMock(t, func(items []map[string]interface{}) rendererResponse {
		require.Len(t, items, 1)
		return rendererResponse{
			Items: []rendererResponseItem{{
				// what the renderer returns for the templates below
				// with data {"cluster_name": "test-cluster"}
				Rendered: map[string]string{
					"reason":     "Reason test-cluster",
					"resolution": "Resolution test-cluster",
					"more_info":  "More info test-cluster",
					"details":    "Static details",
				},
				Errors: map[string]string{},
			}},
		}
	})

	testServer := rendererTestServer(rendererServer.URL)

	rules := []types.RuleWithContentResponse{
		{
			RuleID:       "test_rule_check",
			Reason:       "Reason {{=pydata.cluster_name}}",
			Resolution:   "Resolution {{=pydata.cluster_name}}",
			MoreInfo:     "More info {{=pydata.cluster_name}}",
			Generic:      "Static details",
			TemplateData: map[string]interface{}{"cluster_name": "test-cluster"},
		},
	}

	require.NoError(t, server.RenderRuleWithContent(testServer, context.Background(), rules))

	assert.Equal(t, "Reason test-cluster", rules[0].Reason)
	assert.Equal(t, "Resolution test-cluster", rules[0].Resolution)
	assert.Equal(t, "More info test-cluster", rules[0].MoreInfo)
	assert.Equal(t, "Static details", rules[0].Generic)
	require.NotNil(t, rules[0].Rendering)
	assert.Equal(t, "ok", rules[0].Rendering.Status)
	assert.Empty(t, rules[0].Rendering.Errors)
}

func TestRenderRuleWithContentRenderingFailure(t *testing.T) {
	rendererServer := startRendererMock(t, func(items []map[string]interface{}) rendererResponse {
		require.Len(t, items, 1)
		return rendererResponse{
			Items: []rendererResponseItem{{
				Rendered: map[string]string{
					"reason": "Reason test-cluster",
				},
				Errors: map[string]string{
					"resolution": "Failed to render template: TypeError: pydata.missing is undefined",
				},
			}},
		}
	})

	testServer := rendererTestServer(rendererServer.URL)

	rules := []types.RuleWithContentResponse{
		{
			RuleID:       "test_rule_check",
			Reason:       "Reason {{=pydata.cluster_name}}",
			Resolution:   "Resolution {{=pydata.missing.value()}}",
			TemplateData: map[string]interface{}{"cluster_name": "test-cluster"},
		},
	}

	require.NoError(t, server.RenderRuleWithContent(testServer, context.Background(), rules))

	// successfully rendered field is replaced
	assert.Equal(t, "Reason test-cluster", rules[0].Reason)
	// failed field keeps its template
	assert.Equal(t, "Resolution {{=pydata.missing.value()}}", rules[0].Resolution)
	require.NotNil(t, rules[0].Rendering)
	assert.Equal(t, "failed", rules[0].Rendering.Status)
	assert.Contains(t, rules[0].Rendering.Errors["resolution"], "pydata.missing")
}

func TestRenderRuleWithContentEmpty(t *testing.T) {
	// no rules means no call to the renderer service
	testServer := rendererTestServer("http://localhost:0/")

	require.NoError(t, server.RenderRuleWithContent(testServer, context.Background(), nil))
}

func TestRenderDVORecommendations(t *testing.T) {
	rendererServer := startRendererMock(t, func(items []map[string]interface{}) rendererResponse {
		require.Len(t, items, 1)
		return rendererResponse{
			Items: []rendererResponseItem{{
				// what the renderer returns for the templates below
				// with data {"name": "test-workload"}
				Rendered: map[string]string{
					"reason":     "Reason test-workload",
					"resolution": "Resolution test-workload",
					"details":    "Details test-workload",
				},
				Errors: map[string]string{
					"more_info": "Failed to render template: TypeError: pydata.missing is undefined",
				},
			}},
		}
	})

	testServer := rendererTestServer(rendererServer.URL)

	recommendations := []types.DVORecommendation{
		{
			Check:        "test_rule_check",
			Reason:       "Reason {{=pydata.name}}",
			Resolution:   "Resolution {{=pydata.name}}",
			MoreInfo:     "More info {{=pydata.missing.value()}}",
			Details:      "Details {{=pydata.name}}",
			TemplateData: map[string]interface{}{"name": "test-workload"},
		},
	}

	require.NoError(t, server.RenderDVORecommendations(testServer, context.Background(), recommendations))

	assert.Equal(t, "Reason test-workload", recommendations[0].Reason)
	assert.Equal(t, "Resolution test-workload", recommendations[0].Resolution)
	assert.Equal(t, "Details test-workload", recommendations[0].Details)
	assert.Equal(t, "More info {{=pydata.missing.value()}}", recommendations[0].MoreInfo)
	require.NotNil(t, recommendations[0].Rendering)
	assert.Equal(t, "failed", recommendations[0].Rendering.Status)
	assert.Contains(t, recommendations[0].Rendering.Errors["more_info"], "pydata.missing")
}

func TestRenderDVORecommendationsWithoutTemplateData(t *testing.T) {
	// recommendations without extra_data must not break the whole request:
	// the renderer requires an object, so nil template data is normalized
	var requestedData []map[string]interface{}

	rendererServer := startRendererMock(t, func(items []map[string]interface{}) rendererResponse {
		requestedData = items
		return rendererResponse{
			Items: []rendererResponseItem{{
				// with empty template data, pydata.name is undefined
				Rendered: map[string]string{},
				Errors: map[string]string{
					"reason": "Failed to render template: TypeError: pydata.name is undefined",
				},
			}},
		}
	})

	testServer := rendererTestServer(rendererServer.URL)

	recommendations := []types.DVORecommendation{
		{
			Check:  "test_rule_check",
			Reason: "Reason {{=pydata.name}}",
			// TemplateData left nil on purpose
		},
	}

	require.NoError(t, server.RenderDVORecommendations(testServer, context.Background(), recommendations))

	require.Len(t, requestedData, 1)
	assert.NotNil(t, requestedData[0]["data"], "nil template data must be sent as an object, not null")

	// the renderer cannot resolve pydata.name with empty template data
	assert.Equal(t, "Reason {{=pydata.name}}", recommendations[0].Reason)
	require.NotNil(t, recommendations[0].Rendering)
	assert.Equal(t, "failed", recommendations[0].Rendering.Status)
	assert.Contains(t, recommendations[0].Rendering.Errors["reason"], "pydata.name")
}

func TestCallRendererServiceUnavailable(t *testing.T) {
	rendererServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(rendererServer.Close)

	testServer := rendererTestServer(rendererServer.URL)

	rules := []types.RuleWithContentResponse{
		{RuleID: "test_rule_check", Reason: "template"},
	}

	err := server.RenderRuleWithContent(testServer, context.Background(), rules)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestCallRendererIncompleteItem(t *testing.T) {
	// a nonempty field missing from both rendered and errors must be rejected,
	// otherwise the item would be reported as successfully rendered while
	// keeping its unresolved template
	rendererServer := startRendererMock(t, func(items []map[string]interface{}) rendererResponse {
		require.Len(t, items, 1)
		return rendererResponse{
			Items: []rendererResponseItem{{
				Rendered: map[string]string{
					"reason": "Rendered reason",
				},
				Errors: map[string]string{},
			}},
		}
	})

	testServer := rendererTestServer(rendererServer.URL)

	rules := []types.RuleWithContentResponse{
		{
			RuleID:       "test_rule_check",
			Reason:       "Reason {{=pydata.cluster_name}}",
			Resolution:   "Resolution {{=pydata.cluster_name}}",
			TemplateData: map[string]interface{}{"cluster_name": "test-cluster"},
		},
	}

	err := server.RenderRuleWithContent(testServer, context.Background(), rules)
	require.Error(t, err)
	var unavailable *server.RendererServiceUnavailableError
	require.ErrorAs(t, err, &unavailable)
}

func TestCallRendererItemCountMismatch(t *testing.T) {
	rendererServer := startRendererMock(t, func(_ []map[string]interface{}) rendererResponse {
		return rendererResponse{Items: nil}
	})

	testServer := rendererTestServer(rendererServer.URL)

	rules := []types.RuleWithContentResponse{
		{RuleID: "test_rule_check", Reason: "template"},
	}

	err := server.RenderRuleWithContent(testServer, context.Background(), rules)
	require.Error(t, err)
}

func TestReadRenderParam(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/cluster/123/reports?render=true", nil)
	render, err := server.ReadRenderParam(request)
	require.NoError(t, err)
	assert.True(t, render)

	request = httptest.NewRequest(http.MethodGet, "/cluster/123/reports", nil)
	render, err = server.ReadRenderParam(request)
	require.NoError(t, err)
	assert.False(t, render)

	request = httptest.NewRequest(http.MethodGet, "/cluster/123/reports?render=notabool", nil)
	_, err = server.ReadRenderParam(request)
	require.Error(t, err)
	// unparsable values must be reported as a bad request, not an internal error
	var parsingError *server.RouterParsingError
	require.ErrorAs(t, err, &parsingError)
	assert.Equal(t, server.RenderParam, parsingError.ParamName)
}
