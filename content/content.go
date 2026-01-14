// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

var (
	ruleContentDirectory      *ctypes.RuleContentDirectory
	ruleContentDirectoryReady = sync.NewCond(&sync.Mutex{})
	stopUpdateContentLoop     = make(chan struct{})
	rulesWithContentStorage   = getEmptyRulesWithContentMap()
	contentDirectoryTimeout   = 5 * time.Second
	dotReport                 = ".report"
)

type ruleIDAndErrorKey struct {
	RuleID   ctypes.RuleID
	ErrorKey ctypes.ErrorKey
}

// RulesWithContentStorage is a key:value structure to store processed rules.
// It's thread safe
type RulesWithContentStorage struct {
	rules            map[ctypes.RuleID]*ctypes.RuleContent
	rulesWithContent map[ruleIDAndErrorKey]*types.RuleWithContent
	// recommendationsWithContent map has the same contents as rulesWithContent but the keys
	// are composite of "rule.module|ERROR_KEY" optimized for Insights Advisor
	recommendationsWithContent map[ctypes.RuleID]*types.RuleWithContent
	internalRuleIDs            []ctypes.RuleID
	externalRuleIDs            []ctypes.RuleID
}

// SetRuleContentDirectory is made for easy testing fake rules etc. from other directories
func SetRuleContentDirectory(contentDir *ctypes.RuleContentDirectory) {
	ruleContentDirectory = contentDir
}

// GetRuleWithErrorKeyContent returns content for rule with error key
func (s *RulesWithContentStorage) GetRuleWithErrorKeyContent(
	ruleID ctypes.RuleID, errorKey ctypes.ErrorKey,
) (*types.RuleWithContent, bool) {
	res, found := s.rulesWithContent[ruleIDAndErrorKey{
		RuleID:   ruleID,
		ErrorKey: errorKey,
	}]
	return res, found
}

// GetContentForRecommendation returns content for rule with error key
func (s *RulesWithContentStorage) GetContentForRecommendation(
	ruleID ctypes.RuleID,
) (*types.RuleWithContent, bool) {
	res, found := s.recommendationsWithContent[ruleID]
	return res, found
}

// GetAllContentV1 returns content for rule for api v1
func (s *RulesWithContentStorage) GetAllContentV1() []types.RuleContentV1 {
	res := make([]types.RuleContentV1, 0, len(s.rules))
	for _, rule := range s.rules {
		res = append(res, RuleContentToV1(rule))
	}

	return res
}

// GetAllContentV2 returns content for api/v2
func (s *RulesWithContentStorage) GetAllContentV2() []types.RuleContentV2 {
	res := make([]types.RuleContentV2, 0, len(s.rules))
	for _, rule := range s.rules {
		res = append(res, RuleContentToV2(rule))
	}

	return res
}

// SetRuleWithContent sets content for rule with error key
func (s *RulesWithContentStorage) SetRuleWithContent(
	ruleID ctypes.RuleID, errorKey ctypes.ErrorKey, ruleWithContent *types.RuleWithContent,
) {
	compositeRuleID, err := generators.GenerateCompositeRuleID(ctypes.RuleFQDN(ruleID), errorKey)
	if err == nil {
		s.recommendationsWithContent[compositeRuleID] = ruleWithContent
	} else {
		log.Warn().Err(err).Msgf("Error generating composite rule ID for [%v] and [%v]", ruleID, errorKey)
	}

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
	ruleID ctypes.RuleID, ruleContent *ctypes.RuleContent,
) {
	s.rules[ruleID] = ruleContent
}

// GetRuleIDs gets rule IDs for rules (rule modules)
func (s *RulesWithContentStorage) GetRuleIDs() []string {
	ruleIDs := make([]string, 0, len(s.rules))

	for _, ruleContent := range s.rules {
		ruleIDs = append(ruleIDs, ruleContent.Plugin.PythonModule)
	}

	return ruleIDs
}

// GetInternalRuleIDs returns the composite rule IDs ("| format") of internal rules
func (s *RulesWithContentStorage) GetInternalRuleIDs() []ctypes.RuleID {
	return s.internalRuleIDs
}

// GetExternalRuleIDs returns the composite rule IDs ("| format") of external rules
func (s *RulesWithContentStorage) GetExternalRuleIDs() []ctypes.RuleID {
	return s.externalRuleIDs
}

