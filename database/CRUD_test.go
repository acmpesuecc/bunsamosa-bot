package database

import (
	"os"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapProductionLogger() *zap.SugaredLogger {
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
			zapcore.AddSync(os.Stdout),            // Sync to stdout
			c,                                     // Use the same level as the original core
		)
	}))
	sugar := customLogger.Sugar()
	sugar.Infof("Initialized Test Logger")
	return sugar
}

func TestIssueStorage(t *testing.T) {

	logger := NewZapProductionLogger()

	dbManager := DBManager{}

	if err := os.Remove("../tests/dev.db"); err != nil {
		t.Errorf("Failed to remove test db")
	}
	dbManager.Init("../tests/dev.db", logger)

	dbManager.db.Create(&Repo{URL: "repo 1"})
	dbManager.db.Create(&Repo{URL: "repo 2"})
	dbManager.db.Create(&Repo{URL: "repo 2"})

	t.Run("assign issue to contributors", func(t *testing.T) {
		got_bool, got_err := dbManager.AssignIssue("issue 1", "c 1", "repo 1")
		want_bool := true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 2", "c 2", "repo 2")
		want_bool = true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 3", "c 3", "repo 2")
		want_bool = true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 1", "c 2", "repo 1")
		want_bool = false
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

	t.Run("withdraw by a contributor from an issue", func(t *testing.T) {
		got_bool, got_err := dbManager.WithdrawIssue("issue 3", "c 3")
		want_bool := true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

	t.Run("contributor withdrawing from an issue not assigned to them", func(t *testing.T) {
		got_bool, got_err := dbManager.WithdrawIssue("issue 2", "c 1")
		want_bool := false
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

}
