package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/acmpesuecc/bunsamosa-bot/globals"
	ghwebhooks "github.com/go-playground/webhooks/v6/github"
	"github.com/google/go-github/v74/github"
	"github.com/rs/zerolog"
)

// handlers global constants
var TimerDaemonURL string

// Setting logger in main.go
// var SugaredLogger *zap.SugaredLogger

type HandlerState struct{}

type EmitMessageFormat struct {
	Owner     string
	Commenter string
	Repo      string
	Number    int64
}

func newIssueHandler(parsedHook *ghwebhooks.IssuesPayload) {
	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you for opening this issue! A Maintainer will review it soon!"
	comment := github.IssueComment{Body: &response}

	_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Str("scope", "ISSUE_HANDLER").
			Str("repoName", parsedHook.Repository.FullName).
			Int64("issueNum", parsedHook.Issue.Number).
			Str("issueTitle", parsedHook.Issue.Title).
			Msg("Could not Comment on Issue")
	} else {
		globals.AppState.ZeroLogger.Info().
			Str("scope", "ISSUE_HANDLER").
			Str("repoName", parsedHook.Repository.FullName).
			Int64("issueNum", parsedHook.Issue.Number).
			Str("issueTitle", parsedHook.Issue.Title).
			Msg("Successfully Commented on Issue")
	}
}

func newIssueCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {
	globals.AppState.ZeroLogger.Info().
		Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN")).
		Str("repoName", parsedHook.Repository.FullName).
		Int64("issueNum", parsedHook.Issue.Number).
		Str("issueTitle", parsedHook.Issue.Title).
		Str("commentBody", parsedHook.Comment.Body).
		Msg("Received new comment")

	// MAINTAINER:  !assgin @handle MINS/default: 45
	// MAINTAINER:  !deassign
	// CONTRIBUTOR: !withdraw

	issue_comment := parsedHook.Comment.Body
	commentCommand := getCommand(issue_comment, globals.AppState.ZeroLogger)

	isMaintainer, err := globals.AppState.DBManager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("CHECK_MAINTAINER")).
			Msg("Could not check isMaintainer")
		return
	}

	globals.AppState.ZeroLogger.Info().
		Str("scope", "ISSUE_COMMENT_HANDLER").Str("commentCommand", commentCommand)
	if strings.Contains(commentCommand, "!assign") && isMaintainer {
		assignIssue(commentCommand, parsedHook)
	} else if strings.Contains(commentCommand, "!deassign") && isMaintainer {
		deassignIssue(parsedHook)
	} else if strings.Contains(commentCommand, "!withdraw") {
		// todo
		// first query db and check
		withdrawIssue(parsedHook)
	} else if strings.Contains(commentCommand, "!extend") && isMaintainer {
		extendIssue(commentCommand, parsedHook)
	} else {
		// Invalid command
		globals.AppState.ZeroLogger.Error().Str("scope", "ISSUE_COMMENT_HANDLER").Msg("Invalid bot command")
	}
}

