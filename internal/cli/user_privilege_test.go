package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/cleura/cleura-cli/internal/config"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

// The flag tokens and the display list are two hand-written lists of the same
// seven areas; if they drift, --area silently stops addressing an area the
// list column still shows.
func TestPrivilegeAreaTokensMatch(t *testing.T) {
	var p api.CommonUserLoginPrivileges
	fields := privilegeFields(&p)
	tokens := privilegeAreaTokens()
	if len(fields) != len(tokens) {
		t.Fatalf("privilegeFields has %d areas, privilegeAreaTokens has %d", len(fields), len(tokens))
	}
	for _, token := range tokens {
		if _, ok := fields[token]; !ok {
			t.Errorf("area %q is displayed but not addressable by --area", token)
		}
	}
}

func TestParsePrivilegeLevel(t *testing.T) {
	for in, want := range map[string]api.UserUserLoginPrivilegeType{"full": api.Full, "READ": api.Read, " read ": api.Read} {
		got, err := parsePrivilegeLevel(in)
		if err != nil || got != want {
			t.Errorf("parse %q = %q, %v", in, got, err)
		}
	}
	if _, err := parsePrivilegeLevel("project"); err == nil || !strings.Contains(err.Error(), "not a level") {
		t.Errorf("project: %v", err)
	}
	if _, err := parsePrivilegeLevel("admin"); err == nil || !strings.Contains(err.Error(), "want full or read") {
		t.Errorf("admin: %v", err)
	}
}

// base is a user with per-project OpenStack access (carrying meta the CLI
// cannot express) plus an unrelated account-wide area.
func basePrivileges() api.CommonUserLoginPrivileges {
	meta := "granted by support"
	return api.CommonUserLoginPrivileges{
		Openstack: &api.CommonUserLoginPrivilege{
			Type: api.Project,
			Meta: &meta,
			ProjectPrivileges: &[]api.CommonUserLoginProjectPrivilege{
				{ProjectId: "aaaa1111", DomainId: "dddd", Type: api.Full},
				{ProjectId: "bbbb2222", DomainId: "dddd", Type: api.Read},
			},
		},
		Invoice: privilege(api.Read),
	}
}

// The property that matters: touching one area must leave every other area
// byte-for-byte intact, including fields this CLI cannot express.
func TestSetAreaPrivilegeLeavesOtherAreasIntact(t *testing.T) {
	base := basePrivileges()
	got, replaced := setAreaPrivilege(base, "invoice", api.Full)

	if got.Invoice.Type != api.Full {
		t.Errorf("invoice = %q, want full", got.Invoice.Type)
	}
	if replaced != 0 {
		t.Errorf("replaced = %d, want 0 (invoice had no project grants)", replaced)
	}
	if got.Openstack.Meta == nil || *got.Openstack.Meta != "granted by support" {
		t.Error("an untouched area lost its meta")
	}
	if len(existingProjectGrants(got.Openstack)) != 2 {
		t.Error("an untouched area lost its per-project grants")
	}
	if got.Users != nil || got.Account != nil {
		t.Error("privileges were invented for areas the user does not have")
	}
	if base.Invoice.Type != api.Read {
		t.Error("the caller's privileges object was mutated")
	}
}

// Setting an area account-wide replaces its project grants. That is intended,
// but it must be reported so the command can say so.
func TestSetAreaPrivilegeReportsDiscardedGrants(t *testing.T) {
	got, replaced := setAreaPrivilege(basePrivileges(), "openstack", api.Read)
	if replaced != 2 {
		t.Errorf("replaced = %d, want 2", replaced)
	}
	if got.Openstack.Type != api.Read {
		t.Errorf("type = %q, want read", got.Openstack.Type)
	}
	if got.Openstack.ProjectPrivileges != nil {
		t.Error("account-wide access must not keep a project list")
	}
	if got.Openstack.Meta == nil {
		t.Error("meta should survive a level change")
	}
}

func TestSetProjectPrivilege(t *testing.T) {
	// Adding a project keeps the existing ones.
	got, narrowed := setProjectPrivilege(basePrivileges(), "cccc3333", "dddd", api.Read)
	if narrowed != "" {
		t.Errorf("narrowed = %q, want empty (already project-scoped)", narrowed)
	}
	grants := existingProjectGrants(got.Openstack)
	if len(grants) != 3 {
		t.Fatalf("grants = %d, want 3", len(grants))
	}

	// Re-granting an existing project replaces that grant, not appends.
	got, _ = setProjectPrivilege(got, "aaaa1111", "dddd", api.Read)
	grants = existingProjectGrants(got.Openstack)
	if len(grants) != 3 {
		t.Fatalf("re-grant appended instead of replacing: %d grants", len(grants))
	}
	idx := slices.IndexFunc(grants, func(g api.CommonUserLoginProjectPrivilege) bool { return g.ProjectId == "aaaa1111" })
	if grants[idx].Type != api.Read {
		t.Errorf("re-granted level = %q, want read", grants[idx].Type)
	}

	// Dashes in an ID must not create a duplicate grant.
	got, _ = setProjectPrivilege(got, "AAAA-1111", "dddd", api.Full)
	if n := len(existingProjectGrants(got.Openstack)); n != 3 {
		t.Errorf("a dashed/uppercase ID created a duplicate: %d grants", n)
	}
}