// GetExternalRuleSeverities returns a map of external rule IDs and their severity (total risk)
// along with a list of unique severities
func (s *RulesWithContentStorage) GetExternalRuleSeverities() (
	severityMap map[ctypes.RuleID]int,
	uniqueSeverities []int,
) {
	severityMap = make(map[ctypes.RuleID]int)
	uniqueMap := make(map[int]interface{})

	for _, ruleID := range s.externalRuleIDs {
		totalRisk := s.recommendationsWithContent[ruleID].TotalRisk
		severityMap[ruleID] = totalRisk
		uniqueMap[totalRisk] = nil
	}

	for k := range uniqueMap {
		uniqueSeverities = append(uniqueSeverities, k)
	}

	return
}

// GetExternalRulesManagedInfo returns a map of rule IDs and the information whether a rule is managed
// (has osd_customer tag) or not
func (s *RulesWithContentStorage) GetExternalRulesManagedInfo() (managedMap map[ctypes.RuleID]bool) {
	managedMap = make(map[ctypes.RuleID]bool)

	for _, ruleID := range s.externalRuleIDs {
		managedMap[ruleID] = s.recommendationsWithContent[ruleID].OSDCustomer
	}

	return
}

// RuleContentDirectoryTimeoutError is used, when the content directory is empty for too long time
type RuleContentDirectoryTimeoutError struct{}

func (e *RuleContentDirectoryTimeoutError) Error() string {
	return "Content directory cache has been empty for too long time; timeout triggered"
}

// WaitForContentDirectoryToBeReady ensures the rule content directory is safe to read/write
func WaitForContentDirectoryToBeReady() error {
	// according to the example in the official dock,
	// lock is required here
	if ruleContentDirectory == nil {
		ruleContentDirectoryReady.L.Lock()

		done := make(chan struct{})
		go func() {
			ruleContentDirectoryReady.Wait()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(contentDirectoryTimeout):
			err := &RuleContentDirectoryTimeoutError{}
			log.Error().Err(err).Msg("Cannot retrieve content")
			return err
		}

		ruleContentDirectoryReady.L.Unlock()
	}

	return nil
}

// GetRuleWithErrorKeyContent returns content for rule with provided `rule id` and `error key`.
// Caching is done under the hood, don't worry about it.
func GetRuleWithErrorKeyContent(
	ruleID ctypes.RuleID, errorKey ctypes.ErrorKey,
) (*types.RuleWithContent, error) {
	// to be sure the data is there
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	ruleID = ctypes.RuleID(strings.TrimSuffix(string(ruleID), dotReport))

	res, found := rulesWithContentStorage.GetRuleWithErrorKeyContent(ruleID, errorKey)
	if !found {
		return nil, &utypes.ItemNotFoundError{ItemID: fmt.Sprintf("%v/%v", ruleID, errorKey)}
	}

	return res, nil
}

// GetContentForRecommendation returns content for rule with provided composite rule ID
func GetContentForRecommendation(
	ruleID ctypes.RuleID,
) (*types.RuleWithContent, error) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	res, found := rulesWithContentStorage.GetContentForRecommendation(ruleID)
	if !found {
		return nil, &utypes.ItemNotFoundError{ItemID: fmt.Sprintf("%v", ruleID)}
	}

	return res, nil
}

// GetRuleContentV1 returns content for rule with provided `rule id`
// Caching is done under the hood, don't worry about it.
func GetRuleContentV1(ruleID ctypes.RuleID) (*types.RuleContentV1, error) {
	res, err := getRuleContent(ruleID)
	if err == nil {
		resV1 := RuleContentToV1(res)
		return &resV1, nil
	}
	return nil, err
}

// GetRuleContentV2 provides single rule for api v2
func GetRuleContentV2(ruleID ctypes.RuleID) (*types.RuleContentV2, error) {
	res, err := getRuleContent(ruleID)
	if err == nil {
		resV2 := RuleContentToV2(res)
		return &resV2, nil
	}
	return nil, err
}

func getRuleContent(ruleID ctypes.RuleID) (*ctypes.RuleContent, error) {
	// to be sure the data is there
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	ruleID = ctypes.RuleID(strings.TrimSuffix(string(ruleID), dotReport))

	res, found := rulesWithContentStorage.getRuleContent(ruleID)
	if !found {
		return nil, &utypes.ItemNotFoundError{ItemID: ruleID}
	}

	return res, nil
}

func getEmptyRulesWithContentMap() *RulesWithContentStorage {
	s := RulesWithContentStorage{}
	s.rules = make(map[types.RuleID]*types.RuleContent)
	s.rulesWithContent = make(map[ruleIDAndErrorKey]*types.RuleWithContent)
	s.recommendationsWithContent = make(map[ctypes.RuleID]*types.RuleWithContent)
	s.internalRuleIDs = make([]ctypes.RuleID, 0)
	s.externalRuleIDs = make([]ctypes.RuleID, 0)
	return &s
}

