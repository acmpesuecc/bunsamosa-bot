package database

import (
	"os"
	"testing"
)

func TestIssueStorage(t *testing.T) {
	dbManager := DBManager{}

	if err:= os.Remove("../tests/dev.db"); err != nil {
		t.Errorf("Failed to remove test db")
	}
	dbManager.Init("../tests/dev.db")

	dbManager.db.Create(&Repo{URL: "repo 1"})
	dbManager.db.Create(&Repo{URL: "repo 2"})
	dbManager.db.Create(&Repo{URL: "repo 2"})

	t.Run("assign issue to contributors", func(t *testing.T) {
		got_bool, got_err := dbManager.AssignIssue("issue 1", "c 1", "repo 1")
		want_bool := true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 2", "c 2", "repo 2")
		want_bool = true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 3", "c 3", "repo 2")
		want_bool = true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
		got_bool, got_err = dbManager.AssignIssue("issue 1", "c 2", "repo 1")
		want_bool = false
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

	t.Run("withdraw by a contributor from an issue", func(t *testing.T) {
		got_bool, got_err := dbManager.WithdrawIssue("issue 3", "c 3")
		want_bool := true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

	t.Run("contributor withdrawing from an issue not assigned to them", func(t *testing.T) {
		got_bool, got_err := dbManager.WithdrawIssue("issue 2", "c 1")
		want_bool := false
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %v", got_bool, want_bool, got_err)
		}
	})

}
