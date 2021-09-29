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
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/parsers"
	"github.com/RedHatInsights/insights-operator-utils/types"
)

// HTTP response-related constants
const (
	contentType = "Content-Type"
	appJSON     = "application/json; charset=utf-8"

	authTokenFormatError       = "unable to read orgID and userID from auth. token!"
	improperRuleSelectorFormat = "improper rule selector format"
	readRuleStatusError        = "read rule status error"
)

// method getAcknowledge retrieves the info about rule acknowledgement made
// from this account. Acks are created, deleted, and queired by Insights rule
// ID, not by their own ack ID.
//
// An example response:
//
// {
//   "rule": "string",
//   "justification": "string",  <- can not be set by this call!!!
//   "created_by": "string",
//   "created_at": "2021-09-04T17:52:48.976Z",
//   "updated_at": "2021-09-04T17:52:48.976Z"
// }
func (server *HTTPServer) getAcknowledge(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set(contentType, appJSON)

	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		// everything's handled already
		return
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		log.Error().Err(err).Msg(improperRuleSelectorFormat)
		// server error has been handled already
		return
	}

	// we seem to have all data -> let's display them
	logFullRuleSelector(orgID, userID, ruleID, errorKey)

	// test if the rule has been acknowledged already
	ruleAck, found, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// rule was not acked -> nothing to return
	if !found {
		writer.WriteHeader(http.StatusNotFound)
		log.Info().Msg("Rule has not been disabled previously -> nothing to return!")
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
// {
//   "rule_id": "string",
//   "justification": "string"
// }
//
// An example response:
//
// {
//   "rule": "string",
//   "justification": "string",  <- can not be set by this call!!!
//   "created_by": "string",
//   "created_at": "2021-09-04T17:52:48.976Z",
//   "updated_at": "2021-09-04T17:52:48.976Z"
// }
//
// HTTP/1.1 200 OK is returned if rule has been already acked
// HTTP/1.1 201 Created is returned if rule has been acked by this call
func (server *HTTPServer) acknowledgePost(writer http.ResponseWriter, request *http.Request) {
	const readRuleJustificationError = "can not retrieve rule disable justification from Aggregator"

	writer.Header().Set(contentType, appJSON)

	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		// everything's handled already
		return
	}

	parameters, err := readRuleSelectorAndJustificationFromBody(writer, request)
	if err != nil {
		// everything's handled already
		return
	}

	// we seem to have all data -> let's display them
	log.Info().
		Int("org", int(orgID)).
		Str("account", string(userID)).
		Str("rule", string(parameters.RuleSelector)).
		Str("value", parameters.Value).
		Msg("Proper payload provided")

	// check if rule selector has the proper format
	ruleID, errorKey, err := parsers.ParseRuleSelector(parameters.RuleSelector)
	if err != nil {
		log.Error().Err(err).Msg(improperRuleSelectorFormat)
		// return HTTP code 400 to client
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// display parsed rule ID and error key
	log.Info().
		Str("ruleID", string(ruleID)).
		Str("errorKey", string(errorKey)).
		Msg("Parsed rule selector")

	// test if the rule has been acknowledged already
	_, found, err := server.readRuleDisableStatus(ruleID, errorKey, orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// if acknowledgement has been found -> return 200 OK with the existing rule ack
	// if acknowledgement has NOT been found -> return 201 Created with the created rule ack
	if found {
		writer.WriteHeader(http.StatusOK)
		log.Info().Msg("Rule has been already disabled")
	} else {
		writer.WriteHeader(http.StatusCreated)
		log.Info().Msg("Rule has not been disabled previously")

		// acknowledge rule
		err := server.ackRuleSystemWide(ruleID, errorKey, orgID, userID, parameters.Value)
		if err != nil {
			log.Error().Err(err).Msg(readRuleJustificationError)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Aggregator REST API is source of truth - let's re-read rule status
	// from it
	updatedAcknowledgement, _, err := server.readRuleDisableStatus(ruleID, errorKey, orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleJustificationError)
		http.Error(writer, err.Error(), http.StatusBadRequest)
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
	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		// everything's handled already
		return
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		log.Error().Err(err).Msg(improperRuleSelectorFormat)
		// server error has been handled already
		return
	}

	// we seem to have all data -> let's display them
	logFullRuleSelector(orgID, userID, ruleID, errorKey)

	// test if the rule has been acknowledged already
	_, found, err := server.readRuleDisableStatus(types.Component(ruleID), errorKey, orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(readRuleStatusError)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if !found {
		writer.WriteHeader(http.StatusNotFound)
		log.Info().Msg("Rule has not been disabled previously -> ACK won't be deleted")
		return
	}

	// rule has been found -> let's delete the ACK
	// delete acknowledgement for a rule
	log.Info().Msg("About to delete ACK for a rule")
	err = server.deleteAckRuleSystemWide(types.Component(ruleID), errorKey, orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to delete rule acknowledgement")
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// return 204 -> rule ack has been deleted
	writer.WriteHeader(http.StatusNoContent)
}
