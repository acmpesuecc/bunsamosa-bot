package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/acmpesuecc/bunsamosa-bot/globals"
	"github.com/rs/zerolog"
)

func LeaderboardAllRecords(response http.ResponseWriter, request *http.Request) {
	records, err := globals.AppState.LeaderboardGetAllRecords()
	subLogger := globals.AppState.ZeroLogger.With().Str("scope", "LEADERBOARD_HANDLER").Logger()
	if err != nil {
		subLogger.Err(err).Msg("Could not get all records")
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			subLogger.Err(err).Msg("Failed to Marshal records into JSON")
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}

func Leaderboard_nonmaterialized(response http.ResponseWriter, request *http.Request) {
	records, err := globals.AppState.Leaderboard_GetNonMaterialized()
	subLogger := globals.AppState.ZeroLogger.With().Array("scope", zerolog.Arr().Str("LEADERBOARD_HANDLER").Str("NOT_MATERIALIZED")).Logger()
	if err != nil {
		subLogger.Err(err).Msg("Could not get all records")
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			subLogger.Err(err).Msg("Failed to Marshal records into JSON")
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}

func LeaderboardMaterialized(response http.ResponseWriter, request *http.Request) {
	records, err := globals.AppState.Leaderboard_GetMaterialized()
	subLogger := globals.AppState.ZeroLogger.With().Array("scope", zerolog.Arr().Str("LEADERBOARD_HANDLER").Str("MATERIALIZED")).Logger()
	if err != nil {
		subLogger.Err(err).Msg("Could not get all records")
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			subLogger.Err(err).Msg("Failed to Marshal records into JSON")
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}

func LeaderboardUserSpecific(response http.ResponseWriter, request *http.Request) {
	queryParams := request.URL.Query()
	user := queryParams.Get("user")
	if user == "" {
		http.Error(response, "user param is required", http.StatusBadRequest)
		return
	}

	records, err := globals.AppState.Leaderboard_GetUserRecords(user)
	subLogger := globals.AppState.ZeroLogger.With().Array("scope", zerolog.Arr().Str("LEADERBOARD_HANDLER").Str("USER_SPECIFIC")).Logger()
	if err != nil {
		subLogger.Err(err).Msg("Could not get all records")
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			subLogger.Err(err).Msg("Failed to Marshal records into JSON")
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}
