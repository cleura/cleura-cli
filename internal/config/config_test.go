package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLEURA_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	p := cfg.Profile("default")
	p.Cloud = "public"
	p.Username = "johndoe"
	p.Token = "secret"
	cfg.CurrentProfile = "default"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config file permissions = %o, want 600", perm)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Profiles["default"]
	if got == nil || got.Username != "johndoe" || got.Token != "secret" || got.Cloud != "public" {
		t.Errorf("round-trip mismatch: %+v", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# Managed by the cleura CLI") {
		t.Errorf("saved config missing the managed-file header")
	}
}

func TestVersionStampAndFutureRejection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLEURA_CONFIG", path)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Profile("default").Token = "t"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "version: 1") {
		t.Errorf("Save should stamp the schema version, got:\n%s", data)
	}

	if err := os.WriteFile(path, []byte("version: 99\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "newer cleura") {
		t.Errorf("future config versions must be rejected with guidance, got: %v", err)
	}
}

func TestTokenStoredAtResolution(t *testing.T) {
	t.Setenv("CLEURA_PROFILE", "")
	t.Setenv("CLEURA_API_TOKEN", "")
	stored := time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC)
	cfg := &Config{Profiles: map[string]*Profile{
		"default": {Username: "u", Token: "t", TokenStoredAt: stored},
	}}

	if s := cfg.Resolve(Flags{}); !s.TokenStoredAt.Equal(stored) {
		t.Errorf("profile token should carry its stored-at time, got %v", s.TokenStoredAt)
	}

	// An env token has unknown age: TokenStoredAt must stay zero.
	t.Setenv("CLEURA_API_TOKEN", "env-token")
	if s := cfg.Resolve(Flags{}); !s.TokenStoredAt.IsZero() {
		t.Errorf("env token must not inherit the profile's stored-at time, got %v", s.TokenStoredAt)
	}
}

func TestLoadCorruptConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLEURA_CONFIG", path)
	if err := os.WriteFile(path, []byte("profiles: hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "run 'cleura login' to recreate it") {
		t.Errorf("corrupt config error should include a recovery hint, got: %v", err)
	}
}

func TestResolveProfileExists(t *testing.T) {
	t.Setenv("CLEURA_PROFILE", "")
	// A present key with a nil value (hand-edited config) counts as existing.
	cfg := &Config{Profiles: map[string]*Profile{"real": {}, "nilentry": nil}}
	if s := cfg.Resolve(Flags{Profile: "real"}); !s.ProfileExists {
		t.Error("existing profile reported as missing")
	}
	if s := cfg.Resolve(Flags{Profile: "nilentry"}); !s.ProfileExists {
		t.Error("present-but-nil profile key must count as existing")
	}
	if s := cfg.Resolve(Flags{Profile: "ghost"}); s.ProfileExists {
		t.Error("missing profile reported as existing")
	}
}

