package main

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
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
	botHandle      = "ci-bunsamosabot[bot]"
	dbPath         = "test.db"
)

var (
	globalClient   *github.Client
	globalIssueNum int
	globalPRNum    int
	globalContext  = context.Background()
)

// --- Auth helpers ---

func generateJWT() string {
	keyPath := os.Getenv("CERT_FILE")
	privKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("❌ failed to read private key: %v", err)
	}
	block, _ := pem.Decode(privKeyBytes)
	if block == nil {
		log.Fatalf("❌ failed to parse PEM block containing private key")
	}
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("❌ failed to parse RSA private key: %v", err)
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
		log.Fatalf("❌ failed to sign JWT: %v", err)
	}
	return signed
}

func getInstallationToken(ctx context.Context, jwt string) string {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: jwt})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	token, _, err := client.Apps.CreateInstallationToken(ctx, installationID, nil)
	if err != nil {
		log.Fatalf("❌ failed to create installation token: %v", err)
	}
	return token.GetToken()
}

// --- Setup / Teardown ---

func setupGHClientAndToken() (*github.Client, string) {
	_jwt := generateJWT()
	token := getInstallationToken(globalContext, _jwt)
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(globalContext, ts)
	client := github.NewClient(tc)

	return client, token
}

func setupIssue(client *github.Client) int {
	// Create issue
	issueReq := &github.IssueRequest{
		Title: github.String("CI-Issue-pending"),
		Body:  github.String("Testing assignment and deassignment flow"),
	}
	issue, _, err := client.Issues.Create(globalContext, repoOwner, repoName, issueReq)
	if err != nil {
		log.Fatalf("❌ failed to create issue: %v", err)
	}
	issueNumber := issue.GetNumber()
	log.Printf("✅ Created issue #%d", issueNumber)

	// Update title to actual number
	updateReq := &github.IssueRequest{
		Title: github.String(fmt.Sprintf("CI-Issue-%d", issueNumber)),
	}
	_, _, err = client.Issues.Edit(globalContext, repoOwner, repoName, issueNumber, updateReq)
	if err != nil {
		log.Fatalf("❌ failed to update issue title: %v", err)
	}
	log.Printf("ℹ️ Renamed issue #%d", issueNumber)

	// Give bot time
	time.Sleep(5 * time.Second)

	return issueNumber
}

func teardownIssue(client *github.Client, issueNumber int) {
	state := "closed"
	_, _, err := client.Issues.Edit(globalContext, repoOwner, repoName, issueNumber, &github.IssueRequest{
		State: &state,
	})
	if err != nil {
		log.Fatalf("❌ failed to close issue #%d: %v", issueNumber, err)
	}
	log.Printf("✅ Closed issue #%d", issueNumber)
}

func setupPR(client *github.Client, token string) (int, string) {
	ctx := globalContext

	branchName := fmt.Sprintf("ci-test-branch-%d", time.Now().Unix())
	fileName := "ci_dummy.txt"
	prTitle := "CI Test PR"
	prBody := "This is a dummy PR created by tests"
	repoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, repoOwner, repoName)

	// --- Step 1: Create new branch with dummy file ---
	log.Printf("➡️ Creating test branch and committing dummy file")

	repoDir := "CI-Repo-Bunsamosa"

	// Step 0: cleanup if repo already exists
	if _, err := os.Stat(repoDir); err == nil {
		log.Printf("🧹 Removing existing repo dir: %s", repoDir)
		if err := os.RemoveAll(repoDir); err != nil {
			log.Fatalf("❌ failed to remove existing repo dir: %v", err)
		}
	}

	// Step 1: clone
	cmd := exec.Command("git", "clone", "https://github.com/foobaruwu/CI-Repo-Bunsamosa.git")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("❌ failed to clone repo: %v", err)
	}

	// Step 2+: run all following commands inside repoDir
	cmds := [][]string{
		{"git", "remote", "set-url", "origin", repoURL},
		{"git", "config", "user.email", "bot@example.com"},
		{"git", "config", "user.name", "CI Bot"},
		{"git", "checkout", "-b", branchName},
		{"bash", "-c", fmt.Sprintf("echo 'ci test' > %s", fileName)},
		{"git", "add", fileName},
		{"git", "commit", "-m", "ci: add dummy file"},
		{"git", "push", "origin", branchName},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = repoDir // 👈 run inside cloned repo
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("❌ failed to run %v: %v\nOutput:\n%s", c, err, string(out))
		}
	}

	// --- Step 2: Create PR ---
	log.Printf("➡️ Opening PR")
	newPR := &github.NewPullRequest{
		Title: github.String(prTitle),
		Head:  github.String(branchName),
		Base:  github.String("main"),
		Body:  github.String(prBody),
	}
	pr, _, err := client.PullRequests.Create(ctx, repoOwner, repoName, newPR)
	if err != nil {
		log.Fatalf("❌ failed to create PR: %v", err)
	}
	prNumber := pr.GetNumber()
	prAuthor := pr.User.GetLogin()
	log.Printf("✅ Created PR #%d by %s", prNumber, prAuthor)

	return prNumber, branchName
}

