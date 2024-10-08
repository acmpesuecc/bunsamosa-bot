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
const TimerDaemonURL = "http://localhost:3000/"

// Setting logger in main.go
var SugaredLogger *zap.SugaredLogger

func newIssueHandler(parsed_hook *ghwebhooks.IssuesPayload) {

	// Generate a New Comment - Text is Customizable

	// TODO Refactor: Add these responses to the App Struct
	response := "Thank you for opening this issue! A Maintainer will review it soon!"
	comment := v3.IssueComment{Body: &response}

	_, _, err := globals.Myapp.RuntimeClient.Issues.CreateComment(context.TODO(), parsed_hook.Repository.Owner.Login, parsed_hook.Repository.Name, int(parsed_hook.Issue.Number), &comment)

	if err != nil {
		log.Printf("[ERROR] Could not Comment on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)
	} else {
		log.Printf("[ISSUEHANDLER] Successfully Commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)
	}
}

func newIssueCommentHandler(parsed_hook *ghwebhooks.IssueCommentPayload) {

	log.Printf("Received new comment on Repository [%s] Issue (#%d)[%s] Comment: %s\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title, parsed_hook.Comment.Body)

	// MAINTAINER:  !assgin @handle MINS/default: 45
	// MAINTAINER:  !deassign
	// CONTRIBUTOR: !withdraw

	issue_comment := parsed_hook.Comment.Body

	comment_command := getCommand(issue_comment)

	if strings.Contains(comment_command, "assign") {
		contributorHandle, time, success := parseAssign(comment_command)
		if success {
			db_success, err := globals.Myapp.Dbmanager.AssignIssue(
				parsed_hook.Comment.IssueURL,
				parsed_hook.Sender.Login,
				parsed_hook.Repository.URL,
			)

			if err != nil {
				SugaredLogger.Errorw("Failed to assign issue to %+v",
					parsed_hook.Sender.Login,
					zap.Strings("scope", []string{"PR_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
				)
			}

			if db_success {
				// http.Post(url string, contentType string, body io.Reader)

				requst := TimeoutEvent{
					EventID:     contributorHandle,
					TimeoutSecs: time,
					Emit:        fmt.Sprintf("Assign issue to %s", contributorHandle),
				}
				request_bytes , err:= json.Marshal(requst)
				if err != nil {
					SugaredLogger.Errorw("Failed to assign issue to %+v. Failed to marshal bytes for request to Timer-Daemon",
						parsed_hook.Sender.Login,
						zap.Strings("scope", []string{"PR_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
					)
				}

				// Sending a POST request to the Timer Daemon to emit
				// the request
				http.Post(TimerDaemonURL, "application/json", bytes.NewReader(request_bytes))

				// http.Post(TimerDaemonURL, "application/json", body io.Reader)
			} else {
				SugaredLogger.Errorw("Failed to assign issue to %+v",
					parsed_hook.Sender.Login,
					zap.Strings("scope", []string{"PR_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
				)
			}

		} else {
			SugaredLogger.Errorw("Failed to assign issue to %v",
				parsed_hook.Sender.Login,
				zap.Strings("scope", []string{"PR_COMMENT_HANDLER", "ASSIGN_ISSUE"}),
			)
		}
	} else if strings.Contains(comment_command, "deassign") {
		//todo
	} else if strings.Contains(comment_command, "withdraw") {
		//todo

		// first query db and check

	} else if strings.Contains(comment_command, "extend") {
		//todo
	} else {
		// Invalid command
		SugaredLogger.Errorw("Invalid bot command",
			zap.Strings("scope", []string{"ISSUE COMMENT"}),
		)
	}

	is_maintainer, err := globals.Myapp.Dbmanager.CheckIsMaintainer(strings.ToLower(parsed_hook.Sender.Login))
	if err != nil {
		log.Printf("[ERROR][BOUNTY] Could not check is_maintainer for issue %q\n", parsed_hook.Issue.Title)
	} else {
		log.Printf("[ISSUE_COMMENT_HANDLE] A maintainer %q has commented on issue %q with title %q\n", parsed_hook.Sender.Login, parsed_hook.Issue.URL, parsed_hook.Issue.Title)
	}

	if is_maintainer {
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

func newPRCommentHandler(parsed_hook *ghwebhooks.IssueCommentPayload) {
	// Parse the current webhook

	is_maintainer, err := globals.Myapp.Dbmanager.CheckIsMaintainer(strings.ToLower(parsed_hook.Sender.Login))
	if err != nil {
		log.Println("[ERROR][BOUNTY] Could not check is_maintainer ->", err)
		return
	}

	if is_maintainer {
		log.Println("A Maintainer Commented -> ")
		log.Printf("[PR_COMMENTHANDLER] Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)

		// parse the comment here to give a bounty
		bounty, valid := parseBountyPoints(parsed_hook.Comment.Body)

		if valid {

			// Assign the bounty points
			err := globals.Myapp.Dbmanager.AssignBounty(
				parsed_hook.Sender.Login,
				parsed_hook.Issue.User.Login,
				parsed_hook.Issue.PullRequest.HTMLURL,
				bounty,
			)
			if err != nil {
				log.Println("[ERROR][BOUNTY] Could not assign bounty points ->", err)
				return
			}

			log.Printf("[PR_COMMENTHANDLER] Successfully Assigned Bounty on Pull Request -> Repository [%s] PR (#%d)[%s] to user %s for %d points\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title, parsed_hook.Issue.User.Login, bounty)

			response := "Assigned " + fmt.Sprint(bounty) + " Bounty points to user @" + parsed_hook.Issue.User.Login + " !"
			comment := v3.IssueComment{Body: &response}

			_, _, new_err := globals.Myapp.RuntimeClient.Issues.CreateComment(context.TODO(), parsed_hook.Repository.Owner.Login, parsed_hook.Repository.Name, int(parsed_hook.Issue.Number), &comment)
			if new_err != nil {
				log.Printf("[ERROR] Could not Comment on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)
				log.Println("Error ->", new_err)
			} else {
				log.Printf("[PRHANDLER] Successfully Commented on Pull Request -> Repository [%s] PR (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)
			}

		}

	} else {
		log.Printf("[WARN] Someone else commented on Issue -> Repository [%s] Issue (#%d)[%s]\n", parsed_hook.Repository.FullName, parsed_hook.Issue.Number, parsed_hook.Issue.Title)
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
