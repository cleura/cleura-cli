package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLICapture runs the command tree against the given config file path with
// a controlled environment, returning stdout and stderr separately.
func runCLICapture(t *testing.T, configPath string, env map[string]string, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("CLEURA_CONFIG", configPath)
	for _, v := range []string{"CLEURA_PROFILE", "CLEURA_CLOUD", "CLEURA_API_URL", "CLEURA_API_USERNAME", "CLEURA_API_TOKEN", "CLEURA_API_PASSWORD", "CLEURA_REGION", "CLEURA_PROJECT_ID"} {
		t.Setenv(v, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	root := NewRootCommand("test")
	var out, errb bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errb)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errb.String(), err
}

func TestConfigSetEmptyValueMissingProfileNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	// Empty config: profile "ghost" does not exist.
	if err := os.WriteFile(path, []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := runCLICapture(t, path, nil, "config", "profile", "set", "region", "", "--profile", "ghost")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "ghost") {
		t.Errorf("unsetting on a missing profile must not create it; config now:\n%s", data)
	}
}

func TestKubeconfigRejectsSubSecondExpiration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("version: 1\ncurrent_profile: p\nprofiles:\n  p:\n    cloud: public\n    username: u\n    token: dummy\n    region: sto1\n    project_id: proj\n"), 0o600)
	// The sub-second guard fires before any network call.
	_, _, err := runCLICapture(t, path, nil, "gardener", "shoot", "kubeconfig", "x", "--expiration", "500ms")
	if err == nil || !strings.Contains(err.Error(), "at least 1s") {
		t.Fatalf("want 'at least 1s' rejection, got %v", err)
	}
}

func TestGetCredentialsWarnsOnlyOnGenuineMix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	os.WriteFile(path, []byte("version: 1\ncurrent_profile: p\nprofiles:\n  p:\n    cloud: public\n    username: alice\n    token: profiletoken\n"), 0o600)

	// env+env: a deliberate CI pair — must NOT warn.
	_, stderr, err := runCLICapture(t, path, map[string]string{"CLEURA_API_USERNAME": "bob", "CLEURA_API_TOKEN": "envtoken"}, "config", "get-credentials")
	if err != nil {
		t.Fatalf("env+env unexpected error: %v", err)
	}
	if strings.Contains(stderr, "may not authenticate") {
		t.Errorf("env+env is a deliberate pair; must not warn. stderr:\n%s", stderr)
	}

	// profile username + env token: a genuine cross-mix — must warn.
	_, stderr, err = runCLICapture(t, path, map[string]string{"CLEURA_API_TOKEN": "envtoken"}, "config", "get-credentials")
	if err != nil {
		t.Fatalf("mix unexpected error: %v", err)
	}
	if !strings.Contains(stderr, "may not authenticate") {
		t.Errorf("profile-username + env-token should warn. stderr:\n%s", stderr)
	}
}
