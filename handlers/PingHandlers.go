package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anirudhRowjee/bunsamosa-bot/globals"
	v3 "github.com/google/go-github/v47/github"
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
	log.Println("[PING] Received Ping request!")
	response.Write([]byte("Pong UwU"))
	// response.WriteHeader(http.StatusOK)
}

func TimerHandler(response http.ResponseWriter, request *http.Request) {
	log.Println("[TIMER] Received Timer request!")

	var timeoutMessage TimeoutMessage
	err := json.NewDecoder(request.Body).Decode(&timeoutMessage)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	//  time is sent as string ig ?
	// [GOD]: Yes

	log.Printf("Event received: %+v\n", timeoutMessage)

	// Now we handle as needed
	// maybe call a deassin heree?
	// [GOD]: Yes my child

	contributorHandle := timeoutMessage.EventID

	var emitInterface struct {
		owner  string
		repo   string
		number int64
	}

	err = json.Unmarshal([]byte(timeoutMessage.Message), &emitInterface)
	if err != nil {
		log.Println("[ERROR] Failed to unmarshal timeoutMessage.Message")
		return
	}

	commentBody := fmt.Sprintf("Hey @%s! The timer for the @%s to work on the issue has finished, deassign and assing a new contributor or extend the current timer", emitInterface.owner, contributorHandle)
	comment := v3.IssueComment{Body: &commentBody}
	_, _, err = globals.Myapp.RuntimeClient.Issues.CreateComment(
		context.TODO(),
		emitInterface.owner,
		emitInterface.repo,
		int(emitInterface.number),
		&comment,
	)

	if err != nil {
		log.Printf("[ERROR] Could not Comment on Issue -> Repository [%s] Issue (#%d)\n", emitInterface.repo, emitInterface.number)
	} else {
	 	log.Printf("[ISSUEHANDLER] Successfully Commented on Issue -> Repository [%s] Issue (#%d)\n", emitInterface.repo, emitInterface.number)
	}
}