func assignIssue(commentCommand string, parsedHook *ghwebhooks.IssueCommentPayload) {
	globals.AppState.ZeroLogger.Info().
		Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN")).
		Msg("Recieved an !assign request")
	contributorHandle, time, success := parseAssign(commentCommand, globals.AppState.ZeroLogger)
	if !success {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Msg("Failed to parse assign")
		return
	}
	// CRUD op called to check assign status of contrib
	isAssigned, assignedIssueURL, err := globals.AppState.DBManager.CheckUserAssigned(contributorHandle)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("user", contributorHandle).Msg("Failed to check if user is already assigned")
		return
	}

	if isAssigned {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("user", contributorHandle).
			Str("assignedIssueURL", assignedIssueURL).
			Msg("User is already assigned to another issue")
		response := fmt.Sprintf("User %s is already assigned to another issue: %s", contributorHandle, assignedIssueURL)
		comment := github.IssueComment{Body: &response}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(
			context.TODO(),
			parsedHook.Repository.Owner.Login,
			parsedHook.Repository.Name,
			int(parsedHook.Issue.Number),
			&comment,
		)
		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
				Msg("Failed to comment on issue")
			return
		}
		return
	}
	dbSuccess, err := globals.AppState.DBManager.AssignIssue(
		parsedHook.Issue.URL,
		contributorHandle,
		parsedHook.Repository.Name,
	)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("contributor", contributorHandle).
			Msg("Failed to assign issue")
		return
	}

	if !dbSuccess {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("contributor", contributorHandle).
			Msg("Failed to assign issue, DB error")
		// return
	}
	globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
		Str("repoOwner", parsedHook.Repository.Owner.Login).
		Str("repoName", parsedHook.Repository.Name).
		Int64("issueNum", parsedHook.Issue.Number).
		Str("assignee", contributorHandle).
		Msg("Attempting to add assignee using GitHub RuntimeClient")
	_, _, err = globals.AppState.RuntimeClient.Issues.AddAssignees(
		context.TODO(),
		parsedHook.Repository.Owner.Login,
		parsedHook.Repository.Name,
		int(parsedHook.Issue.Number),
		[]string{contributorHandle[1:]},
	)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("GH_API")).
			Str("user", contributorHandle).
			Msg("Failed to assign issue, unable to use Github RuntimeClient")
		return
	}

	emitInterface := EmitMessageFormat{
		Owner:     parsedHook.Repository.Owner.Login,
		Commenter: parsedHook.Sender.Login,
		Repo:      parsedHook.Repository.Name,
		Number:    parsedHook.Issue.Number,
	}

	emitJson, err := json.Marshal(emitInterface)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).
			Str("sender", contributorHandle).
			Msg("Failed to assign issue. Failed to marshal bytes for request to TimerDaemon")
		return
	}

	request := TimeoutEvent{
		EventID:     contributorHandle,
		TimeoutSecs: time * 60, // in minutes
		Emit:        string(emitJson),
	}

	globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).
		Any("request", request).
		Msg("Sending request to Timer Daemon")

	requestBytes, err := json.Marshal(request)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("sender", contributorHandle).
			Msg("Failed to assign issue. Failed to marshal bytes for request to TimerDaemon")
		return
	}

	// NOTE:
	// Sending a POST request to the Timer Daemon to emit
	// after "time" _seconds_
	//
	response, err := http.Post(
		TimerDaemonURL+"/register",
		"application/json",
		bytes.NewReader(requestBytes),
	)

	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).
			Str("contributor", contributorHandle).
			Msg("Failed to send /register request to TimerDaemon")
		return
	}

	if response == nil {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).
			Msg("No response from the timer service")

		response := "Failed to assign issue. Failed to allot a timer for the contributor"
		comment := github.IssueComment{Body: &response}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Could not Comment on Issue")
		} else {
			globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Successfully Commented on Issue")
		}

		return
	}

	if response.StatusCode != http.StatusOK {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("contributor", contributorHandle).
			Int("statusCode", response.StatusCode).
			Msg("POST /register recieved")
	} else {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).
			Str("contributor", contributorHandle).
			Int("statusCode", response.StatusCode).
			Msg("POST /register recieved")
	}
}

