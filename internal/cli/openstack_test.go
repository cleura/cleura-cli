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
	// More than one → an actionable error that points at --domain-id.
	_, err = chooseSoleDomain([]api.CommonOpenStackDomain{{Id: "dom-1"}, {Id: "dom-2"}})
	if err == nil || !strings.Contains(err.Error(), "--domain-id") {
		t.Errorf("multiple domains: want an error mentioning --domain-id, got %v", err)
	}
}

func TestRoleIDsByName(t *testing.T) {
	roles := []api.OpenStackIdentityProjectRole{
		{Id: "id-member", Name: "member"},
		{Id: "id-admin", Name: "admin"},
	}
	// Known names resolve to their IDs, in the order requested.
	ids, err := roleIDsByName(roles, []string{"admin", "member"})
	if err != nil || len(ids) != 2 || ids[0] != "id-admin" || ids[1] != "id-member" {
		t.Errorf("known roles: got (%v, %v), want [id-admin id-member]", ids, err)
	}
	// An unknown name errors, naming it and the available roles.
	_, err = roleIDsByName(roles, []string{"member", "nope"})
	if err == nil || !strings.Contains(err.Error(), `"nope"`) || !strings.Contains(err.Error(), "available roles") {
		t.Errorf("unknown role: want an error naming it and the available roles, got %v", err)
	}
}

func TestUserByNameOrID(t *testing.T) {
	users := []api.OpenStackIdentityUser{
		{Id: "id-alice", Name: "alice"},
		{Id: "id-bob", Name: "bob"},
	}
	// By name.
	if u, err := userByNameOrID(users, "bob"); err != nil || u.Id != "id-bob" {
		t.Errorf("by name: got (%v, %v), want id-bob", u, err)
	}
	// By ID.
	if u, err := userByNameOrID(users, "id-alice"); err != nil || u.Name != "alice" {
		t.Errorf("by id: got (%v, %v), want alice", u, err)
	}
	// Unknown.
	if _, err := userByNameOrID(users, "carol"); err == nil {
		t.Error("unknown user: expected an error, got nil")
	}
}

func TestOpenstackCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"openstack", "domain", "list"},
		{"openstack", "project", "list"},
		{"openstack", "project", "create"},
		{"openstack", "project", "edit"},
		{"openstack", "role", "list"},
		{"openstack", "role", "assignment", "create"},
		{"openstack", "role", "assignment", "list"},
		{"openstack", "role", "assignment", "delete"},
		{"openstack", "user", "list"},
		{"openstack", "user", "create"},
		{"openstack", "user", "delete"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Errorf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
}
