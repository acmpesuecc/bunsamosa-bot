package main

import (
	"context"
	_ "crypto/rsa"
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

// Replace with your GitHub App details
const (
	appID          = 1024499  // your GitHub App ID
	installationID = 84355045 // your installation ID
	repoOwner      = "foobaruwu"
	repoName       = "CI-Repo-Bunsamosa"
	assignee       = "DedLad"
)

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

	// Create installation token
	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		t.Fatalf("failed to create installation token: %v", err)
	}
	return token.GetToken()
}

func TestAssignDeassign(t *testing.T) {
	ctx := context.Background()
	jwt := generateJWT(t)
	token := getInstallationToken(ctx, t, jwt)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Step 1: Create issue
	issueReq := &github.IssueRequest{
		Title: github.String("CI-Issue-pending"), // temporary
		Body:  github.String("Testing assignment and deassignment flow"),
	}
	issue, _, err := client.Issues.Create(ctx, repoOwner, repoName, issueReq)
	if err != nil {
		t.Fatalf("failed to create issue: %v", err)
	}
	issueNumber := issue.GetNumber()

	// Step 1b: Update title to include real number
	updateReq := &github.IssueRequest{
		Title: github.String(fmt.Sprintf("CI-Issue-%d", issueNumber)),
	}
	_, _, err = client.Issues.Edit(ctx, repoOwner, repoName, issueNumber, updateReq)
	if err != nil {
		t.Fatalf("failed to update issue title: %v", err)
	}

	// Defer cleanup: close the issue at the end of the test
	defer func() {
		state := "closed"
		_, _, err := client.Issues.Edit(ctx, repoOwner, repoName, issueNumber, &github.IssueRequest{
			State: &state,
		})
		if err != nil {
			t.Logf("⚠️ failed to close issue #%d: %v", issueNumber, err)
		}
	}()

	// Wait a few seconds before commenting, to ensure bot sees the new issue
	time.Sleep(5 * time.Second)

	// Step 2: Comment "!assign @Dedlad"
	comment := &github.IssueComment{Body: github.String("!assign @" + assignee)}
	_, _, err = client.Issues.CreateComment(ctx, repoOwner, repoName, issueNumber, comment)
	if err != nil {
		t.Fatalf("failed to create comment: %v", err)
	}

	// Wait for bot to process
	time.Sleep(10 * time.Second)

	// Step 3: Verify assignee present
	updatedIssue, _, err := client.Issues.Get(ctx, repoOwner, repoName, issueNumber)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	found := false
	for _, a := range updatedIssue.Assignees {
		if a.GetLogin() == assignee {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected %s to be assigned, but not found", assignee)
	}

	// Step 4: Comment "!deassign"
	deassignComment := &github.IssueComment{Body: github.String("!deassign")}
	_, _, err = client.Issues.CreateComment(ctx, repoOwner, repoName, issueNumber, deassignComment)
	if err != nil {
		t.Fatalf("failed to create deassign comment: %v", err)
	}

	// Wait for bot
	time.Sleep(15 * time.Second)

	// Step 5: Verify assignee removed
	updatedIssue, _, err = client.Issues.Get(ctx, repoOwner, repoName, issueNumber)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	for _, a := range updatedIssue.Assignees {
		if a.GetLogin() == assignee {
			t.Fatalf("expected %s to be unassigned, but still present", assignee)
		}
	}
}
