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

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/parsers"
	types "github.com/RedHatInsights/insights-results-types"
)

// HTTP response-related constants
const (
	authTokenFormatError       = "unable to read orgID and userID from auth. token!"
	improperRuleSelectorFormat = "improper rule selector format"
	readRuleStatusError        = "read rule status error"
	readRuleJustificationError = "can not retrieve rule disable justification from Aggregator"
	aggregatorResponseError    = "problem retrieving response from aggregator endpoint"
)

// method readAckList list acks from this account where the rule is active.
// Will return an empty list if this account has no acks.
//
// Response format should look like:
//
//	{
//	  "meta": {
//	    "count": 0
//	  },
//	  "data": [
//	    {
//	      "rule": "string",
//	      "justification": "string",
//	      "created_by": "string",
//	      "created_at": "2021-09-04T17:11:35.130Z",
//	      "updated_at": "2021-09-04T17:11:35.130Z"
//	    }
//	  ]
//	}
func (server *HTTPServer) readAckList(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	acks, err := server.readListOfAckedRules(orgID)
	if err != nil {
		log.Error().Err(err).Msg(ackedRulesError)
		handleServerError(writer, err)
		return
	}

	responseBody := prepareAckList(acks)

	// serialize the above data structure into JSON format
	bytes, err := json.MarshalIndent(responseBody, "", "\t")
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// and send the serialized structure to client
	_, err = writer.Write(bytes)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// return 200 OK (default)
}

// method getAcknowledge retrieves the info about rule acknowledgement made
// from this account. Acks are created, deleted, and queired by Insights rule
// ID, not by their own ack ID.
//
// An example response:
//
//	{
//	  "rule": "string",
//	  "justification": "string",  <- can not be set by this call!!!
//	  "created_by": "string",
//	  "created_at": "2021-09-04T17:52:48.976Z",
//	  "updated_at": "2021-09-04T17:52:48.976Z"
//	}
func (server *HTTPServer) getAcknowledge(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set(contentTypeHeader, JSONContentType)

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		log.Warn().Err(err).Msg(improperRuleSelectorFormat)
		// server error has been handled already
		return
	}

	// we seem to have all data -> let's display them
	logFullRuleSelector(orgID, ruleID, errorKey)

	// test if the rule has been acknowledged already
	ruleAck, found, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		err = errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// rule was not acked -> nothing to return
	if !found {
		writer.WriteHeader(http.StatusNotFound)
		log.Debug().Msg("Rule has not been disabled previously -> nothing to return!")
		return
	}

	// we have the metadata about rule, let's send it into client in
	// response payload
	returnRuleAckToClient(writer, ruleAck)
}

