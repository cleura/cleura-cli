package cli

import (
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

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
