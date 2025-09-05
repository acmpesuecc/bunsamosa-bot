package main

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v55/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

const (
	appID          = 1024499
	installationID = 84355045
	repoOwner      = "foobaruwu"
	repoName       = "CI-Repo-Bunsamosa"
	assignee       = "DedLad"
	botHandle      = "ci-bunsamosabot[bot]" // <-- update to actual bot handle if different
	dbPath         = "test.db"              // <-- path to your sqlite db
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

// --- Git setup using bot creds ---

func setupGitWithBotToken(t *testing.T, token string) {
	repoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, repoOwner, repoName)
	cmds := [][]string{
		{"git", "remote", "set-url", "origin", repoURL},
		{"git", "config", "user.email", "bot@example.com"},
		{"git", "config", "user.name", "CI Bot"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run %v: %v", c, err)
		}
	}
}

// --- SQLite bounty check ---

func getBountyForUser(t *testing.T, dbPath, handle string) int {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var points int
	query := `
		SELECT COALESCE(SUM(bl.assigned_bounty), 0)
		FROM bounty_loggings bl
		JOIN contributors c ON bl.contributor_id = c.id
		WHERE c.github_handle = ?
	`
	if err := db.QueryRow(query, handle).Scan(&points); err != nil {
		t.Fatalf("failed to query bounty: %v", err)
	}
	return points
}

// --- Setup / Teardown ---

func setupIssue(t *testing.T) (*github.Client, int, string) {
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

	return client, issueNumber, token
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

// --- PR Cleanup ---

func cleanupPR(t *testing.T, client *github.Client, prNumber int, branchName string) {
	// Close PR
	state := "closed"
	_, _, err := client.PullRequests.Edit(globalContext, repoOwner, repoName, prNumber, &github.PullRequest{State: &state})
	if err != nil {
		t.Logf("⚠️ failed to close PR #%d: %v", prNumber, err)
	} else {
		t.Logf("Closed PR #%d", prNumber)
	}

	// Delete branch
	_, err = client.Git.DeleteRef(globalContext, repoOwner, repoName, "refs/heads/"+branchName)
	if err != nil {
		t.Logf("⚠️ failed to delete branch %s: %v", branchName, err)
	} else {
		t.Logf("Deleted branch %s", branchName)
	}
}

// --- Main entry for tests ---

func TestMain(m *testing.M) {
	// Setup once
	client, issueNum, token := setupIssue(&testing.T{})
	globalClient = client
	globalIssueNum = issueNum

	// Setup git creds
	setupGitWithBotToken(&testing.T{}, token)

	// Run tests
	code := m.Run()

	// Teardown once
	teardownIssue(&testing.T{}, globalClient, globalIssueNum)

	os.Exit(code)
}

// --- Tests ---

func TestAssignDeassign(t *testing.T) {
	ctx := globalContext

	t.Log("➡️ Starting TestAssignDeassign")

	// Assign
	t.Logf("💬 Commenting to assign %s ...", assignee)
	comment := &github.IssueComment{Body: github.String("!assign @" + assignee)}
	_, _, err := globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, comment)
	if err != nil {
		t.Fatalf("❌ failed to create assign comment: %v", err)
	}
	t.Log("⏳ Waiting 10s for bot to process assignment...")
	time.Sleep(10 * time.Second)

	// Verify assignee
	t.Log("🔍 Verifying that assignee is present...")
	updated, _, err := globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("❌ failed to fetch issue: %v", err)
	}
	found := false
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("❌ expected %s to be assigned, but not found", assignee)
	}
	t.Logf("✅ %s successfully assigned", assignee)

	// Deassign
	t.Logf("💬 Commenting to deassign %s ...", assignee)
	deassign := &github.IssueComment{Body: github.String("!deassign")}
	_, _, err = globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, deassign)
	if err != nil {
		t.Fatalf("❌ failed to create deassign comment: %v", err)
	}
	t.Log("⏳ Waiting 10s for bot to process deassignment...")
	time.Sleep(10 * time.Second)

	// Verify unassigned
	t.Log("🔍 Verifying that assignee is removed...")
	updated, _, err = globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("❌ failed to fetch issue: %v", err)
	}
	for _, a := range updated.Assignees {
		if a.GetLogin() == assignee {
			t.Fatalf("❌ expected %s to be unassigned, but still present", assignee)
		}
	}
	t.Logf("✅ %s successfully deassigned", assignee)

	t.Log("🎉 TestAssignDeassign completed successfully")
}

