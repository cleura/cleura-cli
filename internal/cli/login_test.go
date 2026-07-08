package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runLogin executes cleura login against a scratch config with controlled
// stdin, returning the error.
func runLogin(t *testing.T, configYAML, stdin string, args ...string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLEURA_CONFIG", path)
	for _, v := range []string{"CLEURA_PROFILE", "CLEURA_CLOUD", "CLEURA_API_URL", "CLEURA_API_USERNAME", "CLEURA_API_TOKEN", "CLEURA_API_PASSWORD", "CLEURA_REGION", "CLEURA_PROJECT_ID"} {
		t.Setenv(v, "")
	}

	root := NewRootCommand("test")
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"login"}, args...))
	return root.Execute()
}

const occupiedProfile = `version: 1
current_profile: compliant
profiles:
  compliant:
    cloud: compliant
    username: alice
    token: tok-alice
`

func TestLoginRefusesIdentityOverwriteNonInteractive(t *testing.T) {
	// A different identity aimed at an occupied profile must be refused
	// before any password prompt or API call (empty stdin = answer "no").
	err := runLogin(t, occupiedProfile, "", "-u", "bob", "--cloud", "public")
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("want overwrite refusal, got %v", err)
	}
	if !strings.Contains(err.Error(), "alice") || !strings.Contains(err.Error(), "bob") {
		t.Errorf("refusal should name both identities, got %v", err)
	}
}

func TestLoginSameIdentityIsSilentTokenRefresh(t *testing.T) {
	// Same username and endpoint: no guard — the flow proceeds straight to
	// the password prompt (which fails here on exhausted stdin, proving the
	// guard did not interfere).
	err := runLogin(t, occupiedProfile, "", "-u", "alice", "--cloud", "compliant")
	if err == nil || !strings.Contains(err.Error(), "Password prompt") {
		t.Fatalf("same identity should reach the password prompt, got %v", err)
	}
}
