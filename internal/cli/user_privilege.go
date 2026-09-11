package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

// projectScopedArea is the privilege area that per-project grants apply to.
// The schema allows project_privileges on any area, but OpenStack projects are
// the only thing they can name, so --project-id implies this area.
const projectScopedArea = "openstack"

func newUserPrivilegeCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privilege",
		Short: "View and change a user's privileges",
		Long: `View and change what a Cleura account user may do.

A privilege is an area ('openstack', 'invoice', ...) plus a level. A level is
either 'full' or 'read' account-wide, or per-project: a set of OpenStack
projects, each with its own level. Set per-project access with --project-id and
account-wide access with --area.

The API has no privilege sub-resource, so every change here is a read, a merge
and a write of the whole user. Two changes made at the same moment can
therefore overwrite each other; the last write wins.

Account-admin rights are not part of this and cannot be changed through the
API.`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newUserPrivilegeListCommand(opts),
		newUserPrivilegeSetCommand(opts),
		newUserPrivilegeUnsetCommand(opts),
	)
	return cmd
}

// privilegeRow is one line of 'user privilege list': an area, its level, and
// what that level applies to.
type privilegeRow struct {
	Area  string `json:"area" yaml:"area"`
	Level string `json:"level" yaml:"level"`
	Scope string `json:"scope" yaml:"scope"`
}

func newUserPrivilegeListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "list <user-id or username>",
		Short:             "List a user's privileges",
		ValidArgsFunction: completeAccountUsers(opts),
		Long: `List one user's privileges, one row per area. Per-project grants are listed
in the SCOPE column as project:level, with project names resolved where the
project is visible to you.

Areas with no access are not listed.`,
		Example: `  cleura user privilege list johndoe
  cleura user privilege list 4763 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			user, err := lookupUser(cmd, settings, client, args[0])
			if err != nil {
				return err
			}

			rows := privilegeRows(user.Privileges, projectNamer(cmd, client, user.Privileges))
			header := []string{"AREA", "LEVEL", "SCOPE"}
			return output.Render(cmd.OutOrStdout(), opts.output, rows, func(w io.Writer) error {
				if len(rows) == 0 {
					opts.infof(cmd, "User %q has no privileges", user.Name)
					return output.Table(w, header, nil)
				}
				table := make([][]string, 0, len(rows))
				for _, r := range rows {
					table = append(table, []string{r.Area, r.Level, r.Scope})
				}
				return output.Table(w, header, table)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newUserPrivilegeSetCommand(opts *globalOptions) *cobra.Command {
	var area, projectRef, domainID, level string
	cmd := &cobra.Command{
		Use:               "set <user-id or username>",
		Short:             "Grant a user access to an area or a project",
		ValidArgsFunction: completeAccountUsers(opts),
		Long: `Grant a user access, either across an area or on one OpenStack project.

  --area <area> --level full|read      access to everything in that area
  --project-id <project> --level ...   access to one project only

Setting a project grant leaves that user's other project grants alone, so
repeat the command to add more. Setting an area replaces whatever that area
had, including its per-project grants, and the command says so when it does.

