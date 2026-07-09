package cli

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cleura/cleura-cli/internal/config"
)

const twoProfiles = `version: 1
current_profile: work
profiles:
  work:
    cloud: public
    username: alice
    token: dummy
  spare:
    cloud: compliant
    username: bob
`

func writeConfig(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConfigCurrent(t *testing.T) {
	path := writeConfig(t, twoProfiles)
	out, _, err := runCLICapture(t, path, nil, "config", "profile", "current")
	if err != nil || strings.TrimSpace(out) != "work" {
		t.Fatalf("config current = %q, %v; want work", out, err)
	}
	out, _, err = runCLICapture(t, path, map[string]string{"CLEURA_PROFILE": "spare"}, "config", "profile", "current")
	if err != nil || strings.TrimSpace(out) != "spare" {
		t.Fatalf("config current (env override) = %q, %v; want spare", out, err)
	}
}

func TestRenameProfile(t *testing.T) {
	path := writeConfig(t, twoProfiles)
	if _, _, err := runCLICapture(t, path, nil, "config", "profile", "rename", "work", "primary"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	data, _ := os.ReadFile(path)
	s := string(data)
	if strings.Contains(s, "work:") || !strings.Contains(s, "primary:") || !strings.Contains(s, "current_profile: primary") {
		t.Errorf("rename did not rekey and follow current_profile:\n%s", s)
	}
	if _, _, err := runCLICapture(t, path, nil, "config", "profile", "rename", "spare", "primary"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("rename to existing should refuse, got %v", err)
	}
	if _, _, err := runCLICapture(t, path, nil, "config", "profile", "rename", "ghost", "x"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("rename of missing should refuse, got %v", err)
	}
}

func TestConfigSetSignals(t *testing.T) {
	path := writeConfig(t, twoProfiles)
	// Creating a profile is announced.
	_, stderr, err := runCLICapture(t, path, nil, "config", "profile", "set", "region", "kna1", "--profile", "brandnew")
	if err != nil || !strings.Contains(stderr, `Created profile "brandnew"`) {
		t.Errorf("set on a new profile should announce creation; stderr=%q err=%v", stderr, err)
	}
	// Changing username on a profile with a token warns about desync.
	_, stderr, err = runCLICapture(t, path, nil, "config", "profile", "set", "username", "carol", "--profile", "work")
	if err != nil || !strings.Contains(stderr, "mismatched") {
		t.Errorf("changing username on a token profile should warn; stderr=%q err=%v", stderr, err)
	}
}

func TestDeleteCurrentProfileHints(t *testing.T) {
	path := writeConfig(t, twoProfiles)
	_, stderr, err := runCLICapture(t, path, nil, "config", "profile", "delete", "work")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "config profile use spare") {
		t.Errorf("deleting the current profile should hint at the remaining one; stderr=%q", stderr)
	}
}

func TestConfigViewGhostNote(t *testing.T) {
	path := writeConfig(t, twoProfiles)
	_, stderr, err := runCLICapture(t, path, nil, "config", "view", "--profile", "ghost")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "does not exist; showing defaults") {
		t.Errorf("config view on a nonexistent profile should note it; stderr=%q", stderr)
	}
}

func TestUserAuthError(t *testing.T) {
	s := config.Settings{ProfileName: "p"}
	forbidden := &http.Response{StatusCode: http.StatusForbidden, Status: "403 Forbidden"}

	// Privilege 403 → whoami hint, no login hint.
	privBody := []byte(`{"error":{"code":403,"message":"No access: Your user does not have access to this method."}}`)
	if err := userAuthError("listing users", s, forbidden, privBody); err == nil ||
		!strings.Contains(err.Error(), "whoami") || strings.Contains(err.Error(), "cleura login") {
		t.Errorf("privilege 403 should hint at whoami, not login: %v", err)
	}

	// Token 403 → login hint (from apiAuthError), no whoami hint.
	tokenBody := []byte(`{"error":{"code":403,"message":"Invalid token"}}`)
	if err := userAuthError("listing users", s, forbidden, tokenBody); err == nil ||
		strings.Contains(err.Error(), "whoami") || !strings.Contains(err.Error(), "cleura login") {
		t.Errorf("token 403 should hint at login, not whoami: %v", err)
	}

	// 404 → neither hint.
	notFound := &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found"}
	nfBody := []byte(`{"error":{"code":404,"message":"not found"}}`)
	if err := userAuthError("fetching user", s, notFound, nfBody); err == nil ||
		strings.Contains(err.Error(), "whoami") || strings.Contains(err.Error(), "cleura login") {
		t.Errorf("404 should carry neither hint: %v", err)
	}

	// A 403 with no parseable message must not get BOTH hints: with no
	// reason to judge by, apiAuthError's login hint stands, no whoami.
	if err := userAuthError("listing users", s, forbidden, []byte(``)); err == nil || strings.Contains(err.Error(), "whoami") {
		t.Errorf("message-less 403 should not add the whoami hint: %v", err)
	}
}

func TestListProfilesNoSpuriousOverrideNote(t *testing.T) {
	// current_profile empty + no override: there is nothing to override, so
	// no "selected for this run" note (regression from Batch C).
	path := writeConfig(t, "version: 1\nprofiles:\n  only:\n    cloud: public\n    username: u\n")
	_, stderr, err := runCLICapture(t, path, nil, "config", "profile", "list")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr, "selected for this run") {
		t.Errorf("no override present; must not print an override note; stderr=%q", stderr)
	}
}
