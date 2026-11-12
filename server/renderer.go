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

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"

	"github.com/rs/zerolog/log"
)

const (
	// rendererRenderEndpoint is the endpoint of the insights-content-template-renderer
	// service used for batch rendering of DoT.js templates
	rendererRenderEndpoint = "v2/rendered_reports"

	// rendererRequestTimeout bounds the entire batch request to the renderer.
	// Batches exceeding this limit fail as a whole rather than returning per-field errors.
	rendererRequestTimeout = 30 * time.Second
)

// rendererTemplateFields are the recommendation fields that contain DoT.js
// templates and are rendered when the render query parameter is used
var rendererTemplateFields = []string{"reason", "resolution", "more_info", "details"}

// rendererRenderItem is a single item of the renderer service request
type rendererRenderItem struct {
	// Template maps a recommendation field name to its DoT.js template
	Template map[string]string `json:"template"`
	// Data is the per-hit template data (pydata) used to render the templates
	Data map[string]interface{} `json:"data"`
}

// rendererRenderRequest is the request body for the renderer service
type rendererRenderRequest struct {
	Items []rendererRenderItem `json:"items"`
}

// rendererRenderedItem is a single item of the renderer service response
type rendererRenderedItem struct {
	// Rendered maps a recommendation field name to its rendered text
	Rendered map[string]string `json:"rendered"`
	// Errors maps a recommendation field name to the rendering error message
	Errors map[string]string `json:"errors"`
}

// rendererRenderResponse is the response body from the renderer service
type rendererRenderResponse struct {
	Items []rendererRenderedItem `json:"items"`
}

// callRenderer sends the given items to the renderer service and returns the
// rendered items in the same order. The HTTP client applies a fixed timeout;
// the context also allows cancellation when the inbound request ends.
func (server *HTTPServer) callRenderer(
	ctx context.Context,
	items []rendererRenderItem,
) ([]rendererRenderedItem, error) {
	rendererURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.RendererBaseEndpoint,
		rendererRenderEndpoint,
	)

	httpClient := http.Client{Timeout: rendererRequestTimeout}

	payload, err := json.Marshal(rendererRenderRequest{Items: items})
	if err != nil {
		return nil, err
	}

	rendererRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, rendererURL, bytes.NewReader(payload))
	if err != nil {
		log.Error().Err(err).Str("url", rendererURL).Msg("unable to create the renderer request")
		return nil, err
	}
	rendererRequest.Header.Set(contentTypeHeader, JSONContentType)

	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	response, err := httpClient.Do(rendererRequest)
	if err != nil {
		log.Error().Err(err).Str("url", rendererURL).Msg("error reaching the template renderer service")
		return nil, &RendererServiceUnavailableError{}
	}
	defer services.CloseResponseBody(response)

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Str("url", rendererURL).Msg("unable to read the body of the renderer response")
		return nil, &RendererServiceUnavailableError{}
	}

	if response.StatusCode != http.StatusOK {
		log.Error().Int("status", response.StatusCode).Str("url", rendererURL).
			Msg("template renderer service returned an error")
		return nil, &RendererServiceUnavailableError{}
	}

	rendererResponse := rendererRenderResponse{}
	if err = json.Unmarshal(responseBytes, &rendererResponse); err != nil {
		log.Error().Err(err).Str("url", rendererURL).Msg("error unmarshalling renderer response")
		return nil, &RendererServiceUnavailableError{}
	}

	if len(rendererResponse.Items) != len(items) {
		log.Error().
			Int("requested", len(items)).
			Int("received", len(rendererResponse.Items)).
			Msg("template renderer returned a different number of items than requested")
		return nil, &RendererServiceUnavailableError{}
	}

	// every requested nonempty field must be either rendered or reported as
	// failed; the renderer skips empty fields, so those may be omitted
	for i, requested := range items {
		renderedItem := rendererResponse.Items[i]
		for field, template := range requested.Template {
			if template == "" {
				continue
			}
			_, isRendered := renderedItem.Rendered[field]
			_, isError := renderedItem.Errors[field]
			if !isRendered && !isError {
				log.Error().
					Int("item", i).
					Str("field", field).
					Msg("template renderer response is missing the result of a requested field")
				return nil, &RendererServiceUnavailableError{}
			}
		}
	}

	return rendererResponse.Items, nil
}

// renderReport renders the DoT.js templates of the given rules with
// the per-hit cluster data (extra_data). Successfully rendered fields are
// replaced with their rendered text; fields that fail to render keep their
// templates and the error is reported in the recommendation's rendering
// status.
func (server *HTTPServer) renderReport(ctx context.Context, rules []types.RuleWithContentResponse) error {
	if len(rules) == 0 {
		return nil
	}

	items := make([]rendererRenderItem, 0, len(rules))

	for i := range rules {
		rule := &rules[i]
		templates := map[string]string{
			"reason":     rule.Reason,
			"resolution": rule.Resolution,
			"more_info":  rule.MoreInfo,
			"details":    rule.Generic,
		}
		items = append(items, rendererRenderItem{
			Template: templates,
			Data:     toStringMap(rule.TemplateData),
		})
	}

	renderedItems, err := server.callRenderer(ctx, items)
	if err != nil {
		return err
	}

	for i := range rules {
		applyRenderingStatus(&rules[i].Reason, &rules[i].Resolution, &rules[i].MoreInfo, &rules[i].Generic,
			renderedItems[i], &rules[i].Rendering)
	}

	return nil
}

// applyRenderingStatus replaces the given recommendation fields with their
// rendered texts and fills in the rendering status. Field order in field
// pointers must match rendererTemplateFields order.
func applyRenderingStatus(
	reason, resolution, moreInfo, details *string,
	renderedItem rendererRenderedItem,
	rendering **types.RenderingStatus,
) {
	fields := []*string{reason, resolution, moreInfo, details}
	rendered := renderedItem.Rendered
	errors := renderedItem.Errors

	for i, field := range rendererTemplateFields {
		fieldPtr := fields[i]
		if text, found := rendered[field]; found {
			*fieldPtr = text
		}
	}

	if len(errors) == 0 {
		*rendering = &types.RenderingStatus{Status: "ok"}
	} else {
		*rendering = &types.RenderingStatus{Status: "failed", Errors: errors}
	}
}

// toStringMap converts the given interface{} into map[string]interface{},
// returning an empty map for nil or incompatible values
func toStringMap(value interface{}) map[string]interface{} {
	if typed, ok := value.(map[string]interface{}); ok && typed != nil {
		return typed
	}
	return map[string]interface{}{}
}
