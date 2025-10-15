package globals

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"path/filepath"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/acmpesuecc/bunsamosa-bot/database"
)

type App struct {

	// Initialization Information
	WebhookSecret string
	AppID         int
	OrgID         int
	CertPath      string
	WebServerPort int

	// Runtime Variables and Global Dependencies
	RuntimeClient *github.Client
	AppTransport  *ghinstallation.AppsTransport

	// Add Database Dependencies
	DBPath    string
	DBManager *database.DBManager

	// Console Logger for pretty printed logs
	LogLevel      string
	JSONLogWriter *os.File
	ZeroLogger    *zerolog.Logger

	TimerDaemonURL string

	MentionMaintainerLead string
	MentionTechLead       string
}

var AppState App

func (a *App) ParseFromYAML(path string) {
	yamlInitLogger := a.ZeroLogger.With().Array("scope", zerolog.Arr().Str("INIT").Str("YAML")).Logger()

	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		yamlInitLogger.Fatal().Err(err)
	}

	var yaml_output map[string]string

	err = yaml.Unmarshal(yamlFile, &yaml_output)
	if err != nil {
		yamlInitLogger.Fatal().Err(err).Msg("Failed to unmarshal YAML")
	}

	a.CertPath = yaml_output["certPath"]
	a.WebhookSecret = yaml_output["webhookSecret"]

	a.AppID, err = strconv.Atoi(yaml_output["appID"])
	if err != nil {
		yamlInitLogger.Fatal().Err(err).Msg("Could not parse appID")
	}

	a.OrgID, err = strconv.Atoi(yaml_output["orgID"])
	if err != nil {
		yamlInitLogger.Fatal().Err(err).Msg("Could not parse orgID")
	}

	a.WebServerPort, err = strconv.Atoi(yaml_output["webServerPort"])
	if err != nil {
		yamlInitLogger.Fatal().Err(err).Msg("Could not parse webServerPort")
	}

	a.DBPath = yaml_output["dbPath"]
	a.TimerDaemonURL = yaml_output["timerDaemonURL"]

	a.MentionMaintainerLead = yaml_output["maintainer-leads"]
	a.MentionTechLead = yaml_output["tech-leads"]
	yamlInitLogger.Info().Msg("YAML parsed successfully")
}

func (a *App) InitializeGithubClient() {
	ghInitLogger := a.ZeroLogger.With().Array("scope", zerolog.Arr().Str("INIT").Str("GITHUB")).Logger()
	ghInitLogger.Info().Msg("Initializing Github client...")

	app_transport, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, int64(a.AppID), a.CertPath)
	if err != nil {
		ghInitLogger.Fatal().Err(err).Msg("Could not Create Github App Client")
	}
	a.AppTransport = app_transport

	ghInitLogger.Info().Msg("App Transport Initialized")

	installation, _, err := github.NewClient(&http.Client{Transport: app_transport}).Apps.FindOrganizationInstallation(context.TODO(), fmt.Sprint(a.OrgID))
	if err != nil {
		ghInitLogger.Fatal().Err(err).Msg("Could not Find Organization installation")
	}
	ghInitLogger.Info().Msg("Organization Transport Initialized")

	installationID := installation.GetID()
	installation_transport := ghinstallation.NewFromAppsTransport(app_transport, installationID)

	a.RuntimeClient = github.NewClient(&http.Client{Transport: installation_transport})
	ghInitLogger.Info().Str("Installation-ID", fmt.Sprint(installationID)).Any("Installation-Events", installation.Events).Msg("Successfully initialized Github app client")
}

func (a *App) InitializeLogger() {
	consoleLogger := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(zerolog.MultiLevelWriter(consoleLogger, a.JSONLogWriter)).With().Timestamp().Caller().Logger()
	a.ZeroLogger = &logger

	switch strings.ToLower(a.LogLevel) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		panic("Invalid log level, available levels: trace, debug, error, info")
	}

	a.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("INIT").Str("LOGGER")).Msg("Initialized Console Logger")
}

func (a *App) LeaderboardGetAllRecords() ([]database.BountyLogging, error) {

	// Get all the time series data present so far
	// from the database
	var all_records []database.BountyLogging

	records, err := a.DBManager.GetAllRecords()
	if err != nil {
		return nil, err
	} else {
		all_records = records
	}

	return all_records, nil

}

func (a *App) Leaderboard_GetNonMaterialized() ([]database.LeaderboardEntry, error) {

	// Get a non-materialized view of the leaderboard
	var leaderboard []database.LeaderboardEntry

	records, err := a.DBManager.GetLeaderboard()
	if err != nil {
		return nil, err
	} else {
		leaderboard = records
	}

	return leaderboard, nil

}

func (a *App) Leaderboard_GetMaterialized() ([]database.LeaderboardEntry, error) {

	// Get a materialized view of the leaderboard
	var leaderboard []database.LeaderboardEntry

	records, err := a.DBManager.GetLeaderboardMat()
	if err != nil {
		return nil, err
	} else {
		leaderboard = records
	}

	return leaderboard, nil

}

func (a *App) Leaderboard_GetUserRecords(user string) ([]database.BountyLogging, error) {
	// Take a user's username and return their records

	// Get all the time series data present so far
	// from the database
	var all_records []database.BountyLogging

	records, err := a.DBManager.GetUserRecords(user)
	if err != nil {
		return nil, err
	} else {
		all_records = records
	}

	return all_records, nil
}