--project-id takes a project ID, or a name the CLI resolves against the
projects you can see. Project names are not unique across regions, so a name
that matches more than one project is refused; qualify it as region/name or
pass the ID. For a project you cannot see yourself, pass its ID together with
--domain-id.`,
		Example: `  cleura user privilege set johndoe --area openstack --level full
  cleura user privilege set johndoe --project-id team-alpha --level read
  cleura user privilege set johndoe --project-id sto2/team-alpha --level full
  cleura user privilege set johndoe --project-id 8a22c50f… --domain-id eb76… --level read`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, canonArea, err := privilegeTarget(cmd, area, projectRef)
			if err != nil {
				return err
			}
			area = canonArea
			wanted, err := parsePrivilegeLevel(level)
			if err != nil {
				return err
			}

			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			user, current, err := resolveUserForPrivilegeChange(cmd, settings, client, args[0])
			if err != nil {
				return err
			}

			var privileges api.CommonUserLoginPrivileges
			if target == targetArea {
				var replaced int
				privileges, replaced = setAreaPrivilege(current.Privileges, area, wanted)
				if replaced > 0 {
					opts.warnf(cmd, "replaced %d per-project grant(s) on %s with account-wide %s access", replaced, area, wanted)
				}
			} else {
				projectID, domain, err := resolveProjectRef(cmd, settings, client, projectRef, domainID)
				if err != nil {
					return err
				}
				var narrowedFrom api.UserUserLoginPrivilegeType
				privileges, narrowedFrom = setProjectPrivilege(current.Privileges, projectID, domain, wanted)
				if narrowedFrom != "" {
					opts.warnf(cmd, "%q had account-wide %s access to %s; it is now limited to the listed projects", user.Name, narrowedFrom, projectScopedArea)
				}
			}

			updated, err := patchPrivileges(cmd, settings, client, user, privileges)
			if err != nil {
				return err
			}
			if target == targetArea {
				opts.infof(cmd, "Granted %s access to %s for %q", wanted, area, user.Name)
			} else {
				opts.infof(cmd, "Granted %s access on project %s to %q", wanted, projectRef, user.Name)
			}
			return renderPrivileges(cmd, opts, client, updated)
		},
	}
	cmd.Flags().StringVar(&area, "area", "", "Privilege area to grant access to")
	cmd.Flags().StringVar(&projectRef, "project-id", "", "Project to grant access on, by ID or name (mutually exclusive with --area)")
	cmd.Flags().StringVar(&domainID, "domain-id", "", "Domain the project is in (needed only for a project that is not in your own project list)")
	cmd.Flags().StringVar(&level, "level", "", "Access level: full or read")
	_ = cmd.MarkFlagRequired("level")
	_ = cmd.RegisterFlagCompletionFunc("area", completePrivilegeAreas)
	_ = cmd.RegisterFlagCompletionFunc("level", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"full", "read"}, cobra.ShellCompDirectiveNoFileComp
	})
	addOutputFlag(cmd, opts)
	return cmd
}

func newUserPrivilegeUnsetCommand(opts *globalOptions) *cobra.Command {
	var area, projectRef string
	cmd := &cobra.Command{
		Use:               "unset <user-id or username>",
		Short:             "Remove a user's access to an area or a project",
		ValidArgsFunction: completeAccountUsers(opts),
		Long: `Remove access, either to a whole area or to one OpenStack project.

  --area <area>            remove all access to that area
  --project-id <project>   remove one project grant, leaving the others

