package database_test

import (
	"testing"

	"github.com/anirudhRowjee/bunsamosa-bot/globals"
)

func TestAssignIssue(t *testing.T) {
	t.Run("assign issue to contributors", func(t *testing.T) {
		globals.Myapp = globals.App{}
		globals.Myapp.Db_connection_string = "test.db"
		globals.Myapp.Initialize_database()

		got_bool, got_err := globals.Myapp.Dbmanager.AssignIssue("issue 1", "c 1")
		want_bool := true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %q", got_bool, want_bool, got_err)
		}
		got_bool, got_err = globals.Myapp.Dbmanager.AssignIssue("issue 2", "c 2")
		want_bool = true
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %q", got_bool, want_bool, got_err)
		}
		got_bool, got_err = globals.Myapp.Dbmanager.AssignIssue("issue 1", "c 2")
		want_bool = false
		if got_bool != want_bool {
			t.Errorf("got_bool: %t, want_bool: %t, got_error: %q", got_bool, want_bool, got_err)
		}
	})
}