func teardownPR(client *github.Client, prNumber int, branchName string) {
	// Close PR
	state := "closed"
	_, _, err := client.PullRequests.Edit(globalContext, repoOwner, repoName, prNumber, &github.PullRequest{State: &state})
	if err != nil {
		log.Fatalf("❌ failed to close PR #%d: %v", prNumber, err)
	}
	log.Printf("✅ Closed PR #%d", prNumber)

	// Delete branch
	_, err = client.Git.DeleteRef(globalContext, repoOwner, repoName, "refs/heads/"+branchName)
	if err != nil {
		log.Fatalf("❌ failed to delete branch %s: %v", branchName, err)
	}
	log.Printf("✅ Deleted branch %s", branchName)
}

// --- Main entry for tests ---

func TestMain(m *testing.M) {
	client, token := setupGHClientAndToken()
	globalClient = client //to be used by the unit tests

	issueNum := setupIssue(client)
	globalIssueNum = issueNum

	prNum, branchName := setupPR(client, token)
	globalPRNum = prNum

	// Run tests
	code := m.Run()

	// Teardown once
	teardownIssue(globalClient, globalIssueNum)
	teardownPR(client, prNum, branchName)

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

func getBountyForUserOnPR(t *testing.T, dbPath, handle string, prNumber int) int {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Step 1: resolve internal issue.id from GitHub PR number
	var issueID int
	url := fmt.Sprintf("https://github.com/foobaruwu/CI-Repo-Bunsamosa/pull/%d", prNumber)
	if err := db.QueryRow("SELECT id FROM issues WHERE url = ?", url).Scan(&issueID); err != nil {
		t.Fatalf("failed to resolve issue.id for PR #%d: %v", prNumber, err)
	}

	// Step 2: check bounty_loggings for that contributor & issue
	var bounty int
	query := `
		SELECT COALESCE(SUM(bl.assigned_bounty), 0)
		FROM bounty_loggings bl
		JOIN contributors c ON bl.contributor_id = c.id
		WHERE c.github_handle = ? AND bl.issue_id = ?
	`
	if err := db.QueryRow(query, handle, issueID).Scan(&bounty); err != nil {
		t.Fatalf("failed to query bounty for %s on issue %d: %v", handle, issueID, err)
	}
	return bounty
}

func getBountyFromLeaderboard(t *testing.T, handle string) int {
	resp, err := http.Get("http://localhost:4000/leaderboard_mat")
	if err != nil {
		t.Fatalf("failed to fetch leaderboard: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var leaderboard []struct {
		GithubHandle string
		TotalBounty  int
	}
	if err := json.Unmarshal(body, &leaderboard); err != nil {
		t.Fatalf("failed to unmarshal leaderboard: %v", err)
	}

	for _, entry := range leaderboard {
		if entry.GithubHandle == handle {
			return entry.TotalBounty
		}
	}
	// If not found, treat as 0
	return 0
}

func TestPRBountyFlow(t *testing.T) {
	ctx := globalContext
	prAuthor := botHandle
	prNumber := globalPRNum

	// --- Step 3: Capture starting bounty points for PR author ---
	startPoints := getBountyFromLeaderboard(t, prAuthor)
	t.Logf("ℹ️ %s had %d bounty points before", prAuthor, startPoints)

	// --- Step 4: Comment !bounty 50 ---
	t.Log("➡️ Commenting !bounty 50 on PR")
	comment := &github.IssueComment{Body: github.String("!bounty 50")}
	_, _, err := globalClient.Issues.CreateComment(ctx, repoOwner, repoName, prNumber, comment)
	if err != nil {
		t.Fatalf("failed to create bounty comment: %v", err)
	}

	// --- Step 5: Wait and check bot reply ---
	t.Log("⏳ Waiting 20s for bot to reply...")
	time.Sleep(20 * time.Second)
	comments, _, err := globalClient.Issues.ListComments(ctx, repoOwner, repoName, prNumber, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	})
	if err != nil {
		t.Fatalf("failed to fetch PR comments: %v", err)
	}

	found := false
	for _, c := range comments {
		if c.User.GetLogin() == botHandle && strings.Contains(c.GetBody(), "Assigned 50 Bounty points") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("❌ expected bot reply confirming bounty assignment, not found")
	}
	t.Log("✅ Bot reply confirmed")

	//// --- Step 7: Verify DB logging ---
	//issueBounty := getBountyForUserOnPR(t, dbPath, prAuthor, prNumber)
	//if issueBounty != 50 {
	//	t.Fatalf("❌ expected bounty_loggings to record 50 points for %s on PR #%d, got %d",
	//		prAuthor, prNumber, issueBounty)
	//}
	//t.Logf("✅ bounty_loggings correctly recorded %d points for %s on PR #%d",
	//	issueBounty, prAuthor, prNumber)

	// --- Step 6: Verify leaderboard updated by +50 ---
	endPoints := getBountyFromLeaderboard(t, prAuthor)
	if endPoints-startPoints != 50 {
		t.Fatalf("❌ expected bounty points to increase by 50 for %s (before=%d, after=%d)", prAuthor, startPoints, endPoints)
	}
	t.Logf("✅ Leaderboard shows %s went from %d → %d (+50)", prAuthor, startPoints, endPoints)
}