Removing the last per-project grant removes the area itself, since an empty
project list grants nothing.`,
		Example: `  cleura user privilege unset johndoe --area invoice
  cleura user privilege unset johndoe --project-id team-alpha`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, canonArea, err := privilegeTarget(cmd, area, projectRef)
			if err != nil {
				return err
			}
			area = canonArea

			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			user, current, err := resolveUserForPrivilegeChange(cmd, settings, client, args[0])
			if err != nil {
				return err
			}

			var privileges api.CommonUserLoginPrivileges
			if target == targetArea {
				var existed bool
				privileges, existed = unsetAreaPrivilege(current.Privileges, area)
				if !existed {
					return fmt.Errorf("%q has no %s access to remove", user.Name, area)
				}
			} else {
				projectID, _, err := resolveProjectRef(cmd, settings, client, projectRef, "")
				if err != nil {
					return err
				}
				var removed, areaGone bool
				privileges, removed, areaGone = unsetProjectPrivilege(current.Privileges, projectID)
				if !removed {
					return fmt.Errorf("%q has no grant on project %s", user.Name, projectRef)
				}
				if areaGone {
					opts.infof(cmd, "That was the last per-project grant, so %s access is removed entirely", projectScopedArea)
				}
			}

			updated, err := patchPrivileges(cmd, settings, client, user, privileges)
			if err != nil {
				return err
			}
			if target == targetArea {
				opts.infof(cmd, "Removed %s access from %q", area, user.Name)
			} else {
				opts.infof(cmd, "Removed access on project %s from %q", projectRef, user.Name)
			}
			return renderPrivileges(cmd, opts, client, updated)
		},
	}
	cmd.Flags().StringVar(&area, "area", "", "Privilege area to remove access to")
	cmd.Flags().StringVar(&projectRef, "project-id", "", "Project to remove access on, by ID or name (mutually exclusive with --area)")
	_ = cmd.RegisterFlagCompletionFunc("area", completePrivilegeAreas)
	addOutputFlag(cmd, opts)
	return cmd
}

// privilegeTargetKind is which half of the privilege model a command addresses:
// a whole area, or one project inside the project-scoped area.
type privilegeTargetKind int

const (
	targetArea privilegeTargetKind = iota
	targetProject
)

// privilegeTarget validates that exactly one of --area/--project-id was given
// and returns the canonical area token for the --area case.
func privilegeTarget(cmd *cobra.Command, area, projectRef string) (privilegeTargetKind, string, error) {
	hasArea, hasProject := cmd.Flags().Changed("area"), cmd.Flags().Changed("project-id")
	switch {
	case hasArea && hasProject:
		return 0, "", fmt.Errorf("--area and --project-id are mutually exclusive: --area grants across every project, --project-id grants on one")
	case hasArea:
		canon := canonicalPrivilegeArea(area)
		if err := validPrivilegeArea(canon); err != nil {
			return 0, "", err
		}
		return targetArea, canon, nil
	case hasProject:
		if strings.TrimSpace(projectRef) == "" {
			return 0, "", fmt.Errorf("--project-id must not be empty")
		}
		return targetProject, "", nil
	default:
		return 0, "", fmt.Errorf("pass either --area (access across an area) or --project-id (access on one project)")
	}
}

// parsePrivilegeLevel accepts the two levels a client can set. The API's enum
// also carries "project", but that is a shape, not a level: it is implied by
// --project-id and cannot be requested directly.
func parsePrivilegeLevel(level string) (api.UserUserLoginPrivilegeType, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "full":
		return api.Full, nil
	case "read":
		return api.Read, nil
	case "project":
		return "", fmt.Errorf("--level project is not a level; grant per-project access with --project-id instead")
	default:
		return "", fmt.Errorf("--level %q is not valid, want full or read", level)
	}
}

// privilegeFields addresses each area of a privileges object by its flag token,
// so --area can set one area without a switch at every call site. The keys must
// match the tokens allPrivilegeAreas uses for the list column
// (TestPrivilegeAreaTokensMatch keeps the two in step).
func privilegeFields(p *api.CommonUserLoginPrivileges) map[string]**api.CommonUserLoginPrivilege {
	return map[string]**api.CommonUserLoginPrivilege{
		"account":     &p.Account,
		"ai-gateway":  &p.AiGateway,
		"application": &p.Application,
		"invoice":     &p.Invoice,
		"monitoring":  &p.Monitoring,
		"openstack":   &p.Openstack,
		"users":       &p.Users,
	}
}

// privilegeAreaTokens lists the addressable area tokens in display order.
func privilegeAreaTokens() []string {
	all := allPrivilegeAreas(api.CommonUserLoginPrivileges{})
	names := make([]string, 0, len(all))
	for _, a := range all {
		names = append(names, a.name)
	}
	return names
}

// canonicalPrivilegeArea normalizes an area token, accepting the underscore
// spelling the API uses in its JSON (ai_gateway) alongside the hyphenated token
// the CLI displays.
func canonicalPrivilegeArea(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", "-")
}

func validPrivilegeArea(area string) error {
	if slices.Contains(privilegeAreaTokens(), area) {
		return nil
	}
	return fmt.Errorf("unknown privilege area %q; valid areas: %s", area, strings.Join(privilegeAreaTokens(), ", "))
}

func completePrivilegeAreas(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return privilegeAreaTokens(), cobra.ShellCompDirectiveNoFileComp
}

// resolveUserForPrivilegeChange turns a name-or-ID into the user, and then
// re-reads it by ID. The re-read matters: lookupUser may have answered from the
// user list, and although the list and single-user responses share a schema,
// nothing guarantees the list populates every field. Merging onto a partial
// base would send the missing ones back cleared.
func resolveUserForPrivilegeChange(cmd *cobra.Command, settings config.Settings, client *cleura.Client, ref string) (found, current *api.CommonUserLogin, err error) {
	found, err = lookupUser(cmd, settings, client, ref)
	if err != nil {
		return nil, nil, err
	}
	current, err = fetchUserByID(cmd, settings, client, strconv.Itoa(found.Id))
	if err != nil {
		return nil, nil, err
	}
	return found, current, nil
}

// fetchUserByID reads one user by numeric ID. Note the singular path segment:
// the API exposes GET /identity/v2/user/{id} while create, edit and delete live
// under /identity/v2/users, and there is no GET on the plural path.
func fetchUserByID(cmd *cobra.Command, settings config.Settings, client *cleura.Client, id string) (*api.CommonUserLogin, error) {
	resp, err := client.IdentityGetUserWithResponse(cmd.Context(), id)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, userAuthError("fetching user", settings, resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// patchPrivileges writes the complete privileges object back. The API has no
// privilege sub-resource and does not document whether a PATCH merges or
// replaces a nested object, so the whole object is always sent, built from the
// one the user currently has.
func patchPrivileges(cmd *cobra.Command, settings config.Settings, client *cleura.Client, user *api.CommonUserLogin, privileges api.CommonUserLoginPrivileges) (*api.CommonUserLogin, error) {
	resp, err := client.IdentityEditUserWithResponse(cmd.Context(), strconv.Itoa(user.Id),
		api.IdentityEditUserJSONRequestBody{Privileges: &privileges})
	if err != nil {
		return nil, fmt.Errorf("updating privileges: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, userAuthError("updating privileges", settings, resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// renderPrivileges shows the resulting privileges, so a write reports its own
// outcome in the same shape 'privilege list' does.
func renderPrivileges(cmd *cobra.Command, opts *globalOptions, client *cleura.Client, user *api.CommonUserLogin) error {
	rows := privilegeRows(user.Privileges, projectNamer(cmd, client, user.Privileges))
	header := []string{"AREA", "LEVEL", "SCOPE"}
	return output.Render(cmd.OutOrStdout(), opts.output, rows, func(w io.Writer) error {
		if len(rows) == 0 {
			return output.Table(w, header, nil)
		}
		table := make([][]string, 0, len(rows))
		for _, r := range rows {
			table = append(table, []string{r.Area, r.Level, r.Scope})
		}
		return output.Table(w, header, table)
	})
}

// The four operations below are the whole merge, kept pure so the property
// that matters can be tested directly: changing one area or one project must
// leave everything else byte-for-byte intact. Each takes the user's current
// privileges and returns the complete object to send.
//
// A shallow copy is enough: every field is a pointer that gets replaced, never
// written through, so the caller's object is not mutated.

// setAreaPrivilege gives an area an account-wide level, replacing whatever it
// had. It reports how many per-project grants that discarded, since switching
// an area from project scope to account-wide silently drops them otherwise.
func setAreaPrivilege(base api.CommonUserLoginPrivileges, area string, level api.UserUserLoginPrivilegeType) (api.CommonUserLoginPrivileges, int) {
	out := base
	field := privilegeFields(&out)[area]
	prev := *field
	next := &api.CommonUserLoginPrivilege{Type: level}
	if prev != nil {
		// meta is undocumented (no field descriptions in the spec), so carry it
		// rather than silently drop something we do not understand.
		next.Meta = prev.Meta
	}
	*field = next
	return out, countProjectGrants(prev)
}

// setProjectPrivilege grants a level on one project, leaving that user's other
// project grants alone. If the area previously held account-wide access it
// reports the level that was narrowed away, which is a surprising side effect
// of a command that reads as "grant".
func setProjectPrivilege(base api.CommonUserLoginPrivileges, projectID, domainID string, level api.UserUserLoginPrivilegeType) (api.CommonUserLoginPrivileges, api.UserUserLoginPrivilegeType) {
	out := base
	field := privilegeFields(&out)[projectScopedArea]
	prev := *field
	narrowedFrom := api.UserUserLoginPrivilegeType("")
	if prev != nil && prev.Type != api.Project {
		narrowedFrom = prev.Type
	}
	grants := upsertProjectGrant(existingProjectGrants(prev), projectID, domainID, level)
	next := &api.CommonUserLoginPrivilege{Type: api.Project, ProjectPrivileges: &grants}
	if prev != nil {
		next.Meta = prev.Meta
	}
	*field = next
	return out, narrowedFrom
}

// unsetAreaPrivilege removes an area entirely, reporting whether there was
// anything to remove.
func unsetAreaPrivilege(base api.CommonUserLoginPrivileges, area string) (api.CommonUserLoginPrivileges, bool) {
	out := base
	field := privilegeFields(&out)[area]
	if *field == nil {
		return out, false
	}
	*field = nil
	return out, true
}

// unsetProjectPrivilege removes one project grant. Removing the last one drops
// the area as well, because an empty project list grants nothing and would
// leave a privilege object that says "project access to no projects".
func unsetProjectPrivilege(base api.CommonUserLoginPrivileges, projectID string) (privileges api.CommonUserLoginPrivileges, removed, areaRemoved bool) {
	out := base
	field := privilegeFields(&out)[projectScopedArea]
	prev := *field
	remaining, removed := removeProjectGrant(existingProjectGrants(prev), projectID)
	if !removed {
		return out, false, false
	}
	if len(remaining) == 0 {
		*field = nil
		return out, true, true
	}
	next := &api.CommonUserLoginPrivilege{Type: api.Project, ProjectPrivileges: &remaining}
	next.Meta = prev.Meta
	*field = next
	return out, true, false
}

// privilegeRows flattens a privileges object into one row per area that has
// access. name renders a project ID for display.
func privilegeRows(p api.CommonUserLoginPrivileges, name func(string) string) []privilegeRow {
	rows := make([]privilegeRow, 0)
	for _, area := range privilegeAreas(p) {
		row := privilegeRow{Area: area.name, Level: string(area.p.Type), Scope: "account-wide"}
		if area.p.Type == api.Project {
			grants := existingProjectGrants(area.p)
			parts := make([]string, 0, len(grants))
			for _, g := range grants {
				parts = append(parts, name(g.ProjectId)+":"+string(g.Type))
			}
			if len(parts) == 0 {
				row.Scope = "no projects"
			} else {
				row.Scope = strings.Join(parts, ", ")
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// projectNamer maps project IDs to names for display, using the caller's own
// project list. IDs that cannot be resolved (a project you cannot see, or a
// failed lookup) are shown as-is, so the display degrades instead of erroring.
func projectNamer(cmd *cobra.Command, client *cleura.Client, p api.CommonUserLoginPrivileges) func(string) string {
	identity := func(id string) string { return id }
	needed := false
	for _, area := range privilegeAreas(p) {
		if area.p.Type == api.Project && len(existingProjectGrants(area.p)) > 0 {
			needed = true
		}
	}
	if !needed {
		return identity
	}
	resp, err := client.OpenStackIdentityListRegionsWithProjectsWithResponse(cmd.Context())
	if err != nil || resp.JSON200 == nil {
		return identity
	}
	names := map[string]string{}
	for _, rp := range *resp.JSON200 {
		for _, proj := range rp.Projects {
			names[stripUUID(proj.Id)] = proj.Name
		}
	}
	return func(id string) string {
		if n, ok := names[stripUUID(id)]; ok {
			return n
		}
		return id
	}
}

func countProjectGrants(p *api.CommonUserLoginPrivilege) int {
	return len(existingProjectGrants(p))
}

func existingProjectGrants(p *api.CommonUserLoginPrivilege) []api.CommonUserLoginProjectPrivilege {
	if p == nil || p.ProjectPrivileges == nil {
		return nil
	}
	return *p.ProjectPrivileges
}

// upsertProjectGrant replaces the grant on projectID if there is one, and
// appends it otherwise, leaving every other grant untouched.
func upsertProjectGrant(grants []api.CommonUserLoginProjectPrivilege, projectID, domainID string, level api.UserUserLoginPrivilegeType) []api.CommonUserLoginProjectPrivilege {
	out := make([]api.CommonUserLoginProjectPrivilege, 0, len(grants)+1)
	replaced := false
	for _, g := range grants {
		if stripUUID(g.ProjectId) == stripUUID(projectID) {
			out = append(out, api.CommonUserLoginProjectPrivilege{ProjectId: projectID, DomainId: domainID, Type: level})
			replaced = true
			continue
		}
		out = append(out, g)
	}
	if !replaced {
		out = append(out, api.CommonUserLoginProjectPrivilege{ProjectId: projectID, DomainId: domainID, Type: level})
	}
	return out
}

// removeProjectGrant drops the grant on projectID, reporting whether there was
// one to drop.
func removeProjectGrant(grants []api.CommonUserLoginProjectPrivilege, projectID string) ([]api.CommonUserLoginProjectPrivilege, bool) {
	out := make([]api.CommonUserLoginProjectPrivilege, 0, len(grants))
	removed := false
	for _, g := range grants {
		if stripUUID(g.ProjectId) == stripUUID(projectID) {
			removed = true
			continue
		}
		out = append(out, g)
	}
	return out, removed
}

// stripUUID normalizes an ID for comparison and for the wire. The privilege
// schema wants the dashless 32-character form, while project listings may hand
// out the dashed one.
func stripUUID(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), "-", ""))
}

// projectCandidate is one project a reference could mean.
type projectCandidate struct {
	id, domainID, region, name string
}

// resolveProjectRef turns a project reference into the (project, domain) pair a
// grant needs. The reference is an ID, a name, or a region-qualified
// "region/name". Names are not unique across regions, and granting access to
// the wrong project is not a recoverable mistake, so an ambiguous name is
// refused rather than guessed.
//
// Resolution reads the caller's own project list, which is the only project
// listing the API offers; a project outside it can still be addressed by ID
// together with an explicit domain.
func resolveProjectRef(cmd *cobra.Command, settings config.Settings, client *cleura.Client, ref, domainOverride string) (projectID, domainID string, err error) {
	ref = strings.TrimSpace(ref)
	if domainOverride != "" {
		// The caller named the domain, so take the pair at face value: this is
		// the escape hatch for a project the caller cannot see.
		return stripUUID(ref), stripUUID(domainOverride), nil
	}

	wantRegion := ""
	if region, name, ok := strings.Cut(ref, "/"); ok {
		wantRegion, ref = region, name
	}

	resp, err := client.OpenStackIdentityListRegionsWithProjectsWithResponse(cmd.Context())
	if err != nil {
		return "", "", fmt.Errorf("looking up project %q: %w", ref, err)
	}
	if resp.JSON200 == nil {
		return "", "", apiAuthError("listing projects", settings, resp.HTTPResponse, resp.Body)
	}

	var matches []projectCandidate
	var available []string
	for _, rp := range *resp.JSON200 {
		for _, p := range rp.Projects {
			available = append(available, rp.Region.Tag+"/"+p.Name)
			if wantRegion != "" && !strings.EqualFold(rp.Region.Tag, wantRegion) {
				continue
			}
			if p.Name == ref || stripUUID(p.Id) == stripUUID(ref) {
				matches = append(matches, projectCandidate{stripUUID(p.Id), stripUUID(p.DomainId), rp.Region.Tag, p.Name})
			}
		}
	}
	// The same project listed twice is one match, not an ambiguity.
	matches = slices.CompactFunc(matches, func(a, b projectCandidate) bool { return a.id == b.id })

	switch len(matches) {
	case 1:
		return matches[0].id, matches[0].domainID, nil
	case 0:
		slices.Sort(available)
		if len(available) == 0 {
			return "", "", fmt.Errorf("no project %q: you have access to no projects, so pass the project ID with --domain-id", ref)
		}
		return "", "", fmt.Errorf("no project %q in the projects you can see (%s); for a project you cannot see, pass its ID with --domain-id",
			ref, strings.Join(available, ", "))
	default:
		lines := make([]string, 0, len(matches))
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("\n  %s/%s\t%s", m.region, m.name, m.id))
		}
		slices.Sort(lines)
		return "", "", fmt.Errorf("%q matches %d projects; qualify it as region/name or pass the ID:%s",
			ref, len(matches), strings.Join(lines, ""))
	}
}
