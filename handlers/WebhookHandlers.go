package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/anirudhRowjee/bunsamosa-bot/globals"
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
		globals.AppState.ZeroLogger.Err(err).Str("scope", "ISSUE_HANDLER").Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
	} else {
		globals.AppState.ZeroLogger.Info().Str("scope", "ISSUE_HANDLER").Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
	}
}

func newIssueCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {
	globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN")).Msgf("Received new comment on Repository [%s] Issue (#%d)[%s] Comment: %s\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title, parsedHook.Comment.Body)

	// MAINTAINER:  !assgin @handle MINS/default: 45
	// MAINTAINER:  !deassign
	// CONTRIBUTOR: !withdraw

	issue_comment := parsedHook.Comment.Body
	commentCommand := getCommand(issue_comment, globals.AppState.ZeroLogger)

	isMaintainer, err := globals.AppState.DBManager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
	if err != nil {
		globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("CHECK_MAINTAINER")).Msg("Could not check is_maintainer")
		return
	}

	globals.AppState.ZeroLogger.Info().Str("scope", "ISSUE_COMMENT_HANDLER").Str("commentCommand", commentCommand)
	if strings.Contains(commentCommand, "!assign") && isMaintainer {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN")).Msg("Recieved an !assign request")
		contributorHandle, time, success := parseAssign(commentCommand, globals.AppState.ZeroLogger)
		if success {
			// CRUD op called to check assign status of contrib
			isAssigned, assignedIssueURL, err := globals.AppState.DBManager.CheckUserAssigned(contributorHandle)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Str("user", contributorHandle).Msg("Failed to check if user is already assigned")
				return
			}

			if isAssigned {
				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Str("user", contributorHandle).Str("assignedIssueURL", assignedIssueURL).Msg("User is already assigned to another issue")
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
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msg("Failed to comment on issue")
					return
				}
				return
			}
			db_success, err := globals.AppState.DBManager.AssignIssue(
				parsedHook.Issue.URL,
				contributorHandle,
				parsedHook.Repository.Name,
			)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Str("user", contributorHandle).Msgf("Failed to assign issue to %q", contributorHandle)
				return
			}

			if db_success {
				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("Attempting to add assignee to Github Issue via Client, Repo owner: %s, Repo name: %s, Issue number: %d, Assignees: %v", parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, parsedHook.Issue.Number, []string{contributorHandle[1:]})
				_, _, err = globals.AppState.RuntimeClient.Issues.AddAssignees(
					context.TODO(),
					parsedHook.Repository.Owner.Login,
					parsedHook.Repository.Name,
					int(parsedHook.Issue.Number),
					[]string{contributorHandle[1:]},
				)
				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("GH_API")).Str("user", contributorHandle).Msgf("Failed to assign issue to %+v. Unable to use Github RuntimeClient", contributorHandle)
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
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).Msgf("Failed to marshal message for saturn!")
					return
				}

				request := TimeoutEvent{
					EventID:     contributorHandle,
					TimeoutSecs: time * 60, // in minutes
					Emit:        string(emitJson),
				}

				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).Msgf("Sending request %+v to Saturn Timer Daemon", request)

				requestBytes, err := json.Marshal(request)
				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("Failed to assign issue to %q. Failed to marshal bytes for request to Timer-Daemon", contributorHandle)
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
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).Str("contributorHandle", contributorHandle).Msgf("Failed to send /register request to TimerDaemon for event_id %s", contributorHandle)
					return
				}

				if response == nil {
					globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).Msg("No response from the timer service")

					response := "Failed to assign issue. Failed to allot a timer for the contributor"
					comment := github.IssueComment{Body: &response}

					_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

					if err != nil {
						globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
					} else {
						globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
					}

					return
				}

				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE").Str("TIMER_DAEMON")).Msgf("POST /register event_id %s response STATUS %d", contributorHandle, response.StatusCode)

			} else {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("db fail, Failed to assign issue to %+v", contributorHandle)
			}
		} else {
			globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("ASSIGN_ISSUE")).Msgf("Failed to parse issue")
		}
	} else if strings.Contains(commentCommand, "!deassign") && isMaintainer {
		globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN")).Msg("Recieved an !assign request")
		dbSuccess, err := globals.AppState.DBManager.DeassignIssue(
			parsedHook.Issue.URL,
		)
		if err != nil {
			globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msg("Failed to deassign issue")
		}
		if dbSuccess {
			if parsedHook.Issue.Assignee == nil {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("GH_API")).Msg("Failed to deassign issue, no existing assignees")
				return
			}
			_, _, err := globals.AppState.RuntimeClient.Issues.RemoveAssignees(
				context.TODO(),
				parsedHook.Repository.Owner.Login,
				parsedHook.Repository.Name,
				int(parsedHook.Issue.Number),
				[]string{parsedHook.Issue.Assignee.Login},
			)
			globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Attempting to deassign assignee from Github Issue via Client, Repo owner: %s, Repo name: %s, Issue number: %d, Assignees: %v", parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, parsedHook.Issue.Number, parsedHook.Issue.Assignee.Login)

			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("GH_API")).Msgf("Failed to deassign issue from %s. Unable to use Github Runtime Client", parsedHook.Issue.Assignee.Login)
			}

			cancelRequest := CancelEvent{
				EventID: "@" + parsedHook.Issue.Assignee.Login,
			}

			cancelRequestBytes, err := json.Marshal(cancelRequest)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Failed to deassign issue from %s. Failed to marshal bytes for request to Timer-Daemon", parsedHook.Issue.Assignee.Login)
			}

			response, err := http.Post(
				TimerDaemonURL+"/cancel",
				"application/json",
				bytes.NewReader(cancelRequestBytes),
			)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Failed to send /cancel request to TimerDaemon for event_id %s", parsedHook.Issue.Assignee.Login)
			}

			if response == nil {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE").Str("TIMER_DAEMON")).Msg("No response from the timer service")

				response := "Failed to deassign issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
				comment := github.IssueComment{Body: &response}

				_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				} else {
					globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				}
				return
			}

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Failed to read response bytes from Timer Daemon for POST /cancel request event_id %s", parsedHook.Issue.Assignee.Login)
			}

			var cancelResponse CancelResponse
			err = json.Unmarshal(responseBytes, &cancelResponse)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request event_id %s", parsedHook.Issue.Assignee.Login)
			}

			if response.StatusCode != http.StatusOK {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("POST /cancel event_id %s response STATUS %d MSG %s", parsedHook.Issue.Assignee.Login, response.StatusCode, cancelResponse.Message)
			} else {
				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("POST /cancel event_id %s response STATUS %d MSG %s", parsedHook.Issue.Assignee.Login, response.StatusCode, cancelResponse.Message)
			}

		} else {
			globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("DEASSIGN_ISSUE")).Msgf("Failed to deassign issue for comment made by %s on issue %s", parsedHook.Sender.Login, parsedHook.Issue.URL)
		}

	} else if strings.Contains(commentCommand, "!withdraw") {
		// todo
		// first query db and check
		contributorHandle := parsedHook.Sender.Login

		db_success, err := globals.AppState.DBManager.WithdrawIssue(
			parsedHook.Issue.URL,
			"@"+contributorHandle,
		)
		if err != nil {
			if parsedHook.Issue.Assignee == nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to withdraw issue to %+q", parsedHook.Sender.Login)
			} else {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to withdraw issue to %+q", parsedHook.Issue.Assignee.Login)
			}
		}

		if db_success {
			_, _, err := globals.AppState.RuntimeClient.Issues.RemoveAssignees(
				context.TODO(),
				parsedHook.Repository.Owner.Login,
				parsedHook.Repository.Name,
				int(parsedHook.Issue.Number),
				[]string{parsedHook.Issue.Assignee.Login},
			)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE").Str("GH_API")).Msgf("Failed to withdraw issue to %+v. Unable to use Github RuntimeClient", parsedHook.Sender.Login)
			}

			cancelledRequest := CancelEvent{
				EventID: "@" + parsedHook.Issue.Assignee.Login,
			}

			cancelled_request_bytes, err := json.Marshal(cancelledRequest)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to withdraw issue to %q. Failed to marshal bytes for request to Timer-Daemon", parsedHook.Sender.Login)
			}

			response, err := http.Post(
				TimerDaemonURL+"/cancel",
				"application/json",
				bytes.NewReader(cancelled_request_bytes),
			)

			if response == nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE").Str("TIMER_DAEMON")).Msg("No response from the timer service")

				response := "Failed to withdraw issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
				comment := github.IssueComment{Body: &response}

				_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				} else {
					globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				}
				return
			}

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to read response bytes from Timer Daemon for POST /cancel request event_id %s", parsedHook.Issue.Assignee.Login)
			}

			var cancelResponse CancelResponse
			err = json.Unmarshal(responseBytes, &cancelResponse)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request event_id %s", parsedHook.Issue.Assignee.Login)
			}

			if response.StatusCode != http.StatusOK {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("POST /cancel event_id %s response STATUS %d MSG %s", parsedHook.Issue.Assignee.Login, response.StatusCode, cancelResponse.Message)
			} else {
				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("POST /cancel event_id %s response STATUS %d MSG %s", parsedHook.Issue.Assignee.Login, response.StatusCode, cancelResponse.Message)
			}

		} else {
			globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("WITHDRAW_ISSUE")).Msgf("Failed to withdraw issue for comment made by %s on issue %s", parsedHook.Sender.Login, parsedHook.Issue.URL)
		}

	} else if strings.Contains(commentCommand, "!extend") && isMaintainer {
		extraTime, success := parseExtend(commentCommand)

		if success {
			if parsedHook.Issue.Assignee == nil {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("No Assignee for issue %q extend sent by sender %q", parsedHook.Issue.URL, parsedHook.Sender.Login)
				return
			}

			currentContributorHandle := parsedHook.Issue.Assignee.Login

			extendEventBytes, err := json.Marshal(&ExtendEvent{
				EventID:     "@" + currentContributorHandle,
				TimeoutSecs: extraTime,
			})
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Failed to marshal bytes for request to Timer-Daemon %s", parsedHook.Sender.Login)
			}

			response, err := http.Post(TimerDaemonURL+"/extend", "application/json", bytes.NewReader(extendEventBytes))
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Failed to send /extend request to TimerDaemon for event_id %s", currentContributorHandle)
			}

			if response == nil {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE").Str("TIMER_DAEMON")).Msg("No response from the timer service")

				response := "Failed to extend issue. Failed to allot a timer for the contributor. Contact @bwaklog @anirudhsudhir"
				comment := github.IssueComment{Body: &response}

				_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				} else {
					globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				}
				return
			}

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Failed to read response bytes from Timer Daemon for POST /extend request event_id %s", currentContributorHandle)
			}

			var extendEventResponse ExtendResponse
			err = json.Unmarshal(responseBytes, &extendEventResponse)
			if err != nil {
				globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Failed to unmarshal response bytes from Timer Daemon for POST /extend request event_id %s", currentContributorHandle)
			}

			if response.StatusCode != http.StatusOK {
				globals.AppState.ZeroLogger.Error().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("POST /extend event_id %s response STATUS %d MSG %s", currentContributorHandle, response.StatusCode, extendEventResponse.Message)
			} else {
				globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("POST /extend event_id %s response STATUS %d MSG %s", currentContributorHandle, response.StatusCode, extendEventResponse.Message)

				extendResp := fmt.Sprintf("Extended timer by %d", extraTime)
				comment := github.IssueComment{Body: &extendResp}

				_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

				if err != nil {
					globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				} else {
					globals.AppState.ZeroLogger.Info().Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				}

			}
		} else {
			globals.AppState.ZeroLogger.Err(err).Array("scope", zerolog.Arr().Str("ISSUE_COMMENT_HANDLER").Str("EXTEND_ISSUE")).Msgf("Failed to extend issue for comment made by %s on issue %s", parsedHook.Sender.Login, parsedHook.Issue.URL)
		}

	} else {
		// Invalid command
		globals.AppState.ZeroLogger.Error().Str("scope", "ISSUE_COMMENT_HANDLER").Msg("Invalid bot command")
	}
}

