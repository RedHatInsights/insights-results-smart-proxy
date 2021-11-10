package server

import (
	"encoding/json"
	"fmt"

	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/rs/zerolog/log"
)

func logClusterInfos(orgID types.OrgID, clusterID types.ClusterName, response []types.RuleOnReport) {
	logMessage := fmt.Sprintf("rule hits for %d.%s:", orgID, clusterID)
	for _, ruleHit := range response {
		logMessage += fmt.Sprintf("\n\trule: %s; error key: %s", ruleHit.Module, ruleHit.ErrorKey)
	}
	log.Info().Msg(logMessage)
}

func logClusterInfo(orgID types.OrgID, clusterID types.ClusterName, response *types.RuleOnReport) {
	logClusterInfos(orgID, clusterID, []types.RuleOnReport{*response})
}

func logClustersReport(orgID types.OrgID, reports map[types.ClusterName]json.RawMessage) {
	var report []types.RuleOnReport
	for clusterName, jsonReport := range reports {
		err := json.Unmarshal(jsonReport, &report)
		if err != nil {
			log.Info().Msg("can't log report for cluster " + string(clusterName))
			continue
		}
		logClusterInfos(orgID, clusterName, report)
	}
}
