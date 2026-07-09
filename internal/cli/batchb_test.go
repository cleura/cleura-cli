package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestFlagPlacement locks in Batch B: --region/--project-id live only on the
// commands that use them (gardener subtree + login), and -o/--output only on
// commands that render — never as global no-ops.
func TestFlagPlacement(t *testing.T) {
	root := NewRootCommand("test")
	at := func(path ...string) *cobra.Command {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		return c
	}
	// c.Flag walks local + inherited persistent flags.
	has := func(c *cobra.Command, flag string) bool { return c.Flag(flag) != nil }

	// config view also accepts them — it previews how any setting resolves.
	regionUsers := [][]string{{"gardener", "shoot", "list"}, {"gardener", "shoot", "wake"}, {"login"}, {"config", "view"}}
	regionNonUsers := [][]string{{"whoami"}, {"logout"}, {"user", "list"}, {"config", "set"}}
	for _, p := range regionUsers {
		if !has(at(p...), "region") || !has(at(p...), "project-id") {
			t.Errorf("%v should have --region/--project-id", p)
		}
	}
	for _, p := range regionNonUsers {
		if has(at(p...), "region") || has(at(p...), "project-id") {
			t.Errorf("%v must NOT have --region/--project-id (was a silent no-op)", p)
		}
	}

	outputRenderers := [][]string{{"whoami"}, {"version"}, {"user", "list"}, {"user", "get"}, {"gardener", "shoot", "list"}, {"config", "view"}, {"config", "list-profiles"}}
	outputNonRenderers := [][]string{{"login"}, {"logout"}, {"config", "set"}, {"config", "use-profile"}, {"config", "delete-profile"}, {"config", "path"}, {"config", "get-credentials"}}
	for _, p := range outputRenderers {
		if !has(at(p...), "output") {
			t.Errorf("%v should have -o/--output", p)
		}
	}
	for _, p := range outputNonRenderers {
		if has(at(p...), "output") {
			t.Errorf("%v must NOT have -o/--output (ignored there)", p)
		}
	}

	// Genuinely global flags stay everywhere.
	for _, f := range []string{"profile", "cloud", "api-url", "debug", "quiet"} {
		if !has(at("whoami"), f) {
			t.Errorf("global flag --%s should be available on every command", f)
		}
	}
}
