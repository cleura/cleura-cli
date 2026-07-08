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
	if got := rolesSummary(true, api.CommonUserLoginPrivileges{}); got != "admin" {
		t.Errorf("admin summary = %q", got)
	}
	if got := rolesSummary(false, api.CommonUserLoginPrivileges{}); got != "-" {
		t.Errorf("empty summary = %q", got)
	}

	got := rolesSummary(false, api.CommonUserLoginPrivileges{
		Invoice:   privilege(api.Read),
		Openstack: privilege(api.Project, api.CommonUserLoginProjectPrivilege{ProjectId: "p1", Type: api.Full}, api.CommonUserLoginProjectPrivilege{ProjectId: "p2", Type: api.Read}),
		Users:     privilege(api.Full),
	})
	want := "invoice:read openstack:project(2) users:full"
	if got != want {
		t.Errorf("summary = %q, want %q", got, want)
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
