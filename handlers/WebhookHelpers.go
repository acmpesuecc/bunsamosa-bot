package handlers

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

// Default Issue Times
const defaultAssignment = 45
const defaultExtension = 30

const bountyPattern = `^!bounty\s+(\d+)$`
const assignPattern = `^!assign\s+@(\S+)\s*(\d*)$`
const deassignPattern = `^!deassign$`
const withdrawPattern = `^!withdraw$`
const extendPattern = `^!extend\s*(\d+)?$`

// var commandRegex = regexp.MustCompile(`^!\w+`)
var commandRegex = regexp.MustCompile(`^!.+`)
var bountyRegex = regexp.MustCompile(bountyPattern)
var assignRegex = regexp.MustCompile(assignPattern)
var deassignRegex = regexp.MustCompile(deassignPattern)
var extendRegex = regexp.MustCompile(extendPattern)

// function to check what the command is and parse accordingly
// to perform the correct action (assign / deassign issue, extended time for issue, contributor withdrawal)
func getCommand(comment string, logger *zerolog.Logger) string {
	comment = strings.TrimLeft(comment, " ")
	matches := commandRegex.FindStringSubmatch(comment)
	if len(matches) > 0 {
		botCommand := strings.Trim(matches[0], " ")
		logger.Info().Str("botCommand", botCommand).Msg("Initial bot command regex match")
		return botCommand
	} else {
		return ""
	}

}

// function to validate if PR bounty comment is in the correct format
// and get bounty points
func parseBountyPoints(comment string) (int, bool) {
	comment = strings.TrimLeft(comment, " ")

	// Compile the regular expression
	// Use FindStringSubmatch to search for the pattern in the text
	matches := bountyRegex.FindStringSubmatch(comment)
	//fmt.Println(matches)
	if len(matches) > 0 {
		// Extract the bounty number from the captured group
		bounty := matches[1]
		bounty_num, err := strconv.Atoi(bounty)
		return bounty_num, err == nil
	} else {
		return -1, false
	}
}

// function to validate if PR assign comment is in the correct format
// and assign issue to a contributor for x minutes (default is "defaultAssignment")
func parseAssign(comment string, logger *zerolog.Logger) (string, int, bool) {
	comment = strings.TrimLeft(comment, " ")

	// Compile the regular expression
	// Use FindStringSubmatch to search for the pattern in the text
	matches := assignRegex.FindStringSubmatch(comment)
	if len(matches) > 0 {
		message := strings.Split(strings.Trim(matches[0], " "), " ")
		var duration int
		var err error
		handle := matches[1]
		fmt.Println(message)
		// If time is defined
		if len(matches) > 2 && matches[2] != "" {
			durationStr := matches[2]
			duration, err = strconv.Atoi(durationStr)
			logger.Info().Str("scope", "PARSE_ASSIGN").Str("contributor_handle", handle).Int("duration", duration).Msg("Successfully parsed `assign` comment")
			return handle, duration, err == nil
		} else {
			// default time
			logger.Info().Str("scope", "PARSE_ASSIGN").Str("contributor_handle", handle).Int("duration", defaultAssignment).Msg("Successfully parsed `assign` comment")
			return handle, defaultAssignment, true
		}

	} else {
		logger.Error().Str("scope", "PARSE_ASSIGN").Msg("Failed to parse `assign` comment")
		return "", -1, false
	}
}

// function to validate if PR extend comment is in the correct format
// and extend issue for assigned contributor for x minutes (default is "defaultExtension")
func parseExtend(comment string) (int, bool) {
	comment = strings.TrimLeft(comment, " ")
	matches := extendRegex.FindStringSubmatch(comment)
	if len(matches) == 0 {
		return -1, false
	}
	// If time is defined
	if len(matches) > 1 && matches[1] != "" {
		timeStr := matches[1]
		time, err := strconv.Atoi(timeStr)
		if err != nil {
			return -1, false
		}
		return time, true
	} else {
		// default time
		return defaultExtension, true
	}

}

func extendToAssign(comment string, assignee string) string {
	comment = strings.TrimLeft(comment, " ")
	matches := extendRegex.FindStringSubmatch(comment)
	if len(matches) > 1 && matches[1] != "" {
		timeStr := matches[1]
		return "!assign @" + assignee + " " + timeStr
	} else {
		return "!assign @" + assignee + " 30"
	}
}

// Function to check if a URL is a Pull Request URL
func isPullRequest(url string) bool {
	// Github Pull Request URLs are of the form
	// https://github.com/<org>/<repo>/pull/<number>
	// If we can verify that the second-last element is a string
	// Then we can verify that the given URL is a pull request URL
	parts := strings.Split(url, "/")
	if slices.Contains(parts, "pulls") {
		return true
	} else {
		return false
	}

}
