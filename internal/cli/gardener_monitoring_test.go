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

func TestFmtCores(t *testing.T) {
	if got := fmtCores(0.08366667); got != "0.084" {
		t.Errorf("fmtCores(0.08366667) = %q, want 0.084", got)
	}
	if got := fmtCores(1.8463334); got != "1.846" {
		t.Errorf("fmtCores(1.8463334) = %q, want 1.846", got)
	}
}

func TestBytesToGiB(t *testing.T) {
	// 998195200 bytes ≈ 0.93 GiB (the overview endpoint reports bytes).
	if got := bytesToGiB(998195200); got != "0.93" {
		t.Errorf("bytesToGiB(998195200) = %q, want 0.93", got)
	}
	if got := bytesToGiB(0); got != "0.00" {
		t.Errorf("bytesToGiB(0) = %q, want 0.00", got)
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

func TestFmtPercent(t *testing.T) {
	if got := fmtPercent(4.391667); got != "4.4%" {
		t.Errorf("fmtPercent(4.391667) = %q, want 4.4%%", got)
	}
	if got := fmtPercent(33.35398); got != "33.4%" {
		t.Errorf("fmtPercent(33.35398) = %q, want 33.4%%", got)
	}
}

func TestLatestWithUnit(t *testing.T) {
	sample := func(v string) []api.GardenerSample { return []api.GardenerSample{{Value: v}} }
	tests := []struct {
		samples []api.GardenerSample
		unit    string
		want    string
	}{
		// Rounded to a consistent per-unit precision: GiB 2dp, cores 3dp, % 1dp.
		{sample("0.8327"), "GiB", "0.83 GiB"},
		{sample("1.25"), "GiB", "1.25 GiB"},
		{sample("0.1041"), "cores", "0.104 cores"},
		{sample("0.02"), "cores", "0.020 cores"},
		{sample("6.2074"), "%", "6.2%"}, // percent takes no leading space
		{sample("4.39"), "%", "4.4%"},
		{nil, "GiB", "-"},                     // empty series: no unit appended
		{sample("n/a"), "cores", "n/a cores"}, // non-numeric: raw value, unit appended
	}
	for _, tt := range tests {
		if got := latestWithUnit(tt.samples, tt.unit); got != tt.want {
			t.Errorf("latestWithUnit(%v, %q) = %q, want %q", tt.samples, tt.unit, got, tt.want)
		}
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
