package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
)

const (
	appID          = 1024499
	installationID = 84355045
	repoOwner      = "foobaruwu"
	repoName       = "CI-Repo-Bunsamosa"
	assignee       = "DedLad"
)

var (
	globalClient   *github.Client
	globalIssueNum int
	globalContext  = context.Background()
)

// --- Auth helpers ---

func generateJWT(t *testing.T) string {
	keyPath := os.Getenv("CERT_FILE")
	privKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read private key: %v", err)
	}
	block, _ := pem.Decode(privKeyBytes)
	if block == nil {
		t.Fatal("failed to parse PEM block containing private key")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse RSA private key: %v", err)
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    fmt.Sprint(appID),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}
	return signed
}

func getInstallationToken(ctx context.Context, t *testing.T, jwt string) string {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwt})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		t.Fatalf("failed to create installation token: %v", err)
	}
	return token.GetToken()
}

// --- Setup / Teardown ---

func setupIssue(t *testing.T) (*github.Client, int) {
	jwt := generateJWT(t)
	token := getInstallationToken(globalContext, t, jwt)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(globalContext, ts)
	client := github.NewClient(tc)

	// Create issue
	issueReq := &github.IssueRequest{
		Title: github.String("CI-Issue-pending"),
		Body:  github.String("Testing assignment and deassignment flow"),
	}
	issue, _, err := client.Issues.Create(globalContext, repoOwner, repoName, issueReq)
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}
	issueNumber := issue.GetNumber()
	t.Logf("Created issue #%d", issueNumber)

	// Update title to actual number
	updateReq := &github.IssueRequest{
		Title: github.String(fmt.Sprintf("CI-Issue-%d", issueNumber)),
	}
	_, _, err = client.Issues.Edit(globalContext, repoOwner, repoName, issueNumber, updateReq)
	if err != nil {
		t.Fatalf("failed to update issue title: %v", err)
	}

	// Give bot time
	time.Sleep(5 * time.Second)

	return client, issueNumber
}

func teardownIssue(t *testing.T, client *github.Client, issueNumber int) {
	state := "closed"
	_, _, err := client.Issues.Edit(globalContext, repoOwner, repoName, issueNumber, &github.IssueRequest{
		State: &state,
	})
	if err != nil {
		t.Logf("⚠️ failed to close issue #%d: %v", issueNumber, err)
	} else {
		t.Logf("Closed issue #%d", issueNumber)
	}
}

// --- Main entry for tests ---
func TestMain(m *testing.M) {
	// Setup once
	client, issueNum := setupIssue(&testing.T{})
	globalClient = client
	globalIssueNum = issueNum

	// Run tests
	code := m.Run()

	// Teardown once
	teardownIssue(&testing.T{}, globalClient, globalIssueNum)

	os.Exit(code)
}

// --- Tests ---

func TestAssignDeassign(t *testing.T) {
	ctx := globalContext

	// Assign
	comment := &github.IssueComment{Body: github.String("!assign @" + assignee)}
	_, _, err := globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, comment)
	if err != nil {
		t.Fatalf("failed to create assign comment: %v", err)
	}
	time.Sleep(10 * time.Second)

	// Verify assignee
	updated, _, err := globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	found := false
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s to be assigned, but not found", assignee)
	}

	// Deassign
	deassign := &github.IssueComment{Body: github.String("!deassign")}
	_, _, err = globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, deassign)
	if err != nil {
		t.Fatalf("failed to create deassign comment: %v", err)
	}
	time.Sleep(10 * time.Second)

	// Verify unassigned
	updated, _, err = globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			t.Fatalf("expected %s to be unassigned, but still present", assignee)
		}
	}
}

func TestAssignDeassignFormatted(t *testing.T) {
	ctx := globalContext

	// Assign with space + newline
	comment := &github.IssueComment{Body: github.String(" !assign @" + assignee + "\n")}
	_, _, err := globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, comment)
	if err != nil {
		t.Fatalf("failed to create formatted assign comment: %v", err)
	}
	time.Sleep(10 * time.Second)

	// Verify assignee
	updated, _, err := globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	found := false
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s to be assigned with formatted command, but not found", assignee)
	}

	// Deassign with space + newline
	deassign := &github.IssueComment{Body: github.String(" !deassign\n")}
	_, _, err = globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, deassign)
	if err != nil {
		t.Fatalf("failed to create formatted deassign comment: %v", err)
	}
	time.Sleep(10 * time.Second)

	// Verify unassigned
	updated, _, err = globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			t.Fatalf("expected %s to be unassigned with formatted command, but still present", assignee)
		}
	}
}
