/*
Copyright © 2021 Red Hat, Inc.

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
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/types"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
)

const aggregatorImproperCodeMessage = "Aggregator responded with improper HTTP code: %v"

// readRuleSelectorAndJustificationFromBody function tries to read data
// structure types.AcknowledgementRuleSelectorJustification from response
// payload (body)
func readRuleSelectorAndJustificationFromBody(writer http.ResponseWriter, request *http.Request) (
	types.AcknowledgementRuleSelectorJustification, error) {

	// try to read request body
	var parameters types.AcknowledgementRuleSelectorJustification
	err := json.NewDecoder(request.Body).Decode(&parameters)

	if err != nil {
		log.Error().Err(err).Msg("wrong payload provided by client")
		// return HTTP code 400 to client
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return parameters, err
	}

	// everything seems to be ok
	return parameters, nil
}

// readOrgIDAndUserIDFromToken helper method reads organization ID and user ID
// (account number) from the token
func (server *HTTPServer) readOrgIDAndUserIDFromToken(writer http.ResponseWriter, request *http.Request) (
	types.OrgID, types.UserID, error) {
	// auth. token contains organization ID and user ID we need to use
	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return types.OrgID(0), types.UserID(""), err
	}

	// Organization ID and user ID are to be provided in the token
	orgID := authToken.Internal.OrgID
	userID := authToken.AccountNumber
	return orgID, userID, nil
}

// returnRuleAckToClient returns information about selected rule ack to client.
// This function also tries to process all errors.
func returnRuleAckToClient(writer http.ResponseWriter, ack types.Acknowledgement) {
	// serialize the above data structure into JSON format
	bytes, err := json.MarshalIndent(ack, "", "\t")
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
		return
	}

	// and send the serialized structure to client
	_, err = writer.Write(bytes)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// ackRuleSystemWide method acknowledges rule via Insights Aggregator REST API
func (server *HTTPServer) ackRuleSystemWide(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID, userID types.UserID, justification string) error {
	var j types.AcknowledgementJustification
	j.Value = justification

	// try to ack rule via Insights Aggregator REST API
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.DisableRuleSystemWide,
		ruleID, errorKey, orgID, userID,
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

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(aggregatorImproperCodeMessage, response.StatusCode)
		return err
	}

	return nil
}

// deleteAckRuleSystemWide method deletes the acknowledgement of a rule via
// Insights Aggregator REST API
func (server *HTTPServer) deleteAckRuleSystemWide(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID, userID types.UserID) error {

	// try to ack rule via Insights Aggregator REST API
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.EnableRuleSystemWide,
		ruleID, errorKey, orgID, userID,
	)

	// call PUT method
	req, err := http.NewRequest(http.MethodPut, aggregatorURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	// check the aggregator response
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(aggregatorImproperCodeMessage, response.StatusCode)
		return err
	}

	return nil
}

// readRuleDisableStatus method read system-wide rule disable status from
// Insights Results Aggregator via REST API
func (server *HTTPServer) readRuleDisableStatus(
	ruleID types.Component, errorKey types.ErrorKey,
	orgID types.OrgID, userID types.UserID) (types.Acknowledgement, bool, error) {

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
		ruleID, errorKey, orgID, userID,
	)

	// #nosec G107
	response, err := http.Get(aggregatorURL)
	if err != nil {
		return acknowledgement, false, err
	}

	// check the aggregator response
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		err := fmt.Errorf("Aggregator responded with improper HTTP code: %v", response.StatusCode)
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
