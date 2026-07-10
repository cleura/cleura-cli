package cli

import (
	"io"
	"strings"
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

func TestFmtFloat(t *testing.T) {
	tests := map[float32]string{3.5: "3.5", 3.0: "3", 0: "0", 12.25: "12.25"}
	for in, want := range tests {
		if got := fmtFloat(in); got != want {
			t.Errorf("fmtFloat(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestPct(t *testing.T) {
	if got := pct(1, 4); got != "25.0%" {
		t.Errorf("pct(1,4) = %q, want 25.0%%", got)
	}
	if got := pct(3, 0); got != "-" {
		t.Errorf("pct with zero allocatable = %q, want - (no divide-by-zero)", got)
	}
}

func TestLatestSample(t *testing.T) {
	if got := latestSample(nil); got != "-" {
		t.Errorf("empty series = %q, want -", got)
	}
	series := []api.GardenerSample{{Value: "1"}, {Value: "2"}, {Value: "3"}}
	if got := latestSample(series); got != "3" {
		t.Errorf("latest sample = %q, want 3 (last element)", got)
	}
}

func TestBatchFCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"gardener", "shoot", "monitoring", "credentials"},
		{"gardener", "shoot", "monitoring", "nodes"},
		{"gardener", "shoot", "monitoring", "node"},
		{"gardener", "shoot", "monitoring", "worker-group"},
		{"gardener", "shoot", "ssh-key"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Errorf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
}

// TestSSHKeyRequiresExactlyOneDestination checks that the private key is never
// emitted without an explicit, single destination (guards against an
// accidental stdout leak). Both checks return before any network call.
func TestSSHKeyRequiresExactlyOneDestination(t *testing.T) {
	run := func(args ...string) error {
		root := NewRootCommand("test")
		root.SetArgs(append([]string{"gardener", "shoot", "ssh-key"}, args...))
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}
	// Neither destination.
	if err := run("myshoot"); err == nil || !strings.Contains(err.Error(), "exactly one of --file or --stdout") {
		t.Errorf("no destination: got %v, want the exactly-one error", err)
	}
	// Both destinations.
	if err := run("myshoot", "-f", "/tmp/does-not-matter", "--stdout"); err == nil || !strings.Contains(err.Error(), "exactly one of --file or --stdout") {
		t.Errorf("both destinations: got %v, want the exactly-one error", err)
	}
}
