package main

import (
	"net/http"
	"os"

	// "github.com/go-playground/webhooks/v6"

	"github.com/anirudhRowjee/bunsamosa-bot/globals"
	"github.com/anirudhRowjee/bunsamosa-bot/handlers"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

// TODO Write YAML Parsing for environment variables

func main() {
	// parse YAML File to read in secrets
	// Initialize state
	// TODO Separate the YAML Loading from the value setting
	// var YAML_SECRETS_PATH string
	YAML_SECRETS_PATH := ""

	// Check if we're in a development environment
	IS_DEV_ENV := os.Getenv("BUNSAMOSA_DEV_MODE")

	if IS_DEV_ENV == "1" {
		YAML_SECRETS_PATH = "./secrets-dev.yaml"
	} else {
		YAML_SECRETS_PATH = "/root/bunsamosa-bot/secrets.yaml"
	}

	globals.Myapp = globals.App{}
	globals.Myapp.InitializeLogger()

	globals.Myapp.ParseFromYAML(YAML_SECRETS_PATH)

	// Initialize the Github Client
	// globals.Myapp.Initialize_github_client()

	// Initialize the database
	globals.Myapp.InitializeDatabase()

	// Initialize logger for handlers
	handlers.SugaredLogger = globals.Myapp.SugaredLogger

	// Serve!
	// TODO use Higher-Order Functions to generate this response function
	// with the webhook secret from the YAML Parsed into the app in scope

	mux := http.NewServeMux()
	// Utilised routes
	mux.HandleFunc("POST /Github", handlers.WebhookHandler)
	mux.HandleFunc("GET /leaderboard_mat", handlers.LeaderboardMaterialized)
	mux.HandleFunc("GET /records", handlers.LeaderboardUserSpecific)

	// UwU Route
	mux.HandleFunc("GET /ping", handlers.PingHandler)
	mux.HandleFunc("/timer", handlers.TimerHandler)

	// Unutilised routes
	mux.HandleFunc("GET /lb_all", handlers.LeaderboardAllRecords)
	mux.HandleFunc("GET /leaderboard", handlers.Leaderboard_nonmaterialized)
	globals.Myapp.SugaredLogger.Infow("Registered all routes",
		zap.Strings("scope", []string{"INIT"}),
	)

	globals.Myapp.SugaredLogger.Infow("Registered all routes",
		zap.Strings("scope", []string{"INIT"}),
	)

	// NOTE: Stopped servers for testing
	handler := cors.Default().Handler(mux)
	globals.Myapp.SugaredLogger.Infow("Initialized CORS",
		zap.Strings("scope", []string{"INIT"}),
	)

	globals.Myapp.SugaredLogger.Infow("Starting Web Server",
		zap.Strings("scope", []string{"INIT"}),
	)

	err := http.ListenAndServe("0.0.0.0:3000", handler)

	if err != nil && err != http.ErrServerClosed {
		globals.Myapp.SugaredLogger.Errorw("Unable to start server -> %+v", err,
			zap.Strings("scope", []string{"INIT"}),
		)
	}
}