// Granting on a project when the user had account-wide access narrows their
// access, which the command has to surface.
func TestSetProjectPrivilegeReportsNarrowing(t *testing.T) {
	base := api.CommonUserLoginPrivileges{Openstack: privilege(api.Full)}
	got, narrowed := setProjectPrivilege(base, "aaaa1111", "dddd", api.Read)
	if narrowed != api.Full {
		t.Errorf("narrowed = %q, want full", narrowed)
	}
	if got.Openstack.Type != api.Project || len(existingProjectGrants(got.Openstack)) != 1 {
		t.Errorf("result = %+v", got.Openstack)
	}
}

func TestUnsetAreaPrivilege(t *testing.T) {
	got, existed := unsetAreaPrivilege(basePrivileges(), "invoice")
	if !existed || got.Invoice != nil {
		t.Errorf("existed=%v invoice=%v", existed, got.Invoice)
	}
	if got.Openstack == nil {
		t.Error("unset touched an area it was not given")
	}
	if _, existed := unsetAreaPrivilege(basePrivileges(), "users"); existed {
		t.Error("unset reported success for an area with no access")
	}
}

func TestUnsetProjectPrivilege(t *testing.T) {
	got, removed, areaGone := unsetProjectPrivilege(basePrivileges(), "aaaa1111")
	if !removed || areaGone {
		t.Fatalf("removed=%v areaGone=%v", removed, areaGone)
	}
	if n := len(existingProjectGrants(got.Openstack)); n != 1 {
		t.Errorf("grants = %d, want 1", n)
	}
	if got.Openstack.Meta == nil {
		t.Error("meta lost when removing one grant")
	}

	// Removing the last grant removes the area: "project access to no
	// projects" is not a meaningful state.
	got, removed, areaGone = unsetProjectPrivilege(got, "bbbb2222")
	if !removed || !areaGone || got.Openstack != nil {
		t.Errorf("last grant: removed=%v areaGone=%v openstack=%v", removed, areaGone, got.Openstack)
	}

	if _, removed, _ := unsetProjectPrivilege(basePrivileges(), "9999"); removed {
		t.Error("reported removing a grant that was not there")
	}
}

