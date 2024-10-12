package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	// "github.com/go-playground/webhooks/v6"
	"github.com/anirudhRowjee/bunsamosa-bot/globals"
	ghwebhooks "github.com/go-playground/webhooks/v6/github"
	v3 "github.com/google/go-github/v47/github"
	"go.uber.org/zap"
)

// handlers global constants
var TimerDaemonURL string

// Setting logger in main.go
var SugaredLogger *zap.SugaredLogger

func newIssueHandler(parsedHook *ghwebhooks.IssuesPayload) {

	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you for opening this issue! A Maintainer will review it soon!"
	comment := v3.IssueComment{Body: &response}

	_, _, err := globals.Myapp.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)

	if err != nil {
		log.Printf("[ERROR] Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
	} else {
		log.Printf("[ISSUEHANDLER] Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title)
	}
}

func newIssueCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {

	log.Printf("Received new comment on Repository [%s] Issue (#%d)[%s] Comment: %s\n", parsedHook.Repository.FullName, parsedHook.Issue.Number, parsedHook.Issue.Title, parsedHook.Comment.Body)

	// MAINTAINER:  !assgin @handle MINS/default: 45
	// MAINTAINER:  !deassign
	// CONTRIBUTOR: !withdraw

	issue_comment := parsedHook.Comment.Body

	commentCommand := getCommand(issue_comment)

	isMaintainer, err := globals.Myapp.Dbmanager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
	if err != nil {
		SugaredLogger.Errorw("Could not check is_maintainer ->", err,
			zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "CHECK_MAINTAINER"}),
		)
		return
	}

	if strings.Contains(commentCommand, "assign") && isMaintainer {
		contributorHandle, time, success := parseAssign(commentCommand)
		if success {
			db_success, err := globals.Myapp.Dbmanager.AssignIssue(
				parsedHook.Issue.URL,
				parsedHook.Sender.Login,
				parsedHook.Repository.URL,
			)

			if err != nil {
				SugaredLogger.Errorw("Failed to assign issue to %q",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
				)
			}

			if db_success {
				// http.Post(url string, contentType string, body io.Reader)
				_, _, err = globals.Myapp.RuntimeClient.Issues.AddAssignees(
					context.TODO(),
					parsedHook.Repository.Owner.Login,
					parsedHook.Repository.Name,
					int(parsedHook.Issue.Number),
					[]string{contributorHandle},
				)
				if err != nil {
					SugaredLogger.Errorw("Failed to assign issue to %+v. Unable to use Github RuntimeClient",
						parsedHook.Sender.Login,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "GH_API"}),
					)
				}

				request := TimeoutEvent{
					EventID:     contributorHandle,
					TimeoutSecs: time,
					Emit:        fmt.Sprintf("Assign issue to %s", contributorHandle),
				}
				requestBytes, err := json.Marshal(request)

				if err != nil {
					SugaredLogger.Errorw("Failed to assign issue to %q. Failed to marshal bytes for request to Timer-Daemon",
						parsedHook.Sender.Login,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
					)
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
					SugaredLogger.Errorw("Failed to send /register request to TimerDaemon for event_id %s",
						contributorHandle,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "TIMER_DAEMON"}),
					)
				}

				var responseBytes []byte
				_, err = response.Body.Read(responseBytes)
				if err != nil {
					SugaredLogger.Errorw("Failed to read response bytes from Timer Daemon for POST /register request event_id %s",
						contributorHandle,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "TIMER_DAEMON"}),
					)
				}

				var timeoutResponse TimeoutResponse
				err = json.Unmarshal(responseBytes, &timeoutResponse)
				if err != nil {
					SugaredLogger.Errorw("Failed to unmarshal response bytes from Timer Daemon for POST /register request event_id %s",
						contributorHandle,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "TIMER_DAEMON"}),
					)
				}

				if response.StatusCode != http.StatusOK {
					SugaredLogger.Errorw("POST /register event_id %s response STATUS %d MSG %s",
						contributorHandle,
						response.StatusCode,
						timeoutResponse.Message,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "TIMER_DAEMON"}),
					)
				} else {
					SugaredLogger.Infow("POST /register event_id %s response STATUS %d MSG %s",
						contributorHandle,
						response.StatusCode,
						timeoutResponse.Message,
						zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE", "TIMER_DAEMON"}),
					)
				}

			} else {
				SugaredLogger.Errorw("Failed to assign issue to %+v",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
				)
			}
		} else {
			SugaredLogger.Errorw("Failed to assign issue to %v",
				contributorHandle,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
			)
		}

	} else if strings.Contains(commentCommand, "deassign") && isMaintainer {
		dbSuccess, err := globals.Myapp.Dbmanager.DeassignIssue(
			parsedHook.Issue.URL,
		)
		if err != nil {
			SugaredLogger.Errorw("Failed to deassign issue to %+q",
				parsedHook.Issue.Assignee.Login,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
			)
		}
		if dbSuccess {
			_, _, err := globals.Myapp.RuntimeClient.Issues.RemoveAssignees(
				context.TODO(),
				parsedHook.Repository.Owner.Login,
				parsedHook.Repository.URL,
				int(parsedHook.Issue.Number),
				[]string{parsedHook.Issue.Assignee.Login},
			)
			if err != nil {
				SugaredLogger.Errorw("Failed to deassign issue to %+v. Unable to use Github RuntimeClient",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE", "GH_API"}),
				)
			}

			cancelRequest := CancelEvent{
				EventID: parsedHook.Issue.Assignee.Login,
			}

			cancelRequestBytes, err := json.Marshal(cancelRequest)
			if err != nil {
				SugaredLogger.Errorw("Failed to deassign issue to %q. Failed to marshal bytes for request to Timer-Daemon",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			}

			response, err := http.Post(
				TimerDaemonURL+"/cancel",
				"application/json",
				bytes.NewReader(cancelRequestBytes),
			)
			if err != nil {
				SugaredLogger.Errorw("Failed to send /cancel request to TimerDaemon for event_id %s",
					parsedHook.Issue.Assignee.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			}

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				SugaredLogger.Errorw("Failed to read response bytes from Timer Daemon for POST /cancel request event_id %s",
					parsedHook.Issue.Assignee.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			}

			var cancelResponse CancelResponse
			err = json.Unmarshal(responseBytes, &cancelResponse)
			if err != nil {
				SugaredLogger.Errorw("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request event_id %s",
					parsedHook.Issue.Assignee.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			}

			if response.StatusCode != http.StatusOK {
				SugaredLogger.Errorw("POST /cancel event_id %s response STATUS %d MSG %s",
					parsedHook.Issue.Assignee.Login,
					response.StatusCode,
					cancelResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			} else {
				SugaredLogger.Infow("POST /cancel event_id %s response STATUS %d MSG %s",
					parsedHook.Issue.Assignee.Login,
					response.StatusCode,
					cancelResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
				)
			}

		} else {
			SugaredLogger.Errorw("Failed to deassign issue to %+v",
				parsedHook.Sender.Login,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "DEASSIGN_ISSUE"}),
			)
		}

	} else if strings.Contains(commentCommand, "withdraw") {
		//todo
		// first query db and check
		contributorHandle := parsedHook.Sender.Login

		db_success, err := globals.Myapp.Dbmanager.WithdrawIssue(
			parsedHook.Issue.URL,
			contributorHandle,
		)
		if err != nil {
			SugaredLogger.Errorw("Failed to withdraw issue to %+q",
				parsedHook.Issue.Assignee.Login,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
			)
		}

		if db_success {
			_, _, err := globals.Myapp.RuntimeClient.Issues.RemoveAssignees(
				context.TODO(),
				parsedHook.Repository.Owner.Login,
				parsedHook.Repository.URL,
				int(parsedHook.Issue.Number),
				[]string{parsedHook.Issue.Assignee.Login},
			)
			if err != nil {
				SugaredLogger.Errorw("Failed to withdraw issue to %+v. Unable to use Github RuntimeClient",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE", "GH_API"}),
				)
			}

			cancelledRequest := CancelEvent{
				EventID: parsedHook.Issue.Assignee.Login,
			}

			cancelled_request_bytes, err := json.Marshal(cancelledRequest)
			if err != nil {
				SugaredLogger.Errorw("Failed to withdraw issue to %q. Failed to marshal bytes for request to Timer-Daemon",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
				)
			}

			response, err := http.Post(
				TimerDaemonURL+"/cancel",
				"application/json",
				bytes.NewReader(cancelled_request_bytes),
			)

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				SugaredLogger.Errorw("Failed to read response bytes from Timer Daemon for POST /cancel request event_id %s",
					parsedHook.Issue.Assignee.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
				)
			}

			var cancelResponse CancelResponse
			err = json.Unmarshal(responseBytes, &cancelResponse)
			if err != nil {
				SugaredLogger.Errorw("Failed to unmarshal response bytes from Timer Daemon for POST /cancel request event_id %s",
					parsedHook.Issue.Assignee.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
				)
			}

			if response.StatusCode != http.StatusOK {
				SugaredLogger.Errorw("POST /cancel event_id %s response STATUS %d MSG %s",
					parsedHook.Issue.Assignee.Login,
					response.StatusCode,
					cancelResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
				)
			} else {
				SugaredLogger.Infow("POST /cancel event_id %s response STATUS %d MSG %s",
					parsedHook.Issue.Assignee.Login,
					response.StatusCode,
					cancelResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
				)
			}

		} else {
			SugaredLogger.Errorw("Failed to withdraw issue to %+q",
				parsedHook.Issue.Assignee.Login,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "WITHDRAW_ISSUE"}),
			)
		}

	} else if strings.Contains(commentCommand, "extend") && isMaintainer {
		extraTime, success := parseExtend(commentCommand)

		if success {

			currentContributorHandle := parsedHook.Issue.Assignee.Login

			extendEventBytes, err := json.Marshal(&ExtendEvent{
				EventID:     currentContributorHandle,
				TimeoutSecs: extraTime,
			})

			if err != nil {
				SugaredLogger.Errorw("Failed to marshal bytes for request to Timer-Daemon %s",
					parsedHook.Sender.Login,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			}

			response, err := http.Post(TimerDaemonURL+"/extend", "application/json", bytes.NewReader(extendEventBytes))
			if err != nil {
				SugaredLogger.Errorw("Failed to send /extend request to TimerDaemon for event_id %s",
					currentContributorHandle,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			}

			var responseBytes []byte
			_, err = response.Body.Read(responseBytes)
			if err != nil {
				SugaredLogger.Errorw("Failed to read response bytes from Timer Daemon for POST /extend request event_id %s",
					currentContributorHandle,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			}

			var extendEventResponse ExtendResponse
			err = json.Unmarshal(responseBytes, &extendEventResponse)
			if err != nil {
				SugaredLogger.Errorw("Failed to unmarshal response bytes from Timer Daemon for POST /extend request event_id %s",
					currentContributorHandle,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			}

			if response.StatusCode != http.StatusOK {
				SugaredLogger.Errorw("POST /extend event_id %s response STATUS %d MSG %s",
					currentContributorHandle,
					response.StatusCode,
					extendEventResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			} else {
				SugaredLogger.Infow("POST /extend event_id %s response STATUS %d MSG %s",
					currentContributorHandle,
					response.StatusCode,
					extendEventResponse.Message,
					zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
				)
			}
		} else {
			SugaredLogger.Errorw("Failed to extend issue for %v",
				parsedHook.Sender.Login,
				zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER", "EXTEND_ISSUE"}),
			)
		}

	} else {
		// Invalid command
		SugaredLogger.Errorw("Invalid bot command",
			zap.Strings("scope", []string{"ISSUE_COMMENT_HANDLER"}),
		)
	}
}

func newPRHandler(parsed_hook *ghwebhooks.PullRequestPayload) {

	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you from Opening this Pull Request, @" + parsed_hook.Sender.Login + " ! A Maintainer will review it soon!"
	comment := v3.IssueComment{Body: &response}

	_, _, err := globals.Myapp.RuntimeClient.Issues.CreateComment(context.TODO(), parsed_hook.Repository.Owner.Login, parsed_hook.Repository.Name, int(parsed_hook.PullRequest.Number), &comment)

	if err != nil {
		log.Printf("[ERROR] Could not Comment on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.PullRequest.Number, parsed_hook.PullRequest.Title)
		log.Println("Error ->", err)
	} else {
		log.Printf("[PRHANDLER] Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.PullRequest.Number, parsed_hook.PullRequest.Title)
	}
}

func newPRCommentHandler(parsedHook *ghwebhooks.IssueCommentPayload) {
	// Parse the current webhook

	is_maintainer, err := globals.Myapp.Dbmanager.CheckIsMaintainer(strings.ToLower(parsedHook.Sender.Login))
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
			err := globals.Myapp.Dbmanager.AssignBounty(
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
			comment := v3.IssueComment{Body: &response}

			_, _, new_err := globals.Myapp.RuntimeClient.Issues.CreateComment(context.TODO(), parsedHook.Repository.Owner.Login, parsedHook.Repository.Name, int(parsedHook.Issue.Number), &comment)
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

	//Creating hook parsers :
	hook_secret := ghwebhooks.Options.Secret(globals.Myapp.WebhookSecret)
	hook_parser, err := ghwebhooks.New(hook_secret)
	if err != nil {
		log.Println("[ERROR] Webhook parser creation Failed")
		panic(err)
	}

	//Listing all actions/Events to be parsed :
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

		if err == ghwebhooks.ErrEventNotFound {
			log.Println("[WARN] Undefined GitHub event received. err :", err)
			response.WriteHeader(http.StatusOK)
			return

		} else if err == ghwebhooks.ErrEventNotSpecifiedToParse {
			// FIXME Unsure about this
			log.Println("[WARN] This event hasn't been specified to parse", err)
			response.WriteHeader(http.StatusBadRequest)
			return

		} else {
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
