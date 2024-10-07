package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/anirudhRowjee/bunsamosa-bot/globals"
)

func LeaderboardAllRecords(response http.ResponseWriter, request *http.Request) {
	records, err := globals.Myapp.LeaderboardGetAllRecords()
	if err != nil {
		log.Println("[ERR][LEADERBOARD_HANDLER] Could not get all records ->", err)
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			log.Println("[ERR][LEADERBOARD_HANDLER] Failed to Marshal records into JSON ->", err)
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}

func Leaderboard_nonmaterialized(response http.ResponseWriter, request *http.Request) {

	records, err := globals.Myapp.Leaderboard_GetNonMaterialized()
	if err != nil {
		log.Println("[ERROR][LEADERBOARD_HANDLER][NOT-MATERIALIZED] Could not get all records ->", err)
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			log.Println("[ERROR][LEADERBOARD_HANDLER][NOT-MATERIALIZED] Failed to Marshal records into JSON ->", err)
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}

func LeaderboardMaterialized(response http.ResponseWriter, request *http.Request) {

	records, err := globals.Myapp.Leaderboard_GetMaterialized()
	if err != nil {
		log.Println("[ERROR][LEADERBOARD_HANDLER][MATERIALIZED] Could not get all records ->", err)
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			log.Println("[ERROR][LEADERBOARD_HANDLER][MATERIALIZED] Failed to Marshal records into JSON ->", err)
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

	records, err := globals.Myapp.Leaderboard_GetUserRecords(user)
	if err != nil {
		log.Println("[ERROR][LEADERBOARD_HANDLER][USERSPECIFIC] Could not get all records ->", err)
		response.WriteHeader(http.StatusInternalServerError)
	} else {
		// Marshal into JSON
		json_string, err := json.Marshal(records)
		if err != nil {
			log.Println("[ERROR][LEADERBOARD_HANDLER][USERSPECIFIC] Failed to Marshal records into JSON ->", err)
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.Header().Set("Content-Type", "application/json")
			response.Write(json_string)
		}
	}
}
