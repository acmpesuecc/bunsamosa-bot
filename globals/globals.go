package globals

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"path/filepath"

	v3 "github.com/google/go-github/v47/github"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"

	"github.com/anirudhRowjee/bunsamosa-bot/database"
	"github.com/bradleyfalzon/ghinstallation/v2"
)

type App struct {

	// Initialization Information
	WebhookSecret string
	AppID         int
	OrgID         int
	CertPath      string

	// Runtime Variables and Global Dependencies
	RuntimeClient *v3.Client
	AppTransport  *ghinstallation.AppsTransport

	// Add Database Dependencies
	Db_connection_string string
	Dbmanager            *database.DBManager

	// Sugar Logger for structured logging
	SugaredLogger *zap.SugaredLogger

	TimerDaemonURL string
}

var Myapp App

func (a *App) ParseFromYAML(path string) {

	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		// log.Println("[ERROR] Invalid Secrets YAML Filepath")
		a.SugaredLogger.Panicw("Invalid Secrets YAML Filepath", zap.Strings("scope", []string{"ERROR"}))
		// panic(err)
	}

	var yaml_output map[string]string

	err = yaml.Unmarshal(yamlFile, &yaml_output)
	if err != nil {
		// log.Println("[ERROR] Could not Unmarshal YAML")
		a.SugaredLogger.Panicw("Could not Unmarshal YAML", zap.Strings("scope", []string{"ERROR"}))
		// panic(err)
	}

	// TODO Add error reporting here
	// log.Println("[SECRETS] YAML Parsing Complete")
	a.SugaredLogger.Infof("YAML Parsing Complete", zap.Strings("scope", []string{"SECRETS"}))

	a.CertPath = yaml_output["certPath"]
	a.WebhookSecret = yaml_output["webhookSecret"]

	// TODO better way to do this?
	a.AppID, err = strconv.Atoi(yaml_output["appID"])
	if err != nil {
		a.SugaredLogger.Panicw("Could not Parse AppID", zap.Strings("scope", []string{"SECRETS"}))
		// panic(err)
	}
	a.OrgID, err = strconv.Atoi(yaml_output["orgID"])
	if err != nil {
		a.SugaredLogger.Panicw("Could not Parse OrgID", zap.Strings("scope", []string{"SECRETS"}))
		// panic(err)
	}

	// Read in the Connection String
	a.Db_connection_string = yaml_output["dbConnectionString"]

	a.TimerDaemonURL = yaml_output["timerDaemonURL"]

	a.SugaredLogger.Infof("YAML Parsed successfully", zap.Strings("scope", []string{"INIT"}))
}

func (a *App) InitializeGithubClient() {
	// Initialize the Github Client and AppTransport
	// log.Println("[CLIENT] Initializing Github Client")
	a.SugaredLogger.Infof("Initialized Github Client", zap.Strings("scope", []string{"CLIENT"}))

	app_transport, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, int64(a.AppID), a.CertPath)

	// Push to gloabl scope
	a.AppTransport = app_transport

	if err != nil {
		// log.Println("[ERROR] Could not Create Github App Client")
		a.SugaredLogger.Panicw("Could not Create Github App Client", err, zap.String("scope", "ERROR"))
		// panic(err)
	}

	// log.Println("[CLIENT] App Transport Initialized")
	a.SugaredLogger.Infof("App Transport Initialized", zap.String("scope", "CLIENT"))

	// NOTE Don't forget to install the app in your repository before you do this!
	// Initialize the installation
	installation, _, err := v3.NewClient(&http.Client{Transport: app_transport}).Apps.FindOrganizationInstallation(context.TODO(), fmt.Sprint(a.OrgID))
	if err != nil {
		// log.Println("[ERROR] Could not Find Organization installation", err)
		a.SugaredLogger.Panicw("Could not Find Organization installation", err, zap.String("scope", "ERROR"))
		panic(err)
	}
	// log.Println("[CLIENT] Organization Transport Initialized")
	a.SugaredLogger.Infof("Organization Transport Initialized", zap.String("scope", "CLIENT"))

	// Initialize an authenticated transport for the installation
	installationID := installation.GetID()
	installation_transport := ghinstallation.NewFromAppsTransport(app_transport, installationID)

	a.RuntimeClient = v3.NewClient(&http.Client{Transport: installation_transport})

	// log.Printf("[CLIENT] successfully initialized GitHub app client, installation-id:%s expected-events:%v\n", fmt.Sprint(installationID), installation.Events)
	a.SugaredLogger.Infof("Successfully initialized Github app client, installation-id:%s expected-events:%v",
		fmt.Sprint(installationID),
		installation.Events,
		zap.String("scope", "CLIENT"),
	)
}

