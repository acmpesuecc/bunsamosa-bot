package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Event struct {
	EventID       string    `json:"event_id"`
	Message       string    `json:"message"`
	TimeInitiated time.Time `json:"time_initiated"`
}

type TimeoutEvent struct {
	EventID     string `json:"event_id"`
	TimeoutSecs int    `json:"timeout_seconds"`
	Emit        string `json:"emit"`
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
	Message string `json:"message"`
}

type RemainingEvent struct {
	EventID string `json:"event_id"`
}

type RemainingResponse struct {
	EventID       string `json:"event_id"`
	TimeRemaining string `json:"time_remaining"`
}

func PingHandler(response http.ResponseWriter, request *http.Request) {
	log.Println("[PING] Received Ping request!")
	response.Write([]byte("Pong UwU"))
	// response.WriteHeader(http.StatusOK)
}

func TimerHandler(response http.ResponseWriter, request *http.Request) {
	log.Println("[TIMER] Received Timer request!")

	var event Event
	err := json.NewDecoder(request.Body).Decode(&event)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)
		return
	}

	//  time is sent as string ig ?
	// [GOD]: Yes
	event.TimeInitiated, err = time.Parse(time.RFC3339, event.TimeInitiated.String())
	if err != nil {
		http.Error(response, "Invalid time format", http.StatusBadRequest)
		return
	}

	log.Printf("Event received: %+v\n", event)

	// Now we handle as needed
	// maybe call a deassin heree?
	// [GOD]: Yes my child
}
