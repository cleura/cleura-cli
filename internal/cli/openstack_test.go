package cli

import (
	"strings"
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

func TestChooseSoleDomain(t *testing.T) {
	// No domains → error (nothing to create in).
	if _, err := chooseSoleDomain(nil); err == nil {
		t.Error("no domains: expected an error, got nil")
	}
	// Exactly one → its ID, no error (the auto-resolution path).
	got, err := chooseSoleDomain([]api.CommonOpenStackDomain{{Id: "dom-1"}})
	if err != nil || got != "dom-1" {
		t.Errorf("one domain: got (%q, %v), want (\"dom-1\", nil)", got, err)
	}
	// More than one → an actionable error that points at --domain.
	_, err = chooseSoleDomain([]api.CommonOpenStackDomain{{Id: "dom-1"}, {Id: "dom-2"}})
	if err == nil || !strings.Contains(err.Error(), "--domain") {
		t.Errorf("multiple domains: want an error mentioning --domain, got %v", err)
	}
}

func TestOpenstackCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"openstack", "domain", "list"},
		{"openstack", "project", "list"},
		{"openstack", "project", "create"},
		{"openstack", "project", "edit"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Errorf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
}
