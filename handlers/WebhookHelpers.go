package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Function to check if a string is in an array
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// Default Issue Times
const defaultAssignment = 45
const defaultExtension = 30

const bountyPattern = `^!bounty\s+(\d+)`
const assignPattern = `^!assign\s+@\w+(\s+\d+)?`
const deassignPattern = `^!deassign`
const withdrawPattern = `^!withdraw`
const extendPattern = `^!extend(\s+\d+)`

var commandRegex = regexp.MustCompile(`^!\w+`)
var bountyRegex = regexp.MustCompile(bountyPattern)
var assignRegex = regexp.MustCompile(assignPattern)
var deassignRegex = regexp.MustCompile(deassignPattern)
var extendRegex = regexp.MustCompile(extendPattern)

// function to check what the command is and parse accordingly
// to perform the correct action (assign / deassign issue, extended time for issue, contributor withdrawal)
func getCommand(comment string) string {
	comment = strings.TrimLeft(comment, " ")
	matches := commandRegex.FindStringSubmatch(comment)
	if len(matches) > 0 {
		return strings.Trim(matches[0][1:], " ")
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
func parseAssign(comment string) (string, int, bool) {
	comment = strings.TrimLeft(comment, " ")

	// Compile the regular expression
	// Use FindStringSubmatch to search for the pattern in the text
	matches := assignRegex.FindStringSubmatch(comment)
	if len(matches) > 0 {

		message := strings.Split(strings.Trim(matches[0], " "), " ")
		var time int
		var err error
		handle := message[1]
		fmt.Println(message)
		// If time is defined
		if len(message) > 2 {
			timeStr := message[2]
			time, err = strconv.Atoi(timeStr)
			return handle, time, err == nil
		} else {
			// default time
			return handle, defaultAssignment, true
		}

	} else {

		return "", -1, false
	}
}

// function to validate if PR extend comment is in the correct format
// and extend issue for assigned contributor for x minutes (default is "defaultExtension")
func parseExtend(comment string) (int, bool) {
	comment = strings.TrimLeft(comment, " ")
	matches := extendRegex.FindStringSubmatch(comment)
	if len(matches) > 0 {
		message := strings.Split(strings.Trim(matches[0], " "), " ")
		// If time is defined
		if len(message) > 2 {
			timeStr := strings.Trim(message[1], " ")
			time, err := strconv.Atoi(timeStr)
			return time, err == nil
		} else {
			// default time
			return defaultExtension, true
		}

	} else {
		return -1, false
	}
}

// Function to check if a URL is a Pull Request URL
func isPullRequest(url string) bool {
	// Github Pull Request URLs are of the form
	// https://github.com/<org>/<repo>/pull/<number>
	// If we can verify that the second-last element is a string
	// Then we can verify that the given URL is a pull request URL
	parts := strings.Split(url, "/")
	if contains(parts, "pulls") {
		log.Println("[PR_URLVALID] This is a Pull Request.", parts)
		return true
	} else {
		log.Println("[PR_URLVALID] This is not a Pull Request.", parts)
		return false
	}

}