// method acknowledgePost acknowledges (and therefore hides) a rule from view
// in an account. If there's already an acknowledgement of this rule by this
// account, then return that. Otherwise, a new ack is created.
//
// An example request:
//
//	{
//	  "rule_id": "string",
//	  "justification": "string"
//	}
//
// An example response:
//
//	{
//	  "rule": "string",
//	  "justification": "string",  <- can not be set by this call!!!
//	  "created_by": "string",
//	  "created_at": "2021-09-04T17:52:48.976Z",
//	  "updated_at": "2021-09-04T17:52:48.976Z"
//	}
//
// HTTP/1.1 200 OK is returned if rule has been already acked
// HTTP/1.1 201 Created is returned if rule has been acked by this call
func (server *HTTPServer) acknowledgePost(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set(contentTypeHeader, JSONContentType)

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	parameters, err := readRuleSelectorAndJustificationFromBody(writer, request)
	if err != nil {
		// everything's handled already
		return
	}

	// we seem to have all data -> let's display them
	log.Debug().
		Int("org", int(orgID)).
		Str("rule", string(parameters.RuleSelector)).
		Str("value", parameters.Value).
		Msg("Proper payload provided")

	// check if rule selector has the proper format
	ruleID, errorKey, err := parsers.ParseRuleSelector(parameters.RuleSelector)
	if err != nil {
		log.Warn().Err(err).Msg(improperRuleSelectorFormat)
		// return HTTP code 400 to client
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// display parsed rule ID and error key
	log.Debug().
		Str(ruleIDStr, string(ruleID)).
		Str(errorKeyStr, string(errorKey)).
		Msg("Parsed rule selector")

	// test if the rule has been acknowledged already
	_, previouslyAcked, err := server.readRuleDisableStatus(ruleID, errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		err = errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// if acknowledgement has been found -> return 200 OK with the existing rule ack
	// if acknowledgement has NOT been found -> return 201 Created with the created rule ack
	if previouslyAcked {
		log.Debug().Msg("Rule has been already disabled")
	} else {
		log.Debug().Msg("Rule has not been disabled previously")

		// acknowledge rule
		err := server.ackRuleSystemWide(ruleID, errorKey, orgID, parameters.Value)
		if err != nil {
			log.Error().Err(err).Msg(readRuleJustificationError)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Aggregator REST API is source of truth - let's re-read rule status
	// from it
	updatedAcknowledgement, _, err := server.readRuleDisableStatus(ruleID, errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleJustificationError)
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	if !previouslyAcked {
		// client is expecting 201 CREATED to indicate new entry
		writer.WriteHeader(http.StatusCreated)
	}

	// we have the metadata about rule, let's send it into client in
	// response payload
	returnRuleAckToClient(writer, updatedAcknowledgement)
}

// method updateAcknowledge updates an acknowledgement for a rule, by rule ID.
// A new justification can be supplied. The username is taken from the
// authenticated request. The updated ack is returned.
//
// An example of request:
//
//	{
//	   "justification": "string"
//	}
//
// An example response:
//
//	{
//	  "rule": "string",
//	  "justification": "string",
//	  "created_by": "string",
//	  "created_at": "2021-09-04T17:52:48.976Z",
//	  "updated_at": "2021-09-04T17:52:48.976Z"
//	}
//
// Additionally, if rule is not found, 404 is returned (not mentioned in
// original REST API specification).
func (server *HTTPServer) updateAcknowledge(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		log.Warn().Err(err).Msg(improperRuleSelectorFormat)
		// server error has been handled already
		return
	}

	parameters, err := readJustificationFromBody(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// we seem to have all data -> let's display them
	logFullRuleSelector(orgID, ruleID, errorKey)
	log.Debug().
		Str("justification", parameters.Value).
		Msg("Justification to be set")

	// test if the rule has been acknowledged already
	_, found, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// if acknowledgement has NOT been found -> return 404 NotFound
	if !found {
		log.Debug().Msg("Rule ack can not be found")
		err := &utypes.ItemNotFoundError{ItemID: (ruleID + "|" + types.RuleID(errorKey))}
		handleServerError(writer, err)
		return
	}

	// ok, rule has been found, so update it
	err = server.updateAckRuleSystemWide(types.Component(ruleID), errorKey, orgID, parameters.Value)
	if err != nil {
		log.Error().Err(err).Msg("Unable to update justification for rule acknowledgement")
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// Aggregator REST API is source of truth - let's re-read rule status
	// from it
	updatedAcknowledgement, _, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleJustificationError)
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// we have the metadata about rule, let's send it into client in
	// response payload
	returnRuleAckToClient(writer, updatedAcknowledgement)
}

// method deleteAcknowledge deletes an acknowledgement for a rule, by its rule
// ID. If the ack existed, it is deleted and a 204 is returned. Otherwise, a
// 404 is returned.
func (server *HTTPServer) deleteAcknowledge(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		log.Warn().Err(err).Msg(improperRuleSelectorFormat)
		// server error has been handled already
		return
	}

	// we seem to have all data -> let's display them
	logFullRuleSelector(orgID, ruleID, errorKey)

	// test if the rule has been acknowledged already
	_, found, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	if !found {
		writer.WriteHeader(http.StatusNotFound)
		log.Debug().Msg("Rule has not been disabled previously -> ACK won't be deleted")
		return
	}

	// rule has been found -> let's delete the ACK
	// delete acknowledgement for a rule
	log.Debug().Msg("About to delete ACK for a rule")
	err = server.deleteAckRuleSystemWide(types.Component(ruleID), errorKey, orgID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to delete rule acknowledgement")
		err := errors.New(aggregatorResponseError)
		handleServerError(writer, err)
		return
	}

	// return 204 -> rule ack has been deleted
	writer.WriteHeader(http.StatusNoContent)
}

func generateRuleAckMap(acks []types.SystemWideRuleDisable) (ruleAcksMap map[types.RuleID]bool) {
	ruleAcksMap = make(map[types.RuleID]bool)
	for i := range acks {
		ack := &acks[i]
		compositeRuleID, err := generators.GenerateCompositeRuleID(types.RuleFQDN(ack.RuleID), ack.ErrorKey)
		if err == nil {
			ruleAcksMap[compositeRuleID] = true
		} else {
			log.Error().Err(err).Interface(ruleIDStr, ack.RuleID).
				Interface(errorKeyStr, ack.ErrorKey).Msg(compositeRuleIDError)
		}
	}
	return
}