func TestResolvePrecedence(t *testing.T) {
	t.Setenv("CLEURA_CONFIG", filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("CLEURA_PROFILE", "")
	t.Setenv("CLEURA_CLOUD", "")
	t.Setenv("CLEURA_API_URL", "")
	t.Setenv("CLEURA_API_USERNAME", "")
	t.Setenv("CLEURA_API_TOKEN", "")
	t.Setenv("CLEURA_REGION", "")
	t.Setenv("CLEURA_PROJECT_ID", "")

	cfg := &Config{
		CurrentProfile: "work",
		Profiles: map[string]*Profile{
			"work": {Cloud: "compliant", Username: "stored-user", Token: "stored-token"},
		},
	}

	s := cfg.Resolve(Flags{})
	if s.ProfileName != "work" || s.Cloud != "compliant" || s.Username != "stored-user" {
		t.Errorf("profile values not used: %+v", s)
	}

	t.Setenv("CLEURA_API_USERNAME", "env-user")
	s = cfg.Resolve(Flags{})
	if s.Username != "env-user" {
		t.Errorf("env should override profile, got %q", s.Username)
	}

	s = cfg.Resolve(Flags{Cloud: "public", Profile: "other"})
	if s.Cloud != "public" || s.ProfileName != "other" {
		t.Errorf("flags should override everything: %+v", s)
	}
	if s.Username != "env-user" {
		t.Errorf("env should still apply for unknown profile, got %q", s.Username)
	}

	url, err := s.ResolveURL()
	if err != nil || url != "https://rest.cleura.cloud" {
		t.Errorf("ResolveURL() = %q, %v", url, err)
	}
}

func TestResolveEndpoint(t *testing.T) {
	t.Setenv("CLEURA_PROFILE", "")
	t.Setenv("CLEURA_CLOUD", "")
	t.Setenv("CLEURA_API_URL", "")

	private := "https://rest.cloud.acme.example"
	cfg := &Config{Profiles: map[string]*Profile{
		"acme":    {APIURL: private},
		"private": {Cloud: "acme", APIURL: private},
	}}

	// A stored api_url with no cloud is an incomplete private-cloud config: the
	// endpoint uses the api_url, but the cloud name is left EMPTY rather than
	// invented as "public" (gardener then reports the missing cloud).
	s := cfg.Resolve(Flags{Profile: "acme"})
	if s.APIURL != private || s.Cloud != "" {
		t.Errorf("api_url-only profile should use the url and leave cloud empty: %+v", s)
	}

	// A private cloud stores both: the URL for the endpoint, the cloud name
	// for path parameters.
	s = cfg.Resolve(Flags{Profile: "private"})
	if s.APIURL != private || s.Cloud != "acme" {
		t.Errorf("private cloud should keep both url and cloud name: %+v", s)
	}

	// Explicitly selecting the profile's own cloud must keep using its
	// paired api_url instead of dead-ending on the missing default URL.
	s = cfg.Resolve(Flags{Profile: "private", Cloud: "acme"})
	if s.APIURL != private || s.Cloud != "acme" {
		t.Errorf("--cloud matching the stored pair should keep the api_url: %+v", s)
	}
	t.Setenv("CLEURA_CLOUD", "acme")
	s = cfg.Resolve(Flags{Profile: "private"})
	if s.APIURL != private || s.Cloud != "acme" {
		t.Errorf("env cloud matching the stored pair should keep the api_url: %+v", s)
	}
	t.Setenv("CLEURA_CLOUD", "")

	// A live, unpaired env URL serves an explicitly selected cloud — the
	// user is overriding the endpoint right now — and outranks the
	// profile's stored pairing.
	t.Setenv("CLEURA_API_URL", "https://new.acme.example")
	s = cfg.Resolve(Flags{Profile: "private", Cloud: "acme"})
	if s.APIURL != "https://new.acme.example" {
		t.Errorf("env URL should outrank the profile's paired api_url: %+v", s)
	}
	t.Setenv("CLEURA_API_URL", "")

	// An explicit --cloud flag must beat a stored api_url.
	s = cfg.Resolve(Flags{Profile: "acme", Cloud: "public"})
	if s.Cloud != "public" || s.APIURL != "" {
		t.Errorf("--cloud should override stored api_url: %+v", s)
	}

	// CLEURA_CLOUD (env) also beats the stored api_url.
	t.Setenv("CLEURA_CLOUD", "compliant")
	s = cfg.Resolve(Flags{Profile: "acme"})
	if s.Cloud != "compliant" || s.APIURL != "" {
		t.Errorf("env cloud should override stored api_url: %+v", s)
	}

	// Within the env source an explicit URL wins for the endpoint, while the
	// named cloud stays the logical cloud.
	t.Setenv("CLEURA_API_URL", private)
	s = cfg.Resolve(Flags{Profile: "acme"})
	if s.APIURL != private || s.Cloud != "compliant" {
		t.Errorf("env api_url should win within the env source: %+v", s)
	}

	// Nothing set anywhere defaults to the public cloud.
	t.Setenv("CLEURA_CLOUD", "")
	t.Setenv("CLEURA_API_URL", "")
	s = cfg.Resolve(Flags{Profile: "fresh"})
	if s.Cloud != "public" || s.APIURL != "" {
		t.Errorf("default should be public: %+v", s)
	}
}

func TestResolveSources(t *testing.T) {
	t.Setenv("CLEURA_PROFILE", "")
	t.Setenv("CLEURA_CLOUD", "")
	t.Setenv("CLEURA_API_URL", "")
	t.Setenv("CLEURA_API_USERNAME", "")
	t.Setenv("CLEURA_API_TOKEN", "")
	t.Setenv("CLEURA_REGION", "")
	t.Setenv("CLEURA_PROJECT_ID", "")

	cfg := &Config{
		CurrentProfile: "work",
		Profiles: map[string]*Profile{
			"work": {Cloud: "compliant", Username: "u", Token: "t", Region: "sto1"},
			"acme": {APIURL: "https://rest.cloud.acme.example"},
		},
	}

	s := cfg.Resolve(Flags{})
	want := Sources{Profile: "config", Cloud: "profile", Username: "profile", Token: "profile", Region: "profile", Endpoint: "profile"}
	if s.Sources != want {
		t.Errorf("profile-backed sources = %+v, want %+v", s.Sources, want)
	}

	t.Setenv("CLEURA_API_TOKEN", "envtok")
	s = cfg.Resolve(Flags{})
	if s.Sources.Token != "$CLEURA_API_TOKEN" {
		t.Errorf("env token source = %q", s.Sources.Token)
	}

	s = cfg.Resolve(Flags{Profile: "acme", Cloud: "public"})
	if s.Sources.Profile != "--profile" || s.Sources.Cloud != "--cloud" {
		t.Errorf("flag sources = %+v", s.Sources)
	}
	if s.Sources.APIURL != "" || s.APIURL != "" {
		t.Errorf("flag --cloud should suppress stored api_url: %+v", s)
	}

	s = cfg.Resolve(Flags{Profile: "acme"})
	if s.Sources.APIURL != "profile" || s.Sources.Cloud != "" {
		t.Errorf("acme sources = %+v (cloud should be unset, not defaulted)", s.Sources)
	}
	if s.Sources.Endpoint != "profile" {
		t.Errorf("endpoint source should be the api_url's source, got %q", s.Sources.Endpoint)
	}

	s = cfg.Resolve(Flags{Profile: "fresh"})
	if s.Sources.Cloud != "default" || s.Sources.Username != "" {
		t.Errorf("fresh profile sources = %+v", s.Sources)
	}
	if s.Sources.Endpoint != "default" {
		t.Errorf("default endpoint source = %q, want default", s.Sources.Endpoint)
	}

	// Endpoint determined by a named cloud reports the cloud's source.
	s = cfg.Resolve(Flags{Profile: "work"})
	if s.Sources.Endpoint != "profile" {
		t.Errorf("cloud-determined endpoint source = %q, want profile", s.Sources.Endpoint)
	}
}

func TestResolveRegionAndProject(t *testing.T) {
	t.Setenv("CLEURA_PROFILE", "")
	t.Setenv("CLEURA_REGION", "")
	t.Setenv("CLEURA_PROJECT_ID", "")

	cfg := &Config{Profiles: map[string]*Profile{
		"default": {Region: "sto1", ProjectID: "proj-stored"},
	}}

	s := cfg.Resolve(Flags{})
	if s.Region != "sto1" || s.ProjectID != "proj-stored" {
		t.Errorf("profile region/project not used: %+v", s)
	}

	t.Setenv("CLEURA_REGION", "kna1")
	s = cfg.Resolve(Flags{})
	if s.Region != "kna1" {
		t.Errorf("env region should override profile, got %q", s.Region)
	}

	s = cfg.Resolve(Flags{Region: "fra1", ProjectID: "proj-flag"})
	if s.Region != "fra1" || s.ProjectID != "proj-flag" {
		t.Errorf("flags should override env and profile: %+v", s)
	}
}