// GetRuleIDs returns a list of rule IDs (rule modules)
func GetRuleIDs() ([]string, error) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	return rulesWithContentStorage.GetRuleIDs(), nil
}

// GetInternalRuleIDs returns a list of composite rule IDs ("| format") of internal rules
func GetInternalRuleIDs() ([]ctypes.RuleID, error) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	return rulesWithContentStorage.GetInternalRuleIDs(), nil
}

// GetExternalRuleIDs returns a list of composite rule IDs ("| format") of external rules
func GetExternalRuleIDs() ([]ctypes.RuleID, error) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	return rulesWithContentStorage.GetExternalRuleIDs(), nil
}

// GetExternalRuleSeverities returns a map of rule IDs and their severity (total risk),
// along with a list of unique severities
func GetExternalRuleSeverities() (
	map[ctypes.RuleID]int,
	[]int,
	error,
) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, nil, err
	}

	severityMap, uniqueSeverities := rulesWithContentStorage.GetExternalRuleSeverities()
	return severityMap, uniqueSeverities, nil
}

// GetExternalRulesManagedInfo returns a map of rule IDs and the information whether a rule is managed
// (has osd_customer tag) or not
func GetExternalRulesManagedInfo() (
	map[ctypes.RuleID]bool, error,
) {
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	managedMap := rulesWithContentStorage.GetExternalRulesManagedInfo()
	return managedMap, nil
}

// GetAllContentV1 returns content for all the loaded rules.
func GetAllContentV1() ([]types.RuleContentV1, error) {
	// to be sure the data is there
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	return rulesWithContentStorage.GetAllContentV1(), nil
}

// GetAllContentV2 returns content for api v2
func GetAllContentV2() ([]types.RuleContentV2, error) {
	// to be sure the data is there
	err := WaitForContentDirectoryToBeReady()

	if err != nil {
		return nil, err
	}

	return rulesWithContentStorage.GetAllContentV2(), nil
}

// RunUpdateContentLoop runs loop which updates rules content by ticker
func RunUpdateContentLoop(servicesConf services.Configuration) {
	ticker := time.NewTicker(servicesConf.GroupsPollingTime)

	for {
		UpdateContent(servicesConf)

		select {
		case <-ticker.C:
		case <-stopUpdateContentLoop:
			return
		}
	}
}

// SetContentDirectoryTimeout sets the maximum duration for which
// the smart proxy waits if the content directory is empty
func SetContentDirectoryTimeout(timeout time.Duration) {
	contentDirectoryTimeout = timeout
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
	err = WaitForContentDirectoryToBeReady()
	if err != nil {
		return
	}
	LoadRuleContent(ruleContentDirectory)
}

// FetchRuleContent - fetching content for particular rule
// Return values:
//   - Structure with rules and content
//   - return true if the rule has been filtered by OSDElegible field. False otherwise
//   - return error if the one occurred during retrieval
func FetchRuleContent(rule *ctypes.RuleOnReport, OSDEligible bool) (
	ruleWithContentResponse *types.RuleWithContentResponse,
	osdFiltered bool,
	err error,
) {
	ruleID := rule.Module
	errorKey := rule.ErrorKey

	ruleWithContentResponse = nil
	osdFiltered = false

	ruleWithContent, err := GetRuleWithErrorKeyContent(ruleID, errorKey)
	if err != nil {
		log.Warn().Err(err).Msgf(
			"unable to get content for rule with id %v and error key %v", ruleID, errorKey,
		)
		return
	}

	if OSDEligible && !ruleWithContent.OSDCustomer {
		osdFiltered = true
		return
	}

	ruleWithContentResponse = &types.RuleWithContentResponse{
		CreatedAt:       ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
		Description:     ruleWithContent.Description,
		ErrorKey:        errorKey,
		Generic:         ruleWithContent.Generic,
		Reason:          ruleWithContent.Reason,
		Resolution:      ruleWithContent.Resolution,
		MoreInfo:        ruleWithContent.MoreInfo,
		TotalRisk:       ruleWithContent.TotalRisk,
		RuleID:          ruleID,
		TemplateData:    rule.TemplateData,
		Tags:            ruleWithContent.Tags,
		UserVote:        rule.UserVote,
		Disabled:        rule.Disabled,
		DisableFeedback: rule.DisableFeedback,
		DisabledAt:      rule.DisabledAt,
		Internal:        ruleWithContent.Internal,
	}
	return
}