func deassignIssue(parsedHook *ghwebhooks.IssueCommentPayload) {
	globals.AppState.ZeroLogger.Info().
		Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN")).
		Msg("Recieved an !deassign request")

	dbSuccess, err := globals.AppState.DBManager.DeassignIssue(parsedHook.Issue.URL)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Msg("Failed to deassign issue")
		return
	}

	if !dbSuccess {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("sender", parsedHook.Sender.Login).
			Str("issueURL", parsedHook.Issue.URL).
			Msg("Failed to deassign issue, DB error")
		return
	}

	if parsedHook.Issue.Assignee == nil {
		globals.AppState.ZeroLogger.Error().
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("GH_API")).
			Msg("Failed to deassign issue, no existing assignees")
		return
	}
	_, _, err = globals.AppState.RuntimeClient.Issues.RemoveAssignees(
		context.TODO(),
		parsedHook.Repository.Owner.Login,
		parsedHook.Repository.Name,
		int(parsedHook.Issue.Number),
		[]string{parsedHook.Issue.Assignee.Login},
	)
	globals.AppState.ZeroLogger.Info().
		Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
		Str("repoOwner", parsedHook.Repository.Owner.Login).
		Str("repoName", parsedHook.Repository.Name).
		Int64("issueNum", parsedHook.Issue.Number).
		Str("assignee", parsedHook.Issue.Assignee.Login).
		Msg("Attempting to deassign assignee")

	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("GH_API")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Msg("Failed to deassign issue, unable to use GitHub RuntimeClient")
		return
	}

	cancelRequest := CancelEvent{
		EventID: "@" + parsedHook.Issue.Assignee.Login,
	}

	cancelRequestBytes, err := json.Marshal(cancelRequest)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Msg("Failed to deassign issue. Failed to marshal bytes for request to TimerDaemon")
		return
	}

	response, err := http.Post(
		TimerDaemonURL+"/cancel",
		"application/json",
		bytes.NewReader(cancelRequestBytes),
	)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Msg("Failed to send /cancel request to TimerDaemon")
		return
	}

	if response == nil {
		globals.AppState.ZeroLogger.Error().
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("TIMER_DAEMON")).
			Msg("No response from the timer service")

		response := "Failed to deassign issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
		comment := github.IssueComment{Body: &response}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Could not Comment on Issue")
		} else {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Successfully Commented on Issue")
		}
		return
	}

	var cancelResponse CancelResponse
	err = json.NewDecoder(response.Body).Decode(&cancelResponse)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Msg("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request")
		return
	}

	if response.StatusCode != http.StatusOK {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", cancelResponse.Message).
			Msg("POST /cancel recieved")
	} else {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", cancelResponse.Message).
			Msg("POST /cancel recieved")
	}
}

func withdrawIssue(parsedHook *ghwebhooks.IssueCommentPayload) {
	contributorHandle := parsedHook.Sender.Login

	dbSuccess, err := globals.AppState.DBManager.WithdrawIssue(
		parsedHook.Issue.URL,
		"@"+contributorHandle,
	)
	if err != nil {
		if parsedHook.Issue.Assignee == nil {
			globals.AppState.ZeroLogger.Error().Err(err).
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
				Str("sender", contributorHandle).
				Msg("Failed to withdraw issue, no assignee")
		} else {
			globals.AppState.ZeroLogger.Error().Err(err).
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
				Str("assignee", parsedHook.Issue.Assignee.Login).
				Msg("Failed to withdraw issue")
		}
		return
	}

	if !dbSuccess {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
			Str("sender", contributorHandle).
			Str("issueURL", parsedHook.Issue.URL).
			Msg("Failed to withdraw issue, DB error")
		return
	}
	_, _, err = globals.AppState.RuntimeClient.Issues.RemoveAssignees(
		context.TODO(),
		parsedHook.Repository.Owner.Login,
		parsedHook.Repository.Name,
		int(parsedHook.Issue.Number),
		[]string{parsedHook.Issue.Assignee.Login},
	)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE").Str("GH_API")).
			Str("sender", contributorHandle).
			Msg("Failed to withdraw issue, unable to use Github RuntimeClient")
		return
	}

	cancelledRequest := CancelEvent{
		EventID: "@" + parsedHook.Issue.Assignee.Login,
	}

	cancelledRequestBytes, err := json.Marshal(cancelledRequest)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
			Str("sender", contributorHandle).
			Msg("Failed to withdraw issue. Failed to marshal bytes for request to TimerDaemon")
		return
	}

	response, err := http.Post(
		TimerDaemonURL+"/cancel",
		"application/json",
		bytes.NewReader(cancelledRequestBytes),
	)

	if response == nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE").Str("TIMER_DAEMON")).
			Msg("No response from the timer service")

		response := "Failed to withdraw issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
		comment := github.IssueComment{Body: &response}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Could not Comment on Issue")
		} else {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Successfully Commented on Issue")
		}
		return
	}

	var cancelResponse CancelResponse
	err = json.NewDecoder(response.Body).Decode(&cancelResponse)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Msg("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request")
		return
	}

	if response.StatusCode != http.StatusOK {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", cancelResponse.Message).
			Msg("POST /cancel recieved")
	} else {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", cancelResponse.Message).
			Msg("POST /cancel recieved")
	}

}

