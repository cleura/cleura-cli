package cli

import (
	"io"
	"strings"
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

// TestShootHA covers the derived high-availability string: there is no literal
// HA bool in the API, so it is derived from the optional ControlPlane block.
func TestShootHA(t *testing.T) {
	tests := []struct {
		name string
		s    api.GardenerShootShoot
		want string
	}{
		{"no control plane", api.GardenerShootShoot{}, "none"},
		{"zone tolerance", api.GardenerShootShoot{ControlPlane: &api.GardenerShootControlPlane{
			HighAvailability: api.GardenerShootHighAvailability{
				FailureTolerance: api.GardenerShootFailureTolerance{Type: api.GardenerShootFailureToleranceType("zone")},
			},
		}}, "zone"},
		{"present but no tolerance type", api.GardenerShootShoot{ControlPlane: &api.GardenerShootControlPlane{}}, "enabled"},
	}
	for _, tt := range tests {
		if got := shootHA(tt.s); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestNewWorkerGroupView(t *testing.T) {
	w := api.GardenerShootWorker{
		Name:    "default",
		Machine: api.GardenerShootMachine{Type: "b.2c4gb", Image: api.GardenerShootMachineImage{Name: "gardenlinux", Version: "1443.3"}},
		Zones:   []string{"nova", "nova2"},
	}
	v := newWorkerGroupView(w)
	if v.MachineType != "b.2c4gb" {
		t.Errorf("MachineType: got %q, want b.2c4gb", v.MachineType)
	}
	if v.Image != "gardenlinux 1443.3" {
		t.Errorf("Image: got %q, want %q", v.Image, "gardenlinux 1443.3")
	}
	if v.zonesText != "nova,nova2" {
		t.Errorf("zonesText: got %q, want nova,nova2", v.zonesText)
	}
	// The raw worker is embedded, so its fields survive into -o json.
	if v.Name != "default" {
		t.Errorf("embedded Name lost: got %q", v.Name)
	}
}

func TestCheckNameViewAvailability(t *testing.T) {
	taken := checkNameView{GardenerIsShootNameTaken: api.GardenerIsShootNameTaken{IsTaken: true}, Name: "prod", Available: false}
	if taken.Available {
		t.Error("a taken name must not be available")
	}
	free := checkNameView{GardenerIsShootNameTaken: api.GardenerIsShootNameTaken{IsTaken: false}, Name: "prod", Available: true}
	if !free.Available {
		t.Error("an untaken name must be available")
	}
}

// TestBatchBCommandsWired locks in that the new read commands are registered
// where the roadmap places them.
func TestBatchBCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"gardener", "shoot", "get"},
		{"gardener", "shoot", "check-name"},
		{"gardener", "worker-group", "list"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Errorf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
}

// TestCheckNameIsCloudScoped: check-name is a cloud-only command, so unlike the
// project-scoped gardener commands it must NOT claim the region/project
// prerequisite in its help; the project-scoped reads still must.
func TestCheckNameIsCloudScoped(t *testing.T) {
	root := NewRootCommand("test")
	const projectReq = "region and project must be selected"

	checkName, _, err := root.Find([]string{"gardener", "shoot", "check-name"})
	if err != nil {
		t.Fatalf("find check-name: %v", err)
	}
	if strings.Contains(checkName.Long, projectReq) {
		t.Errorf("check-name is cloud-only; help should not require region/project:\n%s", checkName.Long)
	}

	for _, path := range [][]string{{"gardener", "shoot", "get"}, {"gardener", "worker-group", "list"}} {
		c, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		if !strings.Contains(c.Long, projectReq) {
			t.Errorf("%v is project-scoped; help must state the requirement:\n%s", path, c.Long)
		}
	}
}

// TestCheckNameRejectsProjectFlags: check-name inherits --region/--project-id
// from the gardener parent but is cloud-only, so it must reject them rather
// than silently ignore them (the CLI treats a silent no-op flag as a bug).
func TestCheckNameRejectsProjectFlags(t *testing.T) {
	run := func(args ...string) error {
		root := NewRootCommand("test")
		root.SetArgs(append([]string{"gardener", "shoot", "check-name", "prod"}, args...))
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		return root.Execute()
	}
	for _, flag := range []string{"--region", "--project-id"} {
		err := run(flag, "x")
		if err == nil || !strings.Contains(err.Error(), "does not apply") {
			t.Errorf("check-name %s x: got %v, want a 'does not apply' rejection", flag, err)
		}
	}
}
