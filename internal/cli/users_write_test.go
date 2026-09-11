package cli

import (
	"context"
	"strings"
	"testing"
)

func TestUserWriteCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"user", "create"},
		{"user", "edit"},
		{"user", "delete"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Fatalf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}

	create, _, _ := root.Find([]string{"user", "create"})
	for _, f := range []string{"email", "first-name", "last-name", "ip-restriction"} {
		if create.Flags().Lookup(f) == nil {
			t.Errorf("user create is missing --%s", f)
		}
	}
	// The password must never be settable from the command line.
	if create.Flags().Lookup("password") != nil {
		t.Error("user create must not accept a --password flag")
	}

	edit, _, _ := root.Find([]string{"user", "edit"})
	for _, f := range []string{"username", "email", "first-name", "last-name", "ip-restriction", "clear-ip-restrictions", "set-password", "yes"} {
		if edit.Flags().Lookup(f) == nil {
			t.Errorf("user edit is missing --%s", f)
		}
	}
	if edit.Flags().Lookup("password") != nil {
		t.Error("user edit must not accept a --password flag")
	}
	// Privileges are their own noun; they must not reappear as flags here.
	for _, f := range []string{"privilege", "revoke-privilege"} {
		if create.Flags().Lookup(f) != nil || edit.Flags().Lookup(f) != nil {
			t.Errorf("--%s belongs to 'cleura user privilege', not to create/edit", f)
		}
	}

	del, _, _ := root.Find([]string{"user", "delete"})
	if del.Flags().Lookup("yes") == nil {
		t.Error("user delete is missing --yes")
	}
}

// Running edit with no flags must fail before any network call, rather than
// sending an empty PATCH that reports success while changing nothing.
func TestUserEditRequiresAChange(t *testing.T) {
	root := NewRootCommand("test")
	edit, _, err := root.Find([]string{"user", "edit"})
	if err != nil {
		t.Fatal(err)
	}
	edit.SetContext(context.Background())
	edit.SetArgs([]string{"someone"})
	err = edit.RunE(edit, []string{"someone"})
	if err == nil || !strings.Contains(err.Error(), "nothing to change") {
		t.Errorf("no-op edit: %v", err)
	}
}

func TestValidateCIDRs(t *testing.T) {
	if err := validateCIDRs([]string{"203.0.113.0/24", " 2001:db8::/32 "}); err != nil {
		t.Errorf("valid CIDRs: %v", err)
	}
	for _, bad := range []string{"203.0.113.5", "203.0.113.0/33", "not-a-cidr", ""} {
		if err := validateCIDRs([]string{bad}); err == nil {
			t.Errorf("%q should be rejected as a CIDR", bad)
		}
	}
}

// The edit body types firstname/lastname as non-nullable strings with a
// one-or-more pattern, so an empty value cannot mean "clear it". Fail fast
// rather than sending a value the API rejects.
func TestUserEditRejectsEmptyNames(t *testing.T) {
	for _, flag := range []string{"first-name", "last-name"} {
		root := NewRootCommand("test")
		edit, _, err := root.Find([]string{"user", "edit"})
		if err != nil {
			t.Fatal(err)
		}
		edit.SetContext(context.Background())
		if err := edit.Flags().Set(flag, ""); err != nil {
			t.Fatal(err)
		}
		err = edit.RunE(edit, []string{"someone"})
		if err == nil || !strings.Contains(err.Error(), "cannot be cleared") {
			t.Errorf("--%s \"\": got %v, want a cannot-be-cleared error", flag, err)
		}
	}
}

func TestUserEditRejectsBadCIDR(t *testing.T) {
	root := NewRootCommand("test")
	edit, _, err := root.Find([]string{"user", "edit"})
	if err != nil {
		t.Fatal(err)
	}
	edit.SetContext(context.Background())
	if err := edit.Flags().Set("ip-restriction", "203.0.113.5"); err != nil {
		t.Fatal(err)
	}
	err = edit.RunE(edit, []string{"someone"})
	if err == nil || !strings.Contains(err.Error(), "is not a CIDR block") {
		t.Errorf("bad CIDR: %v", err)
	}
}
