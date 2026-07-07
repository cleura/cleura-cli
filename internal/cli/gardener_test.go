package cli

import (
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

func TestVersionLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.35.6", "1.36.0", true},
		{"1.35.6", "1.35.10", true}, // numeric, not lexicographic
		{"1.36.0", "1.35.6", false},
		{"1.35.6", "1.35.6", false},
		{"1.35", "1.35.6", true},
	}
	for _, tt := range tests {
		if got := versionLess(tt.a, tt.b); got != tt.want {
			t.Errorf("versionLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestUpgradeAvailable(t *testing.T) {
	c := func(cl api.K8sVersionClassification) *api.K8sVersionClassification { return &cl }
	versions := []api.GardenerCloudProfileKubernetesVersion{
		{Version: "1.34.9", Classification: c(api.Deprecated)},
		{Version: "1.35.6", Classification: c(api.Supported)},
		{Version: "1.35.10", Classification: c(api.Supported)},
		{Version: "1.36.2", Classification: c(api.Supported)},
		{Version: "1.37.0", Classification: c(api.Preview)},
	}

	if got := upgradeAvailable("1.35.6", versions); got != "1.36.2" {
		t.Errorf("upgradeAvailable(1.35.6) = %q, want 1.36.2 (preview must not win)", got)
	}
	if got := upgradeAvailable("1.36.2", versions); got != "-" {
		t.Errorf("upgradeAvailable(1.36.2) = %q, want - (already newest supported)", got)
	}
	if got := upgradeAvailable("1.33.0", []api.GardenerCloudProfileKubernetesVersion{}); got != "-" {
		t.Errorf("upgradeAvailable with no versions = %q, want -", got)
	}
}
