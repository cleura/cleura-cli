package cli

import (
	"strings"
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

func TestCARotateStage(t *testing.T) {
	tests := []struct {
		in    string
		want  api.K8sCaRotationStage
		valid bool
	}{
		{"prepare", api.Prepare, true},
		{"PREPARE", api.Prepare, true},
		{"start", api.Prepare, true}, // alias
		{"complete", api.Complete, true},
		{"Complete", api.Complete, true},
		{"", "", false},
		{"preparing", "", false}, // read-only state, not an input
		{"prepared", "", false},
		{"completed", "", false},
		{"garbage", "", false},
	}
	for _, tt := range tests {
		got, ok := caRotateStage(tt.in)
		if ok != tt.valid || (ok && got != tt.want) {
			t.Errorf("caRotateStage(%q) = (%q, %v), want (%q, %v)", tt.in, got, ok, tt.want, tt.valid)
		}
	}
}

// TestCANextAction covers all seven rotation stages so 'ca status' always
// prints a concrete next step (or the raw stage as a fallback).
func TestCANextAction(t *testing.T) {
	tests := []struct {
		stage    api.K8sCaRotationStage
		contains string
	}{
		{api.NotInitiated, "--stage prepare"},
		{api.Preparing, "in progress"},
		{api.Prepared, "--stage complete"},
		{api.Completing, "in progress"},
		{api.Completed, "no action needed"},
		{api.Prepare, "accepted"},
		{api.Complete, "accepted"},
	}
	for _, tt := range tests {
		got := caNextAction("prod", tt.stage)
		if !strings.Contains(got, tt.contains) {
			t.Errorf("caNextAction(%q) = %q, want it to contain %q", tt.stage, got, tt.contains)
		}
	}
	// Unknown stage falls back to the raw string, never empty.
	if got := caNextAction("prod", api.K8sCaRotationStage("Weird")); got != "Weird" {
		t.Errorf("unknown stage: got %q, want raw passthrough", got)
	}
}

// TestBatchDCommandsWired locks in placement and that the destructive commands
// expose --yes while the plain actions do not.
func TestBatchDCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"gardener", "shoot", "maintain"},
		{"gardener", "shoot", "retry"},
		{"gardener", "shoot", "enable-ha"},
		{"gardener", "shoot", "ca", "rotate"},
		{"gardener", "shoot", "ca", "status"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Fatalf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}

	// enable-ha and ca rotate are destructive -> --yes; maintain/retry are not.
	hasYes := func(path ...string) bool {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		return c.Flags().Lookup("yes") != nil
	}
	if !hasYes("gardener", "shoot", "enable-ha") {
		t.Error("enable-ha must expose --yes (destructive)")
	}
	if !hasYes("gardener", "shoot", "ca", "rotate") {
		t.Error("ca rotate must expose --yes (destructive)")
	}
	if hasYes("gardener", "shoot", "maintain") {
		t.Error("maintain must not expose --yes (not destructive)")
	}
	if hasYes("gardener", "shoot", "retry") {
		t.Error("retry must not expose --yes (not destructive)")
	}
}
