package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cleura/cleura-cli/internal/config"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func privilege(t api.UserUserLoginPrivilegeType, projects ...api.CommonUserLoginProjectPrivilege) *api.CommonUserLoginPrivilege {
	p := &api.CommonUserLoginPrivilege{Type: t}
	if len(projects) > 0 {
		p.ProjectPrivileges = &projects
	}
	return p
}

func TestRolesSummary(t *testing.T) {
	if got := rolesSummary(api.CommonUserLoginPrivileges{}); got != "-" {
		t.Errorf("empty summary = %q", got)
	}

	got := rolesSummary(api.CommonUserLoginPrivileges{
		Invoice:   privilege(api.Read),
		Openstack: privilege(api.Project, api.CommonUserLoginProjectPrivilege{ProjectId: "p1", Type: api.Full}, api.CommonUserLoginProjectPrivilege{ProjectId: "p2", Type: api.Read}),
		Users:     privilege(api.Full),
	})
	want := "invoice:read openstack:project(2) users:full"
	if got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}

	// Every area at full access compresses (the typical administrator).
	full := api.CommonUserLoginPrivileges{
		Account: privilege(api.Full), AiGateway: privilege(api.Full),
		Application: privilege(api.Full), Invoice: privilege(api.Full),
		Monitoring: privilege(api.Full), Openstack: privilege(api.Full),
		Users: privilege(api.Full),
	}
	if got := rolesSummary(full); got != "full (all areas)" {
		t.Errorf("all-full summary = %q", got)
	}
}

func TestPrivilegeLabel(t *testing.T) {
	if got := privilegeLabel(privilege(api.Full)); got != "Full Access" {
		t.Errorf("full = %q", got)
	}
	if got := privilegeLabel(privilege(api.Project, api.CommonUserLoginProjectPrivilege{}, api.CommonUserLoginProjectPrivilege{})); got != "Project Access (2 projects)" {
		t.Errorf("project = %q", got)
	}
}

func TestTwoFactorSummary(t *testing.T) {
	if got := twoFactorSummary(nil); got != "none" {
		t.Errorf("nil = %q", got)
	}
	keys := []api.CommonTwoFactorLoginWebauthn{
		{Name: "yubikey", Status: api.Active},
		{Name: "backup", Status: api.Active},
	}
	got := twoFactorSummary(&api.CommonUserLoginTwoFactorLogin{
		Sms:      &api.CommonTwoFactorLoginSms{Status: api.Active},
		Webauthn: &keys,
	})
	if got != "sms, webauthn (2 keys)" {
		t.Errorf("summary = %q", got)
	}

	// A single active key is singular.
	oneKey := []api.CommonTwoFactorLoginWebauthn{{Status: api.Active}}
	if got := twoFactorSummary(&api.CommonUserLoginTwoFactorLogin{Webauthn: &oneKey}); got != "webauthn (1 key)" {
		t.Errorf("single key = %q", got)
	}

	// Enrollments awaiting verification protect nothing: 2FA=no, but the
	// pending state is still named in the detail view.
	pending := &api.CommonUserLoginTwoFactorLogin{
		Sms: &api.CommonTwoFactorLoginSms{Status: api.AwaitVerification},
	}
	if hasTwoFactor(pending) {
		t.Error("awaiting-verification SMS must not count as active 2FA")
	}
	if got := twoFactorSummary(pending); got != "sms (awaiting verification)" {
		t.Errorf("pending = %q", got)
	}
}

// fakeUserAPI serves the two user endpoints lookupUser exercises.
func fakeUserAPI(t *testing.T, listCalls *int) *cleura.Client {
	t.Helper()
	alice := map[string]any{"id": 7, "name": "alice", "admin": false, "privileges": map[string]any{}, "currency": map[string]any{"id": 1, "code": "SEK", "name": "krona"}, "ip_restrictions": []any{}, "auth_provider_id": 1}
	bob := map[string]any{"id": 42, "name": "bob", "admin": false, "privileges": map[string]any{}, "currency": map[string]any{"id": 1, "code": "SEK", "name": "krona"}, "ip_restrictions": []any{}, "auth_provider_id": 1}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /identity/v2/user/42", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(bob)
	})
	mux.HandleFunc("GET /identity/v2/user/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": 404, "message": "User not found"}}`))
	})
	mux.HandleFunc("GET /identity/v2/users", func(w http.ResponseWriter, r *http.Request) {
		*listCalls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{alice, bob})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := cleura.NewClientWithCredentials(srv.URL, "u", "t")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestLookupUser(t *testing.T) {
	listCalls := 0
	client := fakeUserAPI(t, &listCalls)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	settings := config.Settings{ProfileName: "p"}

	user, err := lookupUser(cmd, settings, client, "42")
	if err != nil || user.Name != "bob" {
		t.Fatalf("by id: %v, %v", user, err)
	}

	user, err = lookupUser(cmd, settings, client, "alice")
	if err != nil || user.Name != "alice" || user.Id != 7 {
		t.Fatalf("by username: %+v, %v", user, err)
	}
	if listCalls != 1 {
		t.Errorf("username lookup should use the list once, used %d", listCalls)
	}

	if _, err := lookupUser(cmd, settings, client, "ghost"); err == nil || !strings.Contains(err.Error(), `no user with ID or username "ghost"`) {
		t.Errorf("unknown name: %v", err)
	}

	// Numeric misses report the API's answer without a fallback list call.
	before := listCalls
	if _, err := lookupUser(cmd, settings, client, "99999"); err == nil || !strings.Contains(err.Error(), "User not found") {
		t.Errorf("unknown id: %v", err)
	}
	if listCalls != before {
		t.Error("numeric lookup must not fall back to the list")
	}
}

func TestLookupUserAuthFailureSkipsFallback(t *testing.T) {
	listCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("GET /identity/v2/user/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": 403, "message": "Invalid token"}}`))
	})
	mux.HandleFunc("GET /identity/v2/users", func(w http.ResponseWriter, r *http.Request) { listCalls++ })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, _ := cleura.NewClientWithCredentials(srv.URL, "u", "t")

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	_, err := lookupUser(cmd, config.Settings{ProfileName: "p"}, client, "someone")
	if err == nil || !strings.Contains(err.Error(), "Invalid token") {
		t.Fatalf("want auth error, got %v", err)
	}
	if listCalls != 0 {
		t.Error("auth failures must not trigger a second doomed request")
	}
}

func TestDisplayName(t *testing.T) {
	s := func(v string) *string { return &v }
	if got := displayName(nil, s("Doe")); got != "Doe" {
		t.Errorf("last-only = %q (leading-space regression)", got)
	}
	if got := displayName(s("Jane"), s("Doe")); got != "Jane Doe" {
		t.Errorf("full = %q", got)
	}
	if got := displayName(nil, nil); got != "" {
		t.Errorf("empty = %q", got)
	}
}
