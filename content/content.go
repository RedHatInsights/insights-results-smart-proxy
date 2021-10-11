// Copyright 2020 Red Hat, Inc
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

// Package content provides API to get rule's content by its `rule id` and `error key`.
// It takes all the work of caching rules taken from content service
package content

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	"github.com/RedHatInsights/insights-operator-utils/types"
	local_types "github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

var (
	ruleContentDirectory      *types.RuleContentDirectory
	ruleContentDirectoryReady = sync.NewCond(&sync.Mutex{})
	stopUpdateContentLoop     = make(chan struct{})
	rulesWithContentStorage   = RulesWithContentStorage{
		rules:                      map[types.RuleID]*types.RuleContent{},
		rulesWithContent:           map[ruleIDAndErrorKey]*local_types.RuleWithContent{},
		recommendationsWithContent: map[types.RuleID]*local_types.RuleWithContent{},
	}
)

type ruleIDAndErrorKey struct {
	RuleID   types.RuleID
	ErrorKey types.ErrorKey
}

// RulesWithContentStorage is a key:value structure to store processed rules.
// It's thread safe
type RulesWithContentStorage struct {
	sync.RWMutex
	rules            map[types.RuleID]*types.RuleContent
	rulesWithContent map[ruleIDAndErrorKey]*local_types.RuleWithContent
	// recommendationsWithContent map has the same contents as rulesWithContent but the keys
	// are composite of "rule.module|ERROR_KEY" optimized for Insights Advisor
	recommendationsWithContent map[types.RuleID]*local_types.RuleWithContent
	internalRuleIDs            []types.RuleID
	externalRuleIDs            []types.RuleID
}

// SetRuleContentDirectory is made for easy testing fake rules etc. from other directories
func SetRuleContentDirectory(contentDir *types.RuleContentDirectory) {
	ruleContentDirectory = contentDir
}

// GetRuleWithErrorKeyContent returns content for rule with error key
func (s *RulesWithContentStorage) GetRuleWithErrorKeyContent(
	ruleID types.RuleID, errorKey types.ErrorKey,
) (*local_types.RuleWithContent, bool) {
	s.RLock()
	defer s.RUnlock()

	res, found := s.rulesWithContent[ruleIDAndErrorKey{
		RuleID:   ruleID,
		ErrorKey: errorKey,
	}]
	return res, found
}

// GetRuleContent returns content for rule
func (s *RulesWithContentStorage) GetRuleContent(ruleID types.RuleID) (*types.RuleContent, bool) {
	s.RLock()
	defer s.RUnlock()

	res, found := s.rules[ruleID]
	return res, found
}

// GetContentForRecommendation returns content for rule with error key
func (s *RulesWithContentStorage) GetContentForRecommendation(
	ruleID types.RuleID,
) (*local_types.RuleWithContent, bool) {
	s.RLock()
	defer s.RUnlock()

	res, found := s.recommendationsWithContent[ruleID]
	return res, found
}

// GetAllContent returns content for rule
func (s *RulesWithContentStorage) GetAllContent() []types.RuleContent {
	s.RLock()
	defer s.RUnlock()

	res := make([]types.RuleContent, 0, len(s.rules))
	for _, rule := range s.rules {
		res = append(res, *rule)
	}

	return res
}

// SetRuleWithContent sets content for rule with error key
func (s *RulesWithContentStorage) SetRuleWithContent(
	ruleID types.RuleID, errorKey types.ErrorKey, ruleWithContent *local_types.RuleWithContent,
) {
	compositeRuleID, err := generators.GenerateCompositeRuleID(types.RuleFQDN(ruleID), errorKey)
	if err == nil {
		s.recommendationsWithContent[compositeRuleID] = ruleWithContent
	} else {
		log.Error().Err(err).Msgf("Error generating composite rule ID for [%v] and [%v]", ruleID, errorKey)
	}

	s.Lock()
	defer s.Unlock()

	s.rulesWithContent[ruleIDAndErrorKey{
		RuleID:   ruleID,
		ErrorKey: errorKey,
	}] = ruleWithContent

	if ruleWithContent.Internal {
		s.internalRuleIDs = append(s.internalRuleIDs, compositeRuleID)
	} else {
		s.externalRuleIDs = append(s.externalRuleIDs, compositeRuleID)
	}
}

