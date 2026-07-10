package cli

import (
	"strings"
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

// TestGardenerHelpStatesProjectRequirement locks in that every project-scoped
// gardener command states the region/project prerequisite in its help text
// (not just by listing the flags).
func TestGardenerHelpStatesProjectRequirement(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"gardener"},
		{"gardener", "shoot"},
		{"gardener", "shoot", "list"},
		{"gardener", "shoot", "get"},
		{"gardener", "shoot", "kubeconfig"},
		{"gardener", "shoot", "wake"},
		{"gardener", "shoot", "hibernate"},
		{"gardener", "shoot", "reconcile"},
		{"gardener", "shoot", "maintain"},
		{"gardener", "shoot", "retry"},
		{"gardener", "shoot", "enable-ha"},
		{"gardener", "shoot", "ca"},
		{"gardener", "shoot", "ca", "rotate"},
		{"gardener", "shoot", "ca", "status"},
		{"gardener", "shoot", "monitoring"},
		{"gardener", "shoot", "monitoring", "credentials"},
		{"gardener", "shoot", "monitoring", "nodes"},
		{"gardener", "shoot", "monitoring", "node"},
		{"gardener", "shoot", "monitoring", "worker-group"},
		{"gardener", "shoot", "ssh-key"},
		{"gardener", "worker-group"},
		{"gardener", "worker-group", "list"},
	} {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		help := c.Long
		if !strings.Contains(help, "region and project must be selected") {
			t.Errorf("%v help does not state the region/project requirement:\n%s", path, help)
		}
	}
}

func TestShootStatusSummary(t *testing.T) {
	op := func(state string, progress int) *api.GardenerShootLastOperation {
		return &api.GardenerShootLastOperation{State: api.GardenerShootLastOperationState(state), Type: "Reconcile", Progress: progress}
	}
	hibernated := &api.GardenerShootShootStatus{IsHibernated: true}

	tests := []struct {
		name string
		s    api.GardenerShootShoot
		want string
	}{
		{"in-flight beats hibernation flag", api.GardenerShootShoot{LastOperation: op("Processing", 30), Status: hibernated}, "Processing (Reconcile 30%)"},
		{"hibernated when idle", api.GardenerShootShoot{Status: hibernated}, "hibernated"},
		{"succeeded reads as ready", api.GardenerShootShoot{LastOperation: op("Succeeded", 100)}, "ready"},
		{"other terminal state kept raw", api.GardenerShootShoot{LastOperation: op("Failed", 100)}, "Failed"},
		{"no operation, not hibernated", api.GardenerShootShoot{}, ""},
	}
	for _, tt := range tests {
		if got := shootStatusSummary(tt.s); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
