package cli

import (
	"io"
	"strings"
	"testing"
	"time"

	api "github.com/cleura/cleura-client-go/api"
)

func TestBatchACommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"gardener", "cloud-profile"},
		{"gardener", "cloud-profile", "list"},
		{"gardener", "cloud-profile", "show"},
		{"gardener", "project"},
		{"gardener", "project", "bootstrap"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Errorf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
}

func TestGroupVersions(t *testing.T) {
	sup := api.Supported
	dep := api.K8sVersionClassification("deprecated")
	versions := []api.GardenerCloudProfileKubernetesVersion{
		{Version: "1.30.1", Classification: &sup},
		{Version: "1.29.5", Classification: &dep},
		{Version: "1.31.0"}, // nil classification → supported
		{Version: "1.29.8", Classification: &dep},
	}
	groups := groupVersions(classifyK8s(versions))
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2 (supported, deprecated)", len(groups))
	}
	// Supported group first, preserving API order; nil + explicit Supported.
	if groups[0].label != "supported" || strings.Join(groups[0].versions, ",") != "1.30.1,1.31.0" {
		t.Errorf("supported group = %+v, want [1.30.1 1.31.0]", groups[0])
	}
	if groups[1].label != "deprecated" || strings.Join(groups[1].versions, ",") != "1.29.5,1.29.8" {
		t.Errorf("deprecated group = %+v, want [1.29.5 1.29.8]", groups[1])
	}
	// The list command's usable count still sees nil + Supported (2 of 4).
	if n := len(supportedK8sVersions(versions)); n != 2 {
		t.Errorf("supportedK8sVersions count = %d, want 2", n)
	}
}

func TestGroupVersionsWithExpiry(t *testing.T) {
	sup := api.Supported
	dep := api.K8sVersionClassification("deprecated")
	mar := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	may := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	versions := []api.GardenerCloudProfileKubernetesVersion{
		{Version: "1.35.6", Classification: &sup},                       // supported, no expiry
		{Version: "1.34.8", Classification: &dep, ExpirationDate: &may}, // later
		{Version: "1.33.9", Classification: &dep, ExpirationDate: &mar}, // sooner
		{Version: "1.33.10", Classification: &dep, ExpirationDate: &mar},
	}
	groups := groupVersions(classifyK8s(versions))
	if len(groups) != 3 {
		t.Fatalf("groups = %d, want 3", len(groups))
	}
	// supported first, no expiry suffix.
	if groups[0].label != "supported" || strings.Join(groups[0].versions, ",") != "1.35.6" {
		t.Errorf("group 0 = %+v, want supported [1.35.6]", groups[0])
	}
	// Soonest expiry next, versions sharing the date merged.
	if groups[1].label != "deprecated (expires 2026-03-01)" || strings.Join(groups[1].versions, ",") != "1.33.9,1.33.10" {
		t.Errorf("group 1 = %+v, want deprecated 2026-03-01 [1.33.9 1.33.10]", groups[1])
	}
	// Later expiry after.
	if groups[2].label != "deprecated (expires 2026-05-01)" || strings.Join(groups[2].versions, ",") != "1.34.8" {
		t.Errorf("group 2 = %+v, want deprecated 2026-05-01 [1.34.8]", groups[2])
	}
}

func TestMachineTypeLabel(t *testing.T) {
	usable := api.GardenerCloudProfileMachineType{Cpu: 4, Memory: "8Gi", Architecture: "amd64", Usable: true}
	if got, want := machineTypeLabel(usable), "4 vCPU, 8Gi RAM, amd64"; got != want {
		t.Errorf("machineTypeLabel(usable) = %q, want %q", got, want)
	}
	unusable := api.GardenerCloudProfileMachineType{Cpu: 2, Memory: "4Gi", Usable: false}
	if got := machineTypeLabel(unusable); !strings.Contains(got, "(unusable)") {
		t.Errorf("machineTypeLabel(unusable) = %q, want it to flag (unusable)", got)
	}
	if n := usableMachineTypeCount([]api.GardenerCloudProfileMachineType{usable, unusable}); n != 1 {
		t.Errorf("usableMachineTypeCount = %d, want 1", n)
	}
}

// TestCloudProfileIsCloudScoped: cloud-profile list/show are cloud-only, so
// their help must not claim the region/project prerequisite, and they reject
// those flags if explicitly passed (rather than silently ignoring them).
func TestCloudProfileIsCloudScoped(t *testing.T) {
	root := NewRootCommand("test")
	const projectReq = "region and project must be selected"
	for _, path := range [][]string{
		{"gardener", "cloud-profile"},
		{"gardener", "cloud-profile", "list"},
		{"gardener", "cloud-profile", "show"},
	} {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		if strings.Contains(c.Long, projectReq) {
			t.Errorf("%v is cloud-only; help must not require region/project:\n%s", path, c.Long)
		}
	}

	run := func(args ...string) error {
		r := NewRootCommand("test")
		r.SetArgs(args)
		r.SetOut(io.Discard)
		r.SetErr(io.Discard)
		return r.Execute()
	}
	if err := run("gardener", "cloud-profile", "list", "--region", "sto1"); err == nil || !strings.Contains(err.Error(), "does not apply") {
		t.Errorf("cloud-profile list --region: got %v, want a 'does not apply' rejection", err)
	}
	if err := run("gardener", "cloud-profile", "show", "x", "--project-id", "p"); err == nil || !strings.Contains(err.Error(), "does not apply") {
		t.Errorf("cloud-profile show --project-id: got %v, want a 'does not apply' rejection", err)
	}
}