// SetRule sets content for rule
func (s *RulesWithContentStorage) SetRule(
	ruleID types.RuleID, ruleContent types.RuleContent,
) {
	s.Lock()
	defer s.Unlock()

	s.rules[ruleID] = &ruleContent
}

// ResetContent clear all the contents
func (s *RulesWithContentStorage) ResetContent() {
	s.Lock()
	defer s.Unlock()

	s.rules = make(map[types.RuleID]*types.RuleContent)
	s.rulesWithContent = make(map[ruleIDAndErrorKey]*local_types.RuleWithContent)
	s.recommendationsWithContent = make(map[types.RuleID]*local_types.RuleWithContent)
	s.internalRuleIDs = make([]types.RuleID, 0)
	s.externalRuleIDs = make([]types.RuleID, 0)
}

// GetRuleIDs gets rule IDs for rules (rule modules)
func (s *RulesWithContentStorage) GetRuleIDs() []string {
	s.Lock()
	defer s.Unlock()

	ruleIDs := make([]string, 0, len(s.rules))

	for _, ruleContent := range s.rules {
		ruleIDs = append(ruleIDs, ruleContent.Plugin.PythonModule)
	}

	return ruleIDs
}

// GetInternalRuleIDs returns the composite rule IDs ("| format") of internal rules
func (s *RulesWithContentStorage) GetInternalRuleIDs() []types.RuleID {
	s.Lock()
	defer s.Unlock()

	return s.internalRuleIDs
}

// GetExternalRuleIDs returns the composite rule IDs ("| format") of external rules
func (s *RulesWithContentStorage) GetExternalRuleIDs() []types.RuleID {
	s.Lock()
	defer s.Unlock()

	return s.externalRuleIDs
}

// WaitForContentDirectoryToBeReady ensures the rule content directory is safe to read/write
func WaitForContentDirectoryToBeReady() {
	// according to the example in the official dock,
	// lock is required here
	if ruleContentDirectory == nil {
		ruleContentDirectoryReady.L.Lock()
		ruleContentDirectoryReady.Wait()
		ruleContentDirectoryReady.L.Unlock()
	}
}

// GetRuleWithErrorKeyContent returns content for rule with provided `rule id` and `error key`.
// Caching is done under the hood, don't worry about it.
func GetRuleWithErrorKeyContent(
	ruleID types.RuleID, errorKey types.ErrorKey,
) (*local_types.RuleWithContent, error) {
	// to be sure the data is there
	WaitForContentDirectoryToBeReady()

	ruleID = types.RuleID(strings.TrimSuffix(string(ruleID), ".report"))

	res, found := rulesWithContentStorage.GetRuleWithErrorKeyContent(ruleID, errorKey)
	if !found {
		return nil, &types.ItemNotFoundError{ItemID: fmt.Sprintf("%v/%v", ruleID, errorKey)}
	}

	return res, nil
}

// GetContentForRecommendation returns content for rule with provided composite rule ID
func GetContentForRecommendation(
	ruleID types.RuleID,
) (*local_types.RuleWithContent, error) {

	WaitForContentDirectoryToBeReady()

	res, found := rulesWithContentStorage.GetContentForRecommendation(ruleID)
	if !found {
		return nil, &types.ItemNotFoundError{ItemID: fmt.Sprintf("%v", ruleID)}
	}

	return res, nil
}

// GetRuleContent returns content for rule with provided `rule id`
// Caching is done under the hood, don't worry about it.
func GetRuleContent(ruleID types.RuleID) (*types.RuleContent, error) {
	// to be sure the data is there
	WaitForContentDirectoryToBeReady()

	ruleID = types.RuleID(strings.TrimSuffix(string(ruleID), ".report"))

	res, found := rulesWithContentStorage.GetRuleContent(ruleID)
	if !found {
		return nil, &types.ItemNotFoundError{ItemID: ruleID}
	}

	return res, nil
}

