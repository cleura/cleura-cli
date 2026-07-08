package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// runCLI executes the command tree against a scratch config and returns
// stdout plus the error.
func runCLI(t *testing.T, configYAML string, args ...string) (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if configYAML != "" {
		if err := os.WriteFile(path, []byte(configYAML), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CLEURA_CONFIG", path)
	for _, v := range []string{"CLEURA_PROFILE", "CLEURA_CLOUD", "CLEURA_API_URL", "CLEURA_API_USERNAME", "CLEURA_API_TOKEN", "CLEURA_REGION", "CLEURA_PROJECT_ID"} {
		t.Setenv(v, "")
	}

	root := NewRootCommand("test")
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), err
}

func TestGetCredentialsEnvelope(t *testing.T) {
	out, err := runCLI(t, `version: 1
current_profile: work
profiles:
  work:
    cloud: compliant
    username: svc-user
    token: tok-123
    region: sto1
    project_id: proj-9
    token_stored_at: 2026-07-08T10:00:00Z
`, "config", "get-credentials")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, out)
	}
	want := map[string]any{
		"version":         float64(1),
		"profile":         "work",
		"cloud":           "compliant",
		"endpoint":        "https://rest.compliant.cleura.cloud",
		"username":        "svc-user",
		"token":           "tok-123",
		"region":          "sto1",
		"project_id":      "proj-9",
		"token_stored_at": "2026-07-08T10:00:00Z",
	}
	for key, expected := range want {
		if envelope[key] != expected {
			t.Errorf("envelope[%q] = %v, want %v", key, envelope[key], expected)
		}
	}
}

func TestGetCredentialsNoToken(t *testing.T) {
	out, err := runCLI(t, `profiles:
  default:
    username: someone
`, "config", "get-credentials")

	var coded *ExitCodeError
	if !errors.As(err, &coded) || coded.Code != 2 {
		t.Fatalf("want ExitCodeError code 2, got %v", err)
	}
	var payload map[string]string
	if jsonErr := json.Unmarshal([]byte(out), &payload); jsonErr != nil || payload["error"] == "" {
		t.Errorf("stdout should carry a JSON error object, got: %s", out)
	}
}

func TestGetCredentialsUnresolvableEndpointFallsThrough(t *testing.T) {
	// A profile whose endpoint cannot resolve (unknown cloud, no api_url)
	// cannot yield usable credentials: exit 2 for chain callers, not a
	// malfunction that would abort their credential chain.
	out, err := runCLI(t, `profiles:
  default:
    cloud: acme
    username: u
    token: tok
`, "config", "get-credentials")

	var coded *ExitCodeError
	if !errors.As(err, &coded) || coded.Code != 2 {
		t.Fatalf("want ExitCodeError code 2, got %v", err)
	}
	var payload map[string]string
	if jsonErr := json.Unmarshal([]byte(out), &payload); jsonErr != nil || payload["error"] == "" {
		t.Errorf("stdout should carry a JSON error object, got: %s", out)
	}
}

func TestGetCredentialsOmitsUnsetOptionalFields(t *testing.T) {
	out, err := runCLI(t, `profiles:
  default:
    cloud: public
    username: u
    token: tok
`, "config", "get-credentials")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"region", "project_id", "token_stored_at"} {
		if _, ok := envelope[absent]; ok {
			t.Errorf("unset optional field %q must be omitted, got %v", absent, envelope[absent])
		}
	}
}
