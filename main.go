package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/anirudhRowjee/bunsamosa-bot/database"
	"github.com/anirudhRowjee/bunsamosa-bot/globals"
	"github.com/anirudhRowjee/bunsamosa-bot/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

func main() {
	// Parse YAML File to read in secrets
	// Initialize state
	YAML_SECRETS_PATH := ""

	// Check if we're in a development environment
	IS_DEV_ENV := os.Getenv("BUNSAMOSA_DEV_MODE")

	if IS_DEV_ENV == "1" {
		YAML_SECRETS_PATH = "./secrets-dev.yaml"
	} else {
		YAML_SECRETS_PATH = "/root/bunsamosa-bot/secrets.yaml"
	}

	globals.AppState = globals.App{}
	globals.AppState.LogLevel = os.Getenv("LOG_LEVEL")
	jsonLogDirPath := os.Getenv("JSON_LOG_DIR")

	globals.AppState.JSONLogWriter = openLogFile(jsonLogDirPath)
	globals.AppState.InitializeLogger()

	globals.AppState.ParseFromYAML(YAML_SECRETS_PATH)

	// Initialize the Github Client
	globals.AppState.InitializeGithubClient()

	initLogger := globals.AppState.ZeroLogger.With().Array("scope", zerolog.Arr().Str("INIT")).Logger()

	handlers.TimerDaemonURL = globals.AppState.TimerDaemonURL

	dbManager := &database.DBManager{}
	err := dbManager.Init(globals.AppState.DBPath, globals.AppState.ZeroLogger)
	if err != nil {
		initLogger.Panic().Err(err)
	}
	globals.AppState.DBManager = dbManager
	initLogger.Info().Msg("Initialised Database Successfully!")

	// TODO use Higher-Order Functions to generate this response function
	// with the webhook secret from the YAML Parsed into the app in scope

	mux := http.NewServeMux()
	mux.HandleFunc("POST /Github", handlers.WebhookHandler)
	mux.HandleFunc("GET /leaderboard_mat", handlers.LeaderboardMaterialized)
	mux.HandleFunc("GET /records", handlers.LeaderboardUserSpecific)
	mux.Handle("GET /metrics", promhttp.Handler())

	mux.HandleFunc("/timer", handlers.TimerHandler)

	mux.HandleFunc("GET /ping", handlers.PingHandler)

	// Unutilised routes
	mux.HandleFunc("GET /lb_all", handlers.LeaderboardAllRecords)
	mux.HandleFunc("GET /leaderboard", handlers.Leaderboard_nonmaterialized)
	initLogger.Info().Msg("Registered all routes")

	handler := cors.Default().Handler(mux)
	initLogger.Info().Msg("Initialized CORS")

	webserverAddr := fmt.Sprintf("0.0.0.0:%d", globals.AppState.WebServerPort)
	initLogger.Info().Msg("Starting Web Server")
	err = http.ListenAndServe(webserverAddr, handler)

	if err != nil && err != http.ErrServerClosed {
		initLogger.Fatal().Err(err)
	}
}

func openLogFile(jsonLogsDir string) *os.File {
	if jsonLogsDir == "" {
		panic("JSON_LOG_DIR is a mandatory env var!")
	}
	err := os.MkdirAll(jsonLogsDir, 0777)
	if err != nil {
		panic("JSON Log Directory cannot be created!")
	}
	logRoot, err := os.OpenRoot(jsonLogsDir)
	if err != nil {
		panic("JSON Log Directory cannot be opened!")
	}
	logName := fmt.Sprintf("%d.log", time.Now().Unix())
	logFile, err := logRoot.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		log.Panicf("JSON Log file could not be created! -> %+v", err)
	}

	return logFile
}
