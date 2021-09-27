package server

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

func extractRatingArrayFromBody(request *http.Request) types.RuleRatingArray {
	var ruleRatings types.RuleRatingArray

	decoder := json.NewDecoder(request.Body)
	if err := decoder.Decode(&ruleRatings); err != nil {
		log.Error().Err(err).Msg("Error decoding rating array from the request")
	}

	return ruleRatings
}