func extendIssue(commentCommand string, parsedHook *ghwebhooks.IssueCommentPayload) {
	extraTime, success := parseExtend(commentCommand)

	if !success {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("sender", parsedHook.Sender.Login).
			Str("issueURL", "parsedHook.Issue.URL").
			Msg("Failed to extend issue")
		return
	}
	if parsedHook.Issue.Assignee == nil {
		globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("sender", parsedHook.Sender.Login).
			Str("issueURL", parsedHook.Issue.URL).
			Msg("No Assignee for issue extend request")
		return
	}

	currentContributorHandle := parsedHook.Issue.Assignee.Login

	extendEventBytes, err := json.Marshal(&ExtendEvent{
		EventID:     "@" + currentContributorHandle,
		TimeoutSecs: extraTime,
	})
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("sender", parsedHook.Sender.Login).
			Msg("Failed to marshal bytes for request to TimerDaemon")
		return
	}

	response, err := http.Post(TimerDaemonURL+"/extend", "application/json", bytes.NewReader(extendEventBytes))
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("assignee", currentContributorHandle).
			Msg("Failed to send /extend request to TimerDaemon")
		return
	}

	if response == nil {
		globals.AppState.ZeroLogger.Error().
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE").Str("TIMER_DAEMON")).
			Msg("No response from the timer service")

		response := "Failed to extend issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
		comment := github.IssueComment{Body: &response}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).
				Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Could not Comment on Issue")
		} else {
			globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Successfully Commented on Issue")
		}
		return
	}

	var extendEventResponse ExtendResponse
	err = json.NewDecoder(response.Body).Decode(&extendEventResponse)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("assignee", currentContributorHandle).
			Msg("Failed to unmarshal response bytes from Timer Daemon for POST /extend request")
		return
	}

	if response.StatusCode != http.StatusOK {
		globals.AppState.ZeroLogger.Error().
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", extendEventResponse.Message).
			Msg("POST /extend recieved")
	} else {
		globals.AppState.ZeroLogger.Info().
			Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
			Str("assignee", parsedHook.Issue.Assignee.Login).
			Int("statusCode", response.StatusCode).
			Str("message", extendEventResponse.Message).
			Msg("POST /extend recieved")

		extendResp := fmt.Sprintf("Extended timer by %d", extraTime)
		comment := github.IssueComment{Body: &extendResp}

		_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

		if err != nil {
			globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Could not Comment on Issue")
		} else {
			globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).
				Str("repoName", parsedHook.Repository.FullName).
				Int64("issueNum", parsedHook.Issue.Number).
				Str("issueTitle", parsedHook.Issue.Title).
				Msg("Successfully Commented on Issue")
		}
	}
}

func newPRHandler(parsedHook *ghwebhooks.PullRequestPayload) {
	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you from Opening this Pull Request, @" + parsedHook.Sender.Login + " ! A Maintainer will review it soon!"
	comment := github.IssueComment{Body: &response}

	_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.PullRequest.Number), &comment)

	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Str("scope", "NEW_PR_HANDLER").
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.PullRequest.Number).
			Str("prTitle", parsedHook.PullRequest.Title).
			Msg("Could not Comment on Pull Request")
	} else {
		globals.AppState.ZeroLogger.Info().Str("scope", "NEW_PR_HANDLER").
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.PullRequest.Number).
			Str("prTitle", parsedHook.PullRequest.Title).
			Msg("Successfully Commented on Pull Request")
	}
}

func newPRCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {
	// Parse the current webhook

	isMaintainer, err := globals.AppState.DBManager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("BOUNTY").Str("CHECK_MAINTAINER")).
			Msg("Could not check isMaintainer")
		return
	}

	if !isMaintainer {
		globals.AppState.ZeroLogger.Warn().Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
			Str("repoName", parsedHook.Repository.FullName).
			Int64("issueNum", parsedHook.Issue.Number).
			Str("issueName", parsedHook.Issue.Title).
			Msg("Non-Maintainer Commented on Issue")
		return
	}
	globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
		Str("repoName", parsedHook.Repository.FullName).
		Int64("prNum", parsedHook.Issue.Number).
		Str("prName", parsedHook.Issue.Title).
		Msg("Maintainer Commented on Pull Request")

	// parse the comment here to give a bounty
	bounty, valid := parseBountyPoints(parsedHook.Comment.Body)

	if !valid && strings.Contains(parsedHook.Comment.Body, "!bounty") {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.Issue.Number).
			Str("prName", parsedHook.Issue.Title).
			Msg("Could not assign bounty points, value invalid")
		// TODO: Handle invalid case by writing comment/reacting to comment.
		return
	}
	// Assign the bounty points
	err = globals.AppState.DBManager.AssignBounty(
		parsedHook.Sender.Login,
		parsedHook.Issue.User.Login,
		parsedHook.Issue.PullRequest.HTMLURL,
		bounty,
	)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.Issue.Number).
			Str("prName", parsedHook.Issue.Title).
			Msg("Could not assign bounty points")
		return
	}

	globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
		Str("repoName", parsedHook.Repository.FullName).
		Int64("prNum", parsedHook.Issue.Number).
		Str("prName", parsedHook.Issue.Title).
		Str("contributor", parsedHook.Issue.User.Login).
		Int("bounty", bounty).
		Msg("Successfully Assigned Bounty on Pull Request")

	response := "Assigned " + fmt.Sprint(bounty) + " Bounty points to user @" + parsedHook.Issue.User.Login + " !"
	comment := github.IssueComment{Body: &response}

	_, _, err = globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.Issue.Number).
			Str("prName", parsedHook.Issue.Title).
			Msg("Could not Comment on Pull Request")
	} else {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("BOUNTY").Str("PR_COMMENT_HANDLER")).
			Str("repoName", parsedHook.Repository.FullName).
			Int64("prNum", parsedHook.Issue.Number).
			Str("prName", parsedHook.Issue.Title).
			Msg("Successfully Commented on Pull Request")
	}
	// Return error
}

