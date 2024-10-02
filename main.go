package main

import (
	"log"
	"net/http"
	//"os"

	// "github.com/go-playground/webhooks/v6"

	//"github.com/anirudhRowjee/bunsamosa-bot/globals"
	"github.com/anirudhRowjee/bunsamosa-bot/handlers"
	"github.com/rs/cors"
)

// TODO Write YAML Parsing for environment variables

func main() {

	// parse YAML File to read in secrets
	// Initialize state
	// TODO Separate the YAML Loading from the value setting
	// var YAML_SECRETS_PATH string
	// YAML_SECRETS_PATH := ""
	//
	// // Check if we're in a development environment
	// IS_DEV_ENV := os.Getenv("BUNSAMOSA_DEV_MODE")
	//
	// if IS_DEV_ENV == "1" {
	// 	YAML_SECRETS_PATH = "./secrets-dev.yaml"
	// } else {
	// 	YAML_SECRETS_PATH = "/root/bunsamosa-bot/secrets.yaml"
	// }
	//
	// globals.Myapp = globals.App{}
	//
	// globals.Myapp.Parse_from_YAML(YAML_SECRETS_PATH)
	// log.Println("[INIT] YAML Parsed successfully")
	//
	// // Initialize the Github Client
	// globals.Myapp.Initialize_github_client()
	// // Initialize the database
	// globals.Myapp.Initialize_database()
	//
	// // Serve!
	// // TODO use Higher-Order Functions to generate this response function
	// // with the webhook secret from the YAML Parsed into the app in scope

	mux := http.NewServeMux()
	mux.HandleFunc("POST /Github", handlers.WebhookHandler)
	mux.HandleFunc("GET /ping", handlers.PingHandler)
	mux.HandleFunc("GET /lb_all", handlers.Leaderboard_allrecords)
	mux.HandleFunc("GET /leaderboard", handlers.Leaderboard_nonmaterialized)
	mux.HandleFunc("GET /leaderboard_mat", handlers.Leaderboard_materialized)
	mux.HandleFunc("GET /records", handlers.Leaderboard_userspecific)
	log.Println("[INIT] Registered all routes")

	handler := cors.Default().Handler(mux)
	log.Println("[INIT] Initialized CORS")
	log.Println("[INIT] Starting Web Server")
	err := http.ListenAndServe("0.0.0.0:3000", handler)

	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