func TestPrivilegeRows(t *testing.T) {
	names := map[string]string{"aaaa1111": "team-alpha", "bbbb2222": "team-beta"}
	rows := privilegeRows(basePrivileges(), func(id string) string { return names[stripUUID(id)] })
	want := []privilegeRow{
		{Area: "invoice", Level: "read", Scope: "account-wide"},
		{Area: "openstack", Level: "project", Scope: "team-alpha:full, team-beta:read"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %+v", rows)
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
	if got := privilegeRows(api.CommonUserLoginPrivileges{}, func(s string) string { return s }); len(got) != 0 {
		t.Errorf("no privileges should render no rows, got %+v", got)
	}

	// Project scope with an empty grant list grants nothing. This CLI never
	// writes that state (unsetting the last grant removes the area), but the
	// Control Panel or another client can, so it must render as what it is
	// rather than look like account-wide access.
	empty := api.CommonUserLoginPrivileges{Openstack: &api.CommonUserLoginPrivilege{Type: api.Project}}
	got := privilegeRows(empty, func(s string) string { return s })
	if len(got) != 1 || got[0].Scope != "no projects" {
		t.Errorf("empty project scope = %+v, want scope %q", got, "no projects")
	}
}

// stripUUID is load-bearing for both duplicate detection and grant removal, so
// assert the output, not just that two chosen inputs agree.
func TestStripUUID(t *testing.T) {
	for in, want := range map[string]string{
		" 8A22-C50F ": "8a22c50f",
		"8a22c50f":    "8a22c50f",
		"AAAA-1111":   "aaaa1111",
		"":            "",
	} {
		if got := stripUUID(in); got != want {
			t.Errorf("stripUUID(%q) = %q, want %q", in, got, want)
		}
	}
	// Different IDs must not normalize onto each other.
	if stripUUID("aaaa-1111") == stripUUID("bbbb-2222") {
		t.Error("distinct IDs collided after normalization")
	}
}

func TestPrivilegeTarget(t *testing.T) {
	newSet := func(t *testing.T) *cobra.Command {
		t.Helper()
		root := NewRootCommand("test")
		c, _, err := root.Find([]string{"user", "privilege", "set"})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}

	// Neither flag.
	if _, _, err := privilegeTarget(newSet(t), "", ""); err == nil || !strings.Contains(err.Error(), "either --area") {
		t.Errorf("neither: %v", err)
	}

	// Both flags.
	c := newSet(t)
	_ = c.Flags().Set("area", "openstack")
	_ = c.Flags().Set("project-id", "p1")
	if _, _, err := privilegeTarget(c, "openstack", "p1"); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("both: %v", err)
	}

	// Area is canonicalized, and an unknown one is named.
	c = newSet(t)
	_ = c.Flags().Set("area", "AI_Gateway")
	kind, area, err := privilegeTarget(c, "AI_Gateway", "")
	if err != nil || kind != targetArea || area != "ai-gateway" {
		t.Errorf("canonical area: kind=%v area=%q err=%v", kind, area, err)
	}
	c = newSet(t)
	_ = c.Flags().Set("area", "storage")
	if _, _, err := privilegeTarget(c, "storage", ""); err == nil || !strings.Contains(err.Error(), "unknown privilege area") {
		t.Errorf("unknown area: %v", err)
	}
}

// fakeProjectAPI serves the one project listing the API offers, with a name
// deliberately duplicated across two regions.
func fakeProjectAPI(t *testing.T) *cleura.Client {
	t.Helper()
	region := func(tag string, projects ...map[string]any) map[string]any {
		return map[string]any{
			"region": map[string]any{
				"id": 1, "name": tag, "tag": tag, "object_storage_enabled": true,
				"dns_enabled": true, "backup_enabled": true, "network_provider": "n",
			},
			"projects": projects,
		}
	}
	project := func(id, name, domain string) map[string]any {
		return map[string]any{"id": id, "name": name, "domain_id": domain, "enabled": true}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /openstack/identity/v1/current-user/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]any{
			region("sto2", project("aaaa1111", "team-alpha", "dom-sto"), project("cccc3333", "sandbox", "dom-sto")),
			region("kna1", project("bbbb2222", "team-alpha", "dom-kna")),
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client, err := cleura.NewClientWithCredentials(srv.URL, "u", "t")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestResolveProjectRef(t *testing.T) {
	client := fakeProjectAPI(t)
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	settings := config.Settings{ProfileName: "p"}

	// Unique name resolves to its project and domain.
	id, domain, err := resolveProjectRef(cmd, settings, client, "sandbox", "")
	if err != nil || id != "cccc3333" || domain != "domsto" {
		t.Fatalf("unique name: %q %q %v", id, domain, err)
	}

	// A name used in two regions is refused, not guessed, and both candidates
	// are named. Granting the wrong project is not recoverable.
	_, _, err = resolveProjectRef(cmd, settings, client, "team-alpha", "")
	if err == nil || !strings.Contains(err.Error(), "matches 2 projects") {
		t.Fatalf("ambiguous name: %v", err)
	}
	for _, want := range []string{"sto2/team-alpha", "kna1/team-alpha"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ambiguity error should name %s: %v", want, err)
		}
	}

	// region/name disambiguates.
	id, domain, err = resolveProjectRef(cmd, settings, client, "kna1/team-alpha", "")
	if err != nil || id != "bbbb2222" || domain != "domkna" {
		t.Errorf("qualified name: %q %q %v", id, domain, err)
	}

	// An ID resolves too.
	if id, _, err := resolveProjectRef(cmd, settings, client, "aaaa1111", ""); err != nil || id != "aaaa1111" {
		t.Errorf("by id: %q %v", id, err)
	}

	// An unknown name lists what is visible and points at the escape hatch.
	_, _, err = resolveProjectRef(cmd, settings, client, "nope", "")
	if err == nil || !strings.Contains(err.Error(), "--domain-id") || !strings.Contains(err.Error(), "sto2/sandbox") {
		t.Errorf("unknown name: %v", err)
	}

	// An explicit domain skips resolution entirely, so a project you cannot
	// see is still addressable.
	id, domain, err = resolveProjectRef(cmd, settings, client, "FFFF-9999", "DOM-X")
	if err != nil || id != "ffff9999" || domain != "domx" {
		t.Errorf("explicit domain: %q %q %v", id, domain, err)
	}
}

func TestUserPrivilegeCommandsWired(t *testing.T) {
	root := NewRootCommand("test")
	for _, path := range [][]string{
		{"user", "privilege"},
		{"user", "privilege", "list"},
		{"user", "privilege", "set"},
		{"user", "privilege", "unset"},
	} {
		c, _, err := root.Find(path)
		if err != nil || c.Name() != path[len(path)-1] {
			t.Fatalf("command %v not wired: cmd=%v err=%v", path, c.Name(), err)
		}
	}
	set, _, _ := root.Find([]string{"user", "privilege", "set"})
	for _, f := range []string{"area", "project-id", "domain-id", "level"} {
		if set.Flags().Lookup(f) == nil {
			t.Errorf("privilege set is missing --%s", f)
		}
	}
	unset, _, _ := root.Find([]string{"user", "privilege", "unset"})
	for _, f := range []string{"area", "project-id"} {
		if unset.Flags().Lookup(f) == nil {
			t.Errorf("privilege unset is missing --%s", f)
		}
	}
	// The project flag must be local to the leaf, never the persistent,
	// env-backed context flag: $CLEURA_PROJECT_ID must not be able to supply
	// the project someone is being granted access on.
	if set.InheritedFlags().Lookup("project-id") != nil {
		t.Error("--project-id on privilege set must not be inherited from the project-context flags")
	}
}