func (a *App) InitializeDatabase() {
	// Start the database. Panic on error.

	dbmanager := database.DBManager{}
	// log.Println("[DATABASE] Initializing Database Manager")
	a.SugaredLogger.Logw(zap.InfoLevel, "Initializing Database Manager",
		zap.Strings("scope", []string{"DATABASE"}),
	)
	err := dbmanager.Init(a.Db_connection_string, a.SugaredLogger)
	if err != nil {
		// log.Panicln("[DATABASE] DB Initialization Failed ->", err)
		a.SugaredLogger.Panicw("DB Initialization Failed -> %+v", err, zap.String("scope", "DATABASE"))
	} else {
		a.Dbmanager = &dbmanager
		// log.Println("[DATABASE] DB Manager Initialized successfully")
		a.SugaredLogger.Infof("DB Manager Initialized Successfully", zap.Strings("scope", []string{"DATABASE"}))
	}
}

func (a *App) InitializeLogger() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a custom encoder configuration
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Apply the custom encoder configuration using WithOptions
	customLogger := logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig), // Custom encoder
			zapcore.AddSync(os.Stderr),            // Sync to stdout
			c,                                     // Use the same level as the original core
		)
	}))
	sugar := customLogger.Sugar()
	sugar.Infof("Initialized Logger",
		zap.Strings("scope", []string{"INIT"}),
	)
	a.SugaredLogger = sugar
}

func (a *App) LeaderboardGetAllRecords() ([]database.ContributorRecordModel, error) {

	// Get all the time series data present so far
	// from the database
	var all_records []database.ContributorRecordModel

	// Use the database method
	records, err := a.Dbmanager.GetAllRecords()
	if err != nil {
		return nil, err
	} else {
		all_records = records
	}

	return all_records, nil

}

func (a *App) AssignBountyPoints() ([]database.ContributorRecordModel, error) {

	// Get all the time series data present so far
	// from the database
	var all_records []database.ContributorRecordModel

	// Use the database method
	records, err := a.Dbmanager.GetAllRecords()
	if err != nil {
		return nil, err
	} else {
		all_records = records
	}

	return all_records, nil

}

func (a *App) Leaderboard_GetNonMaterialized() ([]database.ContributorModel, error) {

	// Get a materialized view of the leaderboard
	var leaderboard []database.ContributorModel

	records, err := a.Dbmanager.GetLeaderboard()
	if err != nil {
		return nil, err
	} else {
		leaderboard = records
	}

	return leaderboard, nil

}

func (a *App) Leaderboard_GetMaterialized() ([]database.ContributorModel, error) {

	// Get a materialized view of the leaderboard
	var leaderboard []database.ContributorModel

	records, err := a.Dbmanager.GetLeaderboardMat()
	if err != nil {
		return nil, err
	} else {
		leaderboard = records
	}

	return leaderboard, nil

}

func (a *App) Leaderboard_GetUserRecords(user string) ([]database.ContributorRecordModel, error) {
	// Take a user's username and return their records

	// Get all the time series data present so far
	// from the database
	var all_records []database.ContributorRecordModel

	// Use the database method
	records, err := a.Dbmanager.GetUserRecords(user)
	if err != nil {
		return nil, err
	} else {
		all_records = records
	}

	return all_records, nil
}