// ResetContent clear all the content cached
func ResetContent() {
	WaitForContentDirectoryToBeReady()
	rulesWithContentStorage.ResetContent()
}

// GetRuleIDs returns a list of rule IDs (rule modules)
func GetRuleIDs() []string {
	WaitForContentDirectoryToBeReady()

	return rulesWithContentStorage.GetRuleIDs()
}

// GetInternalRuleIDs returns a list of composite rule IDs ("| format") of internal rules
func GetInternalRuleIDs() []types.RuleID {
	WaitForContentDirectoryToBeReady()

	return rulesWithContentStorage.GetInternalRuleIDs()
}

// GetExternalRuleIDs returns a list of composite rule IDs ("| format") of external rules
func GetExternalRuleIDs() []types.RuleID {
	WaitForContentDirectoryToBeReady()

	return rulesWithContentStorage.GetExternalRuleIDs()
}

// GetAllContent returns content for all the loaded rules.
// Caching is done under the hood, don't worry about it.
func GetAllContent() []types.RuleContent {
	// to be sure the data is there
	WaitForContentDirectoryToBeReady()
	return rulesWithContentStorage.GetAllContent()
}

// RunUpdateContentLoop runs loop which updates rules content by ticker
func RunUpdateContentLoop(servicesConf services.Configuration) {
	ticker := time.NewTicker(servicesConf.GroupsPollingTime)

	for {
		UpdateContent(servicesConf)

		select {
		case <-ticker.C:
		case <-stopUpdateContentLoop:
			break
		}
	}
}

// StopUpdateContentLoop stops the loop
func StopUpdateContentLoop() {
	stopUpdateContentLoop <- struct{}{}
}

// UpdateContent function updates rule content
func UpdateContent(servicesConf services.Configuration) {
	var err error

	contentServiceDirectory, err := services.GetContent(servicesConf)
	if err != nil {
		log.Error().Err(err).Msg("Error retrieving static content")
		return
	}

	SetRuleContentDirectory(contentServiceDirectory)
	WaitForContentDirectoryToBeReady()
	ResetContent()
	LoadRuleContent(ruleContentDirectory)
}

// FetchRuleContent - fetching content for particular rule
// Return values:
//   - Structure with rules and content
//   - return true if fetching content was successful, including filtering
//   - return true if the rule has been filtered by OSDElegible field. False otherwise
func FetchRuleContent(rule types.RuleOnReport, OSDEligible bool) (
	ruleWithContentResponse *local_types.RuleWithContentResponse,
	success bool,
	osdFiltered bool,
) {
	ruleID := rule.Module
	errorKey := rule.ErrorKey

	ruleWithContentResponse = nil
	success = false
	osdFiltered = false

	ruleWithContent, err := GetRuleWithErrorKeyContent(ruleID, errorKey)
	if err != nil {
		log.Error().Err(err).Msgf(
			"unable to get content for rule with id %v and error key %v", ruleID, errorKey,
		)
		return
	}

	if OSDEligible && !ruleWithContent.NotRequireAdmin {
		osdFiltered = true
		return
	}

	ruleWithContentResponse = &local_types.RuleWithContentResponse{
		CreatedAt:       ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
		Description:     ruleWithContent.Description,
		ErrorKey:        errorKey,
		Generic:         ruleWithContent.Generic,
		Reason:          ruleWithContent.Reason,
		Resolution:      ruleWithContent.Resolution,
		MoreInfo:        ruleWithContent.MoreInfo,
		TotalRisk:       ruleWithContent.TotalRisk,
		RiskOfChange:    ruleWithContent.RiskOfChange,
		RuleID:          ruleID,
		TemplateData:    rule.TemplateData,
		Tags:            ruleWithContent.Tags,
		UserVote:        rule.UserVote,
		Disabled:        rule.Disabled,
		DisableFeedback: rule.DisableFeedback,
		DisabledAt:      rule.DisabledAt,
		Internal:        ruleWithContent.Internal,
	}
	success = true
	return
}
