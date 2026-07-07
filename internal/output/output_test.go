package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderYAMLUsesJSONFieldNames(t *testing.T) {
	// The generated API models carry only json tags; YAML output must use
	// the same names as JSON, and honor omitempty, via the JSON round-trip.
	type model struct {
		AuthProviderId int     `json:"auth_provider_id"`
		Firstname      *string `json:"firstname,omitempty"`
	}

	var buf bytes.Buffer
	if err := Render(&buf, "yaml", model{AuthProviderId: 7}, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "auth_provider_id: 7") {
		t.Errorf("yaml should use json field names, got:\n%s", out)
	}
	if strings.Contains(out, "firstname") {
		t.Errorf("yaml should honor omitempty, got:\n%s", out)
	}
}

func TestValid(t *testing.T) {
	for _, ok := range []string{"", "table", "json", "yaml"} {
		if !Valid(ok) {
			t.Errorf("Valid(%q) = false", ok)
		}
	}
	if Valid("junk") {
		t.Error("Valid(junk) = true")
	}
}