func TestAssignWithReminderAndDeassign(t *testing.T) {
	ctx := globalContext

	// Step 1: Assign with reminder of 1 minute
	assignComment := &github.IssueComment{Body: github.String("!assign @" + assignee + " 1")}
	_, _, err := globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, assignComment)
	if err != nil {
		t.Fatalf("failed to create assign-with-reminder comment: %v", err)
	}
	t.Log("Posted !assign with reminder (1 min)")

	// Step 2: Wait enough time for the reminder to fire (1 min + buffer)
	t.Log("Waiting 70 seconds for reminder...")
	time.Sleep(70 * time.Second)

	// Step 3: Fetch recent comments to check for reminder
	comments, _, err := globalClient.Issues.ListComments(ctx, repoOwner, repoName, globalIssueNum, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	})
	if err != nil {
		t.Fatalf("failed to list comments: %v", err)
	}

	foundReminder := false
	for _, c := range comments {
		body := c.GetBody()
		if strings.Contains(body, "timer for the @"+assignee+" to work on the issue has finished") {
			foundReminder = true
			break
		}
	}
	if !foundReminder {
		t.Fatalf("expected reminder comment for assignee %s, but not found", assignee)
	}
	t.Log("✅ Reminder comment found")

	// Step 4: Post !deassign
	deassignComment := &github.IssueComment{Body: github.String("!deassign")}
	_, _, err = globalClient.Issues.CreateComment(ctx, repoOwner, repoName, globalIssueNum, deassignComment)
	if err != nil {
		t.Fatalf("failed to create deassign comment: %v", err)
	}
	t.Log("Posted !deassign")

	// Step 5: Verify assignee is removed
	time.Sleep(10 * time.Second) // wait for bot to process
	updatedIssue, _, err := globalClient.Issues.Get(ctx, repoOwner, repoName, globalIssueNum)
	if err != nil {
		t.Fatalf("failed to fetch issue: %v", err)
	}
	for _, a := range updatedIssue.Assignees {
		if a.GetLogin() == assignee {
			t.Fatalf("expected %s to be unassigned, but still present", assignee)
		}
	}
	t.Log("✅ Assignee successfully removed after reminder")
}

// --- Example PR + bounty flow test ---

func TestPRBountyFlow(t *testing.T) {
	ctx := globalContext

	branchName := fmt.Sprintf("ci-test-branch-%d", time.Now().Unix())
	fileName := "ci_dummy.txt"
	prTitle := "CI Test PR"
	prBody := "This is a dummy PR created by tests"

	// --- Step 1: Create new branch with dummy file ---
	t.Log("➡️ Creating test branch and committing dummy file")
	cmds := [][]string{
		{""},
		{"git", "checkout", "-b", branchName},
		{"bash", "-c", fmt.Sprintf("echo 'ci test' > %s", fileName)},
		{"git", "add", fileName},
		{"git", "commit", "-m", "ci: add dummy file"},
		{"git", "push", "origin", branchName},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to run %v: %v", c, err)
		}
	}

	// --- Step 2: Create PR ---
	t.Log("➡️ Opening PR")
	newPR := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(branchName),
		Base:  github.String("main"),
		Body:  github.String(prBody),
	}
	pr, _, err := globalClient.PullRequests.Create(ctx, repoOwner, repoName, newPR)
	if err != nil {
		t.Fatalf("failed to create PR: %v", err)
	}
	prNumber := pr.GetNumber()
	prAuthor := pr.User.GetLogin()
	t.Logf("✅ Created PR #%d by %s", prNumber, prAuthor)

	// --- Step 3: Capture starting bounty points for PR author ---
	startPoints := getBountyForUser(t, dbPath, prAuthor)
	t.Logf("ℹ️ %s had %d bounty points before", prAuthor, startPoints)

	// --- Step 4: Comment !bounty 50 ---
	t.Log("➡️ Commenting !bounty 50 on PR")
	comment := &github.IssueComment{Body: github.String("!bounty 50")}
	_, _, err = globalClient.Issues.CreateComment(ctx, repoOwner, repoName, prNumber, comment)
	if err != nil {
		t.Fatalf("failed to create bounty comment: %v", err)
	}

	// --- Step 5: Wait and check bot reply ---
	t.Log("⏳ Waiting 10s for bot to reply...")
	time.Sleep(10 * time.Second)
	comments, _, err := globalClient.Issues.ListComments(ctx, repoOwner, repoName, prNumber, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	})
	if err != nil {
		t.Fatalf("failed to fetch PR comments: %v", err)
	}

	found := false
	for _, c := range comments {
		if c.User.GetLogin() == botHandle && strings.Contains(c.GetBody(), "Bounty of 50 assigned") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("❌ expected bot reply confirming bounty assignment, not found")
	}
	t.Log("✅ Bot reply confirmed")

	// --- Step 6: Verify DB updated ---
	endPoints := getBountyForUser(t, dbPath, prAuthor)
	if endPoints <= startPoints {
		t.Fatalf("❌ expected bounty points to increase for %s (before=%d, after=%d)", prAuthor, startPoints, endPoints)
	}
	t.Logf("✅ Bounty persisted in DB: %s went from %d → %d", prAuthor, startPoints, endPoints)

	// --- Step 7: Verify leaderboard matches ---
	resp, err := http.Get("http://localhost:4000/leaderboard_mat")
	if err != nil {
		t.Fatalf("failed to call leaderboard endpoint: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), prAuthor) {
		t.Fatalf("❌ leaderboard does not contain %s", prAuthor)
	}
	t.Log("✅ Leaderboard shows PR author with updated bounty")

	// --- Cleanup: close PR + delete branch ---
	cleanupPR(t, globalClient, prNumber, branchName)
}
