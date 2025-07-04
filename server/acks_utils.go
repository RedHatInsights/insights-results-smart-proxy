/*
Copyright © 2021, 2022 Red Hat, Inc.

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

package server

// Helper functions to be called from request handlers defined in the source
// file acks_handlers.go.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	types "github.com/RedHatInsights/insights-results-types"
)

const aggregatorImproperCodeMessage = "aggregator responded with improper HTTP code: %v"

// readJustificationFromBody function tries to read data
// structure types.AcknowledgemenJustification from response
// payload (body)
func readJustificationFromBody(request *http.Request) (
	types.AcknowledgementJustification, error,
) {
	// try to read request body
	var parameters types.AcknowledgementJustification
	err := json.NewDecoder(request.Body).Decode(&parameters)

	// JSON Decode() will not throw an error when a field isn't present. This is NOT strict decoding.
	if err != nil {
		log.Error().Err(err).Msg("wrong payload (not justification) provided by client")
		err := &RouterMissingParamError{ParamName: "justification"}
		return parameters, err
	}

	// everything seems to be ok
	return parameters, nil
}

// readRuleSelectorAndJustificationFromBody function tries to read data
// structure types.AcknowledgementRuleSelectorJustification from response
// payload (body)
func readRuleSelectorAndJustificationFromBody(writer http.ResponseWriter, request *http.Request) (
	types.AcknowledgementRuleSelectorJustification, error,
) {
	// try to read request body
	var parameters types.AcknowledgementRuleSelectorJustification
	err := json.NewDecoder(request.Body).Decode(&parameters)

	if err != nil {
		log.Warn().Err(err).Msg("wrong payload provided by client")
		// return HTTP code 400 to client
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return parameters, err
	}

	// everything seems to be ok
	return parameters, nil
}

// returnRuleAckToClient returns information about selected rule ack to client.
// This function also tries to process all errors.
func returnRuleAckToClient(writer http.ResponseWriter, ack types.Acknowledgement) {
	// serialize the above data structure into JSON format
	serializedAck, err := json.MarshalIndent(ack, "", "\t")
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
		return
	}

	// and send the serialized structure to client
	_, err = writer.Write(serializedAck)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// ackRuleSystemWide method acknowledges rule via Insights Aggregator REST API
func (server *HTTPServer) ackRuleSystemWide(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID, justification string,
) error {
	var j types.AcknowledgementJustification
	j.Value = justification

	// try to ack rule via Insights Aggregator REST API
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.DisableRuleSystemWide,
		ruleID, errorKey, orgID,
	)

	// generate payload in JSON format
	jsonReq, err := json.Marshal(j)
	if err != nil {
		return err
	}

	// call PUT method, provide the required data in payload
	req, err := http.NewRequest(http.MethodPut, aggregatorURL, bytes.NewBuffer(jsonReq))
	if err != nil {
		return err
	}

	req.Header.Set(contentTypeHeader, JSONContentType)
	client := &http.Client{}
	response, err := client.Do(req) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	if err != nil {
		return err
	}

	defer services.CloseResponseBody(response)

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(aggregatorImproperCodeMessage, response.StatusCode)
		return err
	}

	return nil
}

// updateAckRuleSystemWide method updates rule ACK via Insights Aggregator REST
// API
func (server *HTTPServer) updateAckRuleSystemWide(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID, justification string,
) error {
	var j types.AcknowledgementJustification
	j.Value = justification

	// try to ack rule via Insights Aggregator REST API
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.UpdateRuleSystemWide,
		ruleID, errorKey, orgID,
	)

	// marshal data to be POSTed to Insights Aggregator
	jsonData, err := json.Marshal(j)
	if err != nil {
		return err
	}

	// do POST request and read response from Insights Aggregator
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	response, err := http.Post(aggregatorURL, JSONContentType,
		bytes.NewBuffer(jsonData)) // #nosec G107
	if err != nil {
		return err
	}

	defer services.CloseResponseBody(response)

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(aggregatorImproperCodeMessage,
			response.StatusCode)
		return err
	}

	return nil
}

// deleteAckRuleSystemWide method deletes the acknowledgement of a rule via
// Insights Aggregator REST API
func (server *HTTPServer) deleteAckRuleSystemWide(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID,
) error {
	// try to ack rule via Insights Aggregator REST API
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.EnableRuleSystemWide,
		ruleID, errorKey, orgID,
	)

	// call PUT method
	req, err := http.NewRequest(http.MethodPut, aggregatorURL, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set(contentTypeHeader, JSONContentType)
	client := &http.Client{}
	response, err := client.Do(req) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	if err != nil {
		return err
	}

	defer services.CloseResponseBody(response)

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(aggregatorImproperCodeMessage, response.StatusCode)
		return err
	}

	return nil
}

// Method readListOfAckedRules reads all rules that has been acked system-wide
func (server *HTTPServer) readListOfAckedRules(
	orgID types.OrgID,
) ([]types.SystemWideRuleDisable, error) {
	// wont be used anywhere else
	type responsePayload struct {
		Status      string                        `json:"status"`
		RuleDisable []types.SystemWideRuleDisable `json:"disabledRules"`
	}

	// try to read rule list from Insights Aggregator
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ListOfDisabledRulesSystemWide,
		orgID,
	)

	// #nosec G107
	response, err := http.Get(aggregatorURL) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	if err != nil {
		return nil, err
	}

	defer services.CloseResponseBody(response)

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected HTTP code during reading list of rules: %v", response.StatusCode)
		return nil, err
	}

	var payload responsePayload

	// decode the response payload
	err = json.NewDecoder(response.Body).Decode(&payload)
	if err != nil {
		err = errors.New("problem unmarshalling JSON response from aggregator endpoint")
		return nil, err
	}

	log.Debug().Int("#rules", len(payload.RuleDisable)).Msg("Read disabled rules")
	return payload.RuleDisable, nil
}

// readRuleDisableStatus method read system-wide rule disable status from
// Insights Results Aggregator via REST API
func (server *HTTPServer) readRuleDisableStatus(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID,
) (types.Acknowledgement, bool, error) {
	// wont be used anywhere else
	type responsePayload struct {
		Status      string                      `json:"status"`
		RuleDisable types.SystemWideRuleDisable `json:"disabledRule"`
	}

	var acknowledgement types.Acknowledgement

	// try to read rule disable status from aggregator
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReadRuleSystemWide,
		ruleID, errorKey, orgID,
	)

	// #nosec G107
	response, err := http.Get(aggregatorURL) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	if err != nil {
		return acknowledgement, false, err
	}

	defer services.CloseResponseBody(response)

	// check the aggregator response
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		err := fmt.Errorf("aggregator responded with improper HTTP code: %v", response.StatusCode)
		return acknowledgement, false, err
	}

	var payload responsePayload

	err = json.NewDecoder(response.Body).Decode(&payload)
	if err != nil {
		return acknowledgement, false, err
	}

	acknowledgement.Rule = string(payload.RuleDisable.RuleID) + "|" + string(payload.RuleDisable.ErrorKey)
	acknowledgement.Justification = payload.RuleDisable.Justification
	acknowledgement.CreatedBy = string(payload.RuleDisable.UserID)
	acknowledgement.CreatedAt = formatNullTime(payload.RuleDisable.CreatedAt)
	acknowledgement.UpdatedAt = formatNullTime(payload.RuleDisable.UpdatedAT)

	acknowledgementFound := response.StatusCode == http.StatusOK
	return acknowledgement, acknowledgementFound, nil
}

// format sql.NullTime value accordingly
// TODO: move to utils repository as usual
func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func logFullRuleSelector(orgID types.OrgID, ruleID types.RuleID, errorKey types.ErrorKey) {
	log.Debug().
		Int("org", int(orgID)).
		Str(ruleIDStr, string(ruleID)).
		Str(errorKeyStr, string(errorKey)).
		Msg("Selector for rule acknowledgement")
}

// prepareAckList converts data to format accepted by Insights Advisor
func prepareAckList(acks []types.SystemWideRuleDisable) types.AcknowledgementsResponse {
	var responseBody types.AcknowledgementsResponse

	// fill-in metadata part of response body
	responseBody.Metadata.Count = len(acks)

	// fill-in data part of response body
	responseBody.Data = make([]types.Acknowledgement, len(acks))

	// perform conversion item-by-item
	i := 0
	for _, ack := range acks {
		var acknowledgement types.Acknowledgement
		acknowledgement.Rule = string(ack.RuleID) + "|" + string(ack.ErrorKey)
		acknowledgement.Justification = ack.Justification
		acknowledgement.CreatedBy = string(ack.UserID)
		acknowledgement.CreatedAt = formatNullTime(ack.CreatedAt)
		acknowledgement.UpdatedAt = formatNullTime(ack.UpdatedAT)
		responseBody.Data[i] = acknowledgement
		i++
	}

	return responseBody
}