func newPRHandler(parsed_hook *ghwebhooks.PullRequestPayload) {
	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you from Opening this Pull Request, @" + parsed_hook.Sender.Login + " ! A Maintainer will review it soon!"
	comment := github.IssueComment{Body: &response}

	_, _, err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsed_hook.Repository.Owner.Login, parsed_hook.Repository.Name, int(parsed_hook.PullRequest.Number), &comment)

	if err != nil {
		globals.AppState.ZeroLogger.Err(err).Str("scope", "NEW_PR_HANDLER").Msgf("Could not Comment on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.PullRequest.Number, parsed_hook.PullRequest.Title)
	} else {
		globals.AppState.ZeroLogger.Info().Str("scope", "NEW_PR_HANDLER").Msgf("Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.PullRequest.Number, parsed_hook.PullRequest.Title)
	}
}

func newPRCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {
	// Parse the current webhook

	is_maintainer, err := globals.AppState.DBManager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
	if err != nil {
		log.Println("[ERROR][BOUNTY] Could not check is_maintainer ->", err)
		return
	}

	if is_maintainer {
		log.Println("A Maintainer Commented -> ")
		log.Printf("[PR_COMMENTHANDLER] Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)

		// parse the comment here to give a bounty
		bounty, valid := parseBountyPoints(parsedHook.Comment.Body)

		if valid {

			// Assign the bounty points
			err := globals.AppState.DBManager.AssignBounty(
				parsedHook.Sender.Login,
				parsedHook.Issue.User.Login,
				parsedHook.Issue.PullRequest.HTMLURL,
				bounty,
			)
			if err != nil {
				log.Println("[ERROR][BOUNTY] Could not assign bounty points ->", err)
				return
			}

			log.Printf("[PR_COMMENTHANDLER] Successfully Assigned Bounty on Pull Request -> Repository [%s] PR (#%d)[%s] to user %s for %d points\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title, parsedHook.Issue.User.Login, bounty)

			response := "Assigned " + fmt.Sprint(bounty) + " Bounty points to user @" + parsedHook.Issue.User.Login + " !"
			comment := github.IssueComment{Body: &response}

			_, _, new_err := globals.AppState.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)
			if new_err != nil {
				log.Printf("[ERROR] Could not Comment on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
				log.Println("Error ->", new_err)
			} else {
				log.Printf("[PRHANDLER] Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
			}

		}

	} else {
		log.Printf("[WARN] Someone else commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
	}
	// Return error
}

func WebhookHandler(response http.ResponseWriter, request *http.Request) {
	// Creating hook parsers :
	hook_secret := ghwebhooks.Options.Secret(globals.AppState.WebhookSecret)
	hook_parser, err := ghwebhooks.New(hook_secret)
	if err != nil {
		log.Println("[ERROR] Webhook parser creation Failed")
		panic(err)
	}

	log.Println("Recieved webhook event")

	// Listing all actions/Events to be parsed :
	NeededEvents := []ghwebhooks.Event{
		ghwebhooks.IssueCommentEvent,      // STATUS: Not handled
		ghwebhooks.IssuesEvent,            // STATUS: Handled
		ghwebhooks.PullRequestEvent,       // STATUS: Not handled
		ghwebhooks.PullRequestReviewEvent, // STATUS: POTENTIALLY WILL NOT HANDLE
		ghwebhooks.PingEvent,              // STATUS: Not Handled
		ghwebhooks.PublicEvent,            // STATUS: WILL NOT HANDLE
	}

	parsed_hook, err := hook_parser.Parse(request, NeededEvents...)
	if err != nil {

		log.Println(parsed_hook)
		switch err {
		case ghwebhooks.ErrEventNotFound:
			log.Println("[WARN] Undefined GitHub event received. err :", err)
			response.WriteHeader(http.StatusOK)
			return

		case ghwebhooks.ErrEventNotSpecifiedToParse:
			// FIXME Unsure about this
			log.Println("[WARN] This event hasn't been specified to parse", err)
			response.WriteHeader(http.StatusBadRequest)
			return

		default:
			log.Printf("[ERROR] received malformed GitHub event: %v\n", err)

			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	switch parsed_hook := parsed_hook.(type) {

	// A new issue has been opened.
	case ghwebhooks.IssuesPayload:
		if parsed_hook.Action == "opened" {
			log.Printf("[PAYLOAD] Someone Opened an Issue -> user [%s] Opened an Issue with title [%s] on repository [%s]", parsed_hook.Sender.Login, parsed_hook.Issue.Title, parsed_hook.Repository.FullName)
			go newIssueHandler(&parsed_hook)
		} else {
			log.Printf("[PAYLOAD] Non-Open Issue Event -> user [%s] Did something [%s] On an Issue with title [%s] on repository [%s]", parsed_hook.Sender.Login, parsed_hook.Action, parsed_hook.Issue.Title, parsed_hook.Repository.FullName)
		}

	// The API has been Pinged from Github
	case ghwebhooks.PingPayload:
		log.Println("[PAYLOAD] Ping ->", parsed_hook)

	// Someone has opened a new Pull Request
	case ghwebhooks.PullRequestPayload:

		// TODO Respond with a comment saying congratulations, someone will review your PR soon
		if parsed_hook.Action == "opened" {
			log.Printf("[PAYLOAD] Someone Opened an PR -> user [%s] Opened an Issue with title [%s] on repository [%s]", parsed_hook.Sender.Login, parsed_hook.PullRequest.Title, parsed_hook.Repository.FullName)
			go newPRHandler(&parsed_hook)
			// TODO Add handler to assign bounty points
		} else {
			log.Printf("[PAYLOAD] Non-Open PR Event -> user [%s] Did something [%s] On an PR with title [%s] on repository [%s]", parsed_hook.Sender.Login, parsed_hook.Action, parsed_hook.PullRequest.Title, parsed_hook.Repository.FullName)
		}

	// Someone has commented on an Issue
	// We'll be using this webhook for the following -
	// 		- Assigning Bounty to a user
	// 		- Freezing the Leaderboard
	case ghwebhooks.IssueCommentPayload:

		log.Printf("[PAYLOAD] Someone Commented on an issue -> user [%s] commented [%s] on repository [%s]", parsed_hook.Sender.Login, parsed_hook.Comment.Body, parsed_hook.Repository.FullName)

		// Step 1 -> Validate, make sure the issuecomment is on a PR and not on an issue,
		if (parsed_hook.Issue.PullRequest != nil) && isPullRequest(parsed_hook.Issue.PullRequest.URL) && parsed_hook.Action == "created" {
			go newPRCommentHandler(&parsed_hook)
		} else if (parsed_hook.Issue.PullRequest == nil) && parsed_hook.Action == "created" {
			go newIssueCommentHandler(&parsed_hook)
		}

	// The Repository has been made public
	// TODO Consider if we really need this
	case ghwebhooks.PublicPayload:
		log.Println("[PAYLOAD] Some Public Event ->", parsed_hook)

	default:
		log.Println("[WARN] missing handler")
	}

	log.Println("[PAYLOAD] Webhook Has been Handled!")
	response.WriteHeader(http.StatusOK)
}