func WebhookHandler(response http.ResponseWriter, request *http.Request) {
	// Creating hook parsers :
	hookSecret := ghwebhooks.Options.Secret(globals.AppState.WebhookSecret)
	hookParser, err := ghwebhooks.New(hookSecret)
	if err != nil {
		globals.AppState.ZeroLogger.Error().Err(err).
			Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("CREATE_PARSER")).
			Msg("Webhook parser creation Failed")
		panic(err)
	}

	globals.AppState.ZeroLogger.Info().
		Str("scope", "WEBHOOK_PARSER").
		Msg("Recieved webhook event")

	// Listing all actions/Events to be parsed :
	NeededEvents := []ghwebhooks.Event{
		ghwebhooks.IssueCommentEvent,      // STATUS: Not handled
		ghwebhooks.IssuesEvent,            // STATUS: Handled
		ghwebhooks.PullRequestEvent,       // STATUS: Not handled
		ghwebhooks.PullRequestReviewEvent, // STATUS: POTENTIALLY WILL NOT HANDLE
		ghwebhooks.PingEvent,              // STATUS: Not Handled
		ghwebhooks.PublicEvent,            // STATUS: WILL NOT HANDLE
	}

	parsedHook, err := hookParser.Parse(request, NeededEvents...)
	if err != nil {

		// log.Println(parsedHook)
		switch err {
		case ghwebhooks.ErrEventNotFound:
			globals.AppState.ZeroLogger.Warn().Err(err).
				Str("scope", "WEBHOOK_PARSER").
				Msg("Undefined GitHub event received.")
			response.WriteHeader(http.StatusOK)
			return

		case ghwebhooks.ErrEventNotSpecifiedToParse:
			// FIXME Unsure about this
			globals.AppState.ZeroLogger.Warn().Err(err).
				Str("scope", "WEBHOOK_PARSER").
				Msg("Webhook event recieved that hasn't been specified to parse.")
			response.WriteHeader(http.StatusOK)
			response.WriteHeader(http.StatusBadRequest)
			return

		default:
			globals.AppState.ZeroLogger.Error().Err(err).
				Str("scope", "WEBHOOK_PARSER").
				Msg("Received malformed GitHub event.")
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	switch parsedHook := parsedHook.(type) {

	// A new issue has been opened.
	case ghwebhooks.IssuesPayload:
		if parsedHook.Action == "opened" {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
				Str("user", parsedHook.Sender.Login).
				Str("issueTitle", parsedHook.Issue.Title).
				Str("repoName", parsedHook.Repository.FullName).
				Msg("Someone Opened an Issue")
			go newIssueHandler(&parsedHook)
		} else {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
				Str("user", parsedHook.Sender.Login).
				Str("action", parsedHook.Action).
				Str("issueTitle", parsedHook.Issue.Title).
				Str("repoName", parsedHook.Repository.FullName).
				Msg("Someone did an Action on an Issue")
		}

	// The API has been Pinged from Github
	case ghwebhooks.PingPayload:
		globals.AppState.ZeroLogger.Info().
			Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
			Any("pingPayload", parsedHook).
			Msg("API was pinged from GitHub")

	// Someone has opened a new Pull Request
	case ghwebhooks.PullRequestPayload:

		// TODO Respond with a comment saying congratulations, someone will review your PR soon
		if parsedHook.Action == "opened" {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
				Str("user", parsedHook.Sender.Login).
				Str("issueTitle", parsedHook.PullRequest.Title).
				Str("repoName", parsedHook.Repository.FullName).
				Msg("Someone Opened a PR")
			go newPRHandler(&parsedHook)
			// TODO Add handler to assign bounty points
		} else {
			globals.AppState.ZeroLogger.Info().
				Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
				Str("user", parsedHook.Sender.Login).
				Str("action", parsedHook.Action).
				Str("issueTitle", parsedHook.PullRequest.Title).
				Str("repoName", parsedHook.Repository.FullName).
				Msg("Someone did an Action on a PR")
		}

	// Someone has commented on an Issue
	// We'll be using this webhook for the following -
	// 		- Assigning Bounty to a user
	// 		- Freezing the Leaderboard
	case ghwebhooks.IssueCommentPayload:

		globals.AppState.ZeroLogger.Info().
			Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
			Str("user", parsedHook.Sender.Login).
			Str("commentBody", parsedHook.Comment.Body).
			Str("repoName", parsedHook.Repository.FullName).
			Msg("Someone Commented on an Issue/PR")

		// Step 1 -> Validate, make sure the issuecomment is on a PR and not on an issue,
		if (parsedHook.Issue.PullRequest != nil) && isPullRequest(parsedHook.Issue.PullRequest.URL) && parsedHook.Action == "created" {
			go newPRCommentHandler(&parsedHook)
		} else if (parsedHook.Issue.PullRequest == nil) && parsedHook.Action == "created" {
			go newIssueCommentHandler(&parsedHook)
		}

	// The Repository has been made public
	// TODO Consider if we really need this
	case ghwebhooks.PublicPayload:
		globals.AppState.ZeroLogger.Info().
			Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
			Str("user", parsedHook.Sender.Login).
			Str("repoName", parsedHook.Repository.FullName).
			Msg("Someone made a repo public")

	default:
		globals.AppState.ZeroLogger.Warn().
			Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
			Msg("Missing Webhook Handler")
	}

	globals.AppState.ZeroLogger.Warn().
		Array("scope", zerolog.Arr().Str("WEBHOOK_PARSER").Str("PAYLOAD")).
		Msg("Webhook Has been Handled!")
	response.WriteHeader(http.StatusOK)
}
