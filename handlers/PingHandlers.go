package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/acmpesuecc/bunsamosa-bot/globals"
	"github.com/google/go-github/v74/github"
)

type Event struct {
	EventID       string    `json:"event_id"`
	Message       string    `json:"message"`
	TimeInitiated time.Time `json:"time_initiated"`
}

// Input event type
type TimeoutEvent struct {
	EventID     string `json:"event_id"`
	TimeoutSecs int    `json:"timeout_seconds"`
	Emit        string `json:"emit"`
}

// Input event type
type TimeoutResponse struct {
	EventID string `json:"event_id"`
	Message string `json:"message"`
}

// This is the response that's sent to the webhook
type TimeoutMessage struct {
	EventID       string `json:"event_id"`
	Message       string `json:"message"`
	TimeInitiated string `json:"time_initiated"`
}

// Input cancel event struct
type CancelEvent struct {
	EventID string `json:"event_id"`
}

type CancelResponse struct {
	EventID string `json:"event_id"`
	Message string `json:"message"`
}

type RemainingEvent struct {
	EventID string `json:"event_id"`
}

type RemainingResponse struct {
	EventID       string `json:"event_id"`
	TimeRemaining string `json:"time_remaining"`
	Message       string `json:"message"`
}

type ExtendEvent struct {
	EventID     string `json:"event_id"`
	TimeoutSecs int    `json:"timeout_seconds"`
}

type ExtendResponse struct {
	EventID string `json:"event_id"`
	Message string `json:"message"`
}

func PingHandler(response http.ResponseWriter, request *http.Request) {
	globals.AppState.ZeroLogger.Info().Str("scope", "PING").Msg("Received Ping request!")
	response.Write([]byte("Pong UwU"))
	// response.WriteHeader(http.StatusOK)
}

func TimerHandler(response http.ResponseWriter, request *http.Request) {
	globals.AppState.ZeroLogger.Info().Str("scope", "TIMER_DAEMON").Msg("Received Timer request!")

	var timeoutMessage TimeoutMessage
	err := json.NewDecoder(request.Body).Decode(&timeoutMessage)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	//  time is sent as string ig ?
	// [GOD]: Yes

	globals.AppState.ZeroLogger.Info().Str("scope", "TIMER_DAEMON").Any("timeoutMessage", timeoutMessage).Msg("Event received")

	// Now we handle as needed
	// maybe call a deassin heree?
	// [GOD]: Yes my child

	contributorHandle := timeoutMessage.EventID

	var emitInterface struct {
		Owner     string
		Commenter string
		Repo      string
		Number    int64
	}

	err = json.Unmarshal([]byte(timeoutMessage.Message), &emitInterface)
	if err != nil {
		globals.AppState.ZeroLogger.Err(err).Str("scope", "TIMER_DAEMON").Msg(" Failed to unmarshal timeoutMessage.Message")
		return
	}

	commentBody := fmt.Sprintf("Hey @%s! The timer for the %s to work on the issue has finished, deassign and assign a new contributor or extend the current timer. Contact maintainer leads if inactive @DedLad @polarhive @achyuthcodes30",
		emitInterface.Commenter, contributorHandle)
	comment := github.IssueComment{Body: &commentBody}
	_, _, err = globals.AppState.RuntimeClient.Issues.CreateComment(
		context.TODO(),
		emitInterface.Owner,
		emitInterface.Repo,
		int(emitInterface.Number),
		&comment,
	)

	if err != nil {
		globals.AppState.ZeroLogger.Err(err).Str("scope", "TIMER_DAEMON").
			Str("repoName", emitInterface.Repo).
			Int64("issueNum", emitInterface.Number).
			Msg("Could not Comment on Issue")
	} else {
		globals.AppState.ZeroLogger.Info().Str("scope", "TIMER_DAEMON").
			Str("repoName", emitInterface.Repo).
			Int64("issueNum", emitInterface.Number).
			Msg("Successfully Commented on Issue")
	}
}
