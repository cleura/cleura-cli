package cli

import (
	"testing"

	api "github.com/cleura/cleura-client-go/api"
)

func privilege(t api.UserUserLoginPrivilegeType, projects ...api.CommonUserLoginProjectPrivilege) *api.CommonUserLoginPrivilege {
	p := &api.CommonUserLoginPrivilege{Type: t}
	if len(projects) > 0 {
		p.ProjectPrivileges = &projects
	}
	return p
}

func TestRolesSummary(t *testing.T) {
	if got := rolesSummary(api.CommonUserLoginPrivileges{}); got != "-" {
		t.Errorf("empty summary = %q", got)
	}

	got := rolesSummary(api.CommonUserLoginPrivileges{
		Invoice:   privilege(api.Read),
		Openstack: privilege(api.Project, api.CommonUserLoginProjectPrivilege{ProjectId: "p1", Type: api.Full}, api.CommonUserLoginProjectPrivilege{ProjectId: "p2", Type: api.Read}),
		Users:     privilege(api.Full),
	})
	want := "invoice:read openstack:project(2) users:full"
	if got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}

	// Every area at full access compresses (the typical administrator).
	full := api.CommonUserLoginPrivileges{
		Account: privilege(api.Full), AiGateway: privilege(api.Full),
		Application: privilege(api.Full), Invoice: privilege(api.Full),
		Monitoring: privilege(api.Full), Openstack: privilege(api.Full),
		Users: privilege(api.Full),
	}
	if got := rolesSummary(full); got != "full (all areas)" {
		t.Errorf("all-full summary = %q", got)
	}
}

func TestPrivilegeLabel(t *testing.T) {
	if got := privilegeLabel(privilege(api.Full)); got != "Full Access" {
		t.Errorf("full = %q", got)
	}
	if got := privilegeLabel(privilege(api.Project, api.CommonUserLoginProjectPrivilege{}, api.CommonUserLoginProjectPrivilege{})); got != "Project Access (2 projects)" {
		t.Errorf("project = %q", got)
	}
}

func TestTwoFactorSummary(t *testing.T) {
	if got := twoFactorSummary(nil); got != "none" {
		t.Errorf("nil = %q", got)
	}
	keys := []api.CommonTwoFactorLoginWebauthn{{Name: "yubikey"}, {Name: "backup"}}
	got := twoFactorSummary(&api.CommonUserLoginTwoFactorLogin{
		Sms:      &api.CommonTwoFactorLoginSms{},
		Webauthn: &keys,
	})
	if got != "sms, webauthn (2 keys)" {
		t.Errorf("summary = %q", got)
	}
}

func TestDisplayName(t *testing.T) {
	s := func(v string) *string { return &v }
	if got := displayName(nil, s("Doe")); got != "Doe" {
		t.Errorf("last-only = %q (leading-space regression)", got)
	}
	if got := displayName(s("Jane"), s("Doe")); got != "Jane Doe" {
		t.Errorf("full = %q", got)
	}
	if got := displayName(nil, nil); got != "" {
		t.Errorf("empty = %q", got)
	}
}
