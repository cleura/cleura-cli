package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func newUserCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users in the Cleura account",
		Long: `View and manage the users in your Cleura account and their privileges. This
needs the 'users' privilege or account-admin rights; to see your own account use
'cleura whoami'.

Privileges are their own noun: 'cleura user privilege' grants and revokes
access, per area or per OpenStack project. Account-admin rights (the ADMIN
column) cannot be granted or revoked through the API — the create and edit
request bodies have no such field. Use the Control Panel for that.`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newUserListCommand(opts),
		newUserGetCommand(opts),
		newUserCreateCommand(opts),
		newUserEditCommand(opts),
		newUserDeleteCommand(opts),
		newUserPrivilegeCommand(opts),
	)
	return cmd
}

func newUserListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the users in the account",
		Long: `List the users in the account with their privileges. The PRIVILEGES column
summarizes each privilege area as area:type (types: full, read, or
project(n) for per-project grants), compressed to "full (all areas)" when
every area has full access. The 2FA column counts only active enrollments.
Use 'cleura user get' for the full breakdown.

Viewing other users requires the users privilege or account administrator
rights on the logged-in account.`,
		Example: "  cleura user list\n  cleura user list -o json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}

			resp, err := client.IdentityListUsersWithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing users: %w", err)
			}
			if resp.JSON200 == nil {
				return userAuthError("listing users", settings, resp.HTTPResponse, resp.Body)
			}
			// Build the view model before rendering so the CLI-computed columns
			// (2FA active, privilege summary) also land in -o json/yaml, matching
			// the gardener list commands.
			type userListView struct {
				api.CommonUserLogin
				TwoFactorActive   bool   `json:"two_factor_active" yaml:"two_factor_active"`
				PrivilegesSummary string `json:"privileges_summary" yaml:"privileges_summary"`
			}
			views := make([]userListView, 0, len(*resp.JSON200))
			for _, u := range *resp.JSON200 {
				views = append(views, userListView{
					CommonUserLogin:   u,
					TwoFactorActive:   hasTwoFactor(u.TwoFactorLogin),
					PrivilegesSummary: rolesSummary(u.Privileges),
				})
			}

			header := []string{"ID", "USERNAME", "NAME", "EMAIL", "ADMIN", "2FA", "PRIVILEGES"}
			return output.Render(cmd.OutOrStdout(), opts.output, views, func(w io.Writer) error {
				if len(views) == 0 {
					opts.infof(cmd, "No users in the account")
					return output.Table(w, header, nil)
				}
				// Stable, scannable table; -o json/yaml keep the API's order.
				sorted := append([]userListView(nil), views...)
				slices.SortFunc(sorted, func(a, b userListView) int {
					return strings.Compare(a.Name, b.Name)
				})
				rows := make([][]string, 0, len(sorted))
				for _, v := range sorted {
					rows = append(rows, []string{
						strconv.Itoa(v.Id),
						v.Name,
						displayName(v.Firstname, v.Lastname),
						strDeref(v.Email),
						yesNo(v.Admin),
						yesNo(v.TwoFactorActive),
						v.PrivilegesSummary,
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newUserGetCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "get <user-id or username>",
		ValidArgsFunction: noFileComp,
		Short:             "Show one user with the full privilege breakdown",
		Long: `Show one user (by numeric ID or exact username) with the full privilege
breakdown. Viewing another user requires the 'users' privilege or account
administrator rights; to see your own account without that privilege, use
'cleura whoami'.`,
		Example: `  cleura user get 4763
  cleura user get johndoe`,
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

			return output.Render(cmd.OutOrStdout(), opts.output, user, func(w io.Writer) error {
				return renderUserDetail(w, user)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

// renderUserDetail writes the single-user detail block. Shared by 'user get'
// and by create/edit, which render the user the API returns so a write shows
// its own result in the same shape a read does.
func renderUserDetail(w io.Writer, user *api.CommonUserLogin) error {
	kv := output.NewKVWriter(w)
	kv.Row("ID", user.Id)
	kv.Row("Username", user.Name)
	if name := displayName(user.Firstname, user.Lastname); name != "" {
		kv.Row("Name", name)
	}
	kv.Row("Email", strDeref(user.Email))
	if pending := strDeref(user.PendingEmail); pending != "" {
		kv.Row("Pending email", pending)
	}
	kv.Row("Admin", yesNo(user.Admin))
	kv.Row("2FA", twoFactorSummary(user.TwoFactorLogin))
	areas := privilegeAreas(user.Privileges)
	if len(areas) > 0 {
		kv.Row("Privileges", "")
		for _, area := range areas {
			kv.Row("  "+area.display, privilegeLabel(area.p))
		}
	}
	for _, area := range areas {
		if area.p.ProjectPrivileges == nil || len(*area.p.ProjectPrivileges) == 0 {
			continue
		}
		grants := make([]string, 0, len(*area.p.ProjectPrivileges))
		for _, pp := range *area.p.ProjectPrivileges {
			grants = append(grants, pp.ProjectId+":"+string(pp.Type))
		}
		kv.Row("Projects ("+area.display+")", strings.Join(grants, ", "))
	}
	kv.Row("Currency", user.Currency.Code)
	if lang := strDeref(user.Language); lang != "" {
		kv.Row("Language", lang)
	}
	if len(user.IpRestrictions) > 0 {
		cidrs := make([]string, 0, len(user.IpRestrictions))
		for _, r := range user.IpRestrictions {
			cidrs = append(cidrs, r.Cidr)
		}
		kv.Row("IP restrictions", strings.Join(cidrs, ", "))
	}
	return kv.Flush()
}

// userAuthError is apiAuthError plus a hint specific to the user commands: a
// 403 that is not about the token means the account lacks the users
// privilege, so point the caller at whoami for their own account (the escape
// hatch a non-admin needs). It is kept here, not in the shared apiAuthError,
// so whoami's own 403 and gardener 403s are not given this hint.
func userAuthError(op string, s config.Settings, resp *http.Response, body []byte) error {
	err := apiAuthError(op, s, resp, body)
	if resp.StatusCode != http.StatusForbidden {
		return err
	}
	// Judge by the API's own message (the field apiAuthError also keys on),
	// so the two agree: a token failure — or a 403 with no message to judge
	// by — gets the re-login hint from apiAuthError, not this one. A real
	// authorization message ("No access: ...") gets the privilege hint.
	var e api.FrameworkHttpErrorResponse
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" && !strings.Contains(strings.ToLower(e.Error.Message), "token") {
		return fmt.Errorf("%w\nthis needs the 'users' privilege; to view your own account use 'cleura whoami'", err)
	}
	return err
}

// lookupUser fetches a user by ID, falling back to an exact username match
// from the user list for non-numeric arguments.
func lookupUser(cmd *cobra.Command, settings config.Settings, client *cleura.Client, arg string) (*api.CommonUserLogin, error) {
	resp, err := client.IdentityGetUserWithResponse(cmd.Context(), arg)
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}

	// An auth failure would fail the fallback identically — don't spend a
	// second doomed request on it.
	authFailed := resp.StatusCode() == 401 || resp.StatusCode() == 403
	if _, numErr := strconv.Atoi(arg); numErr != nil && !authFailed {
		// Not an ID — try it as a username.
		list, err := client.IdentityListUsersWithResponse(cmd.Context())
		if err != nil {
			return nil, fmt.Errorf("looking up username %q: %w", arg, err)
		}
		if list.JSON200 == nil {
			return nil, userAuthError("looking up username", settings, list.HTTPResponse, list.Body)
		}
		for i, u := range *list.JSON200 {
			if u.Name == arg {
				return &(*list.JSON200)[i], nil
			}
		}
		return nil, fmt.Errorf("no user with ID or username %q", arg)
	}
	return nil, userAuthError("fetching user", settings, resp.HTTPResponse, resp.Body)
}

// displayName joins the optional first and last name.
func displayName(first, last *string) string {
	var parts []string
	if first != nil && *first != "" {
		parts = append(parts, *first)
	}
	if last != nil && *last != "" {
		parts = append(parts, *last)
	}
	return strings.Join(parts, " ")
}

func strDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// hasTwoFactor reports whether any ACTIVE second factor protects the account.
// Enrollments awaiting verification provide no protection and do not count.
func hasTwoFactor(t *api.CommonUserLoginTwoFactorLogin) bool {
	if t == nil {
		return false
	}
	if t.Sms != nil && t.Sms.Status == api.Active {
		return true
	}
	if t.Webauthn != nil {
		for _, key := range *t.Webauthn {
			if key.Status == api.Active {
				return true
			}
		}
	}
	return false
}

// twoFactorSummary names the enrolled methods and their state, e.g. "sms",
// "sms (awaiting verification)", or "sms, webauthn (2 keys)".
func twoFactorSummary(t *api.CommonUserLoginTwoFactorLogin) string {
	if t == nil {
		return "none"
	}
	var methods []string
	if t.Sms != nil {
		if t.Sms.Status == api.Active {
			methods = append(methods, "sms")
		} else {
			methods = append(methods, "sms (awaiting verification)")
		}
	}
	if t.Webauthn != nil && len(*t.Webauthn) > 0 {
		active := 0
		for _, key := range *t.Webauthn {
			if key.Status == api.Active {
				active++
			}
		}
		switch {
		case active == 1:
			methods = append(methods, "webauthn (1 key)")
		case active > 1:
			methods = append(methods, fmt.Sprintf("webauthn (%d keys)", active))
		default:
			methods = append(methods, "webauthn (awaiting verification)")
		}
	}
	if len(methods) == 0 {
		return "none"
	}
	return strings.Join(methods, ", ")
}

type privilegeArea struct {
	name    string // compact token for the list column
	display string // Control Panel wording for user get
	p       *api.CommonUserLoginPrivilege
}

// privilegeAreas lists the set privilege areas in a stable order.
// allPrivilegeAreas is the full set of privilege areas in display order; its
// length is the "all areas" total that rolesSummary compresses against.
func allPrivilegeAreas(p api.CommonUserLoginPrivileges) []privilegeArea {
	return []privilegeArea{
		{"account", "Account", p.Account},
		{"ai-gateway", "AI Gateway", p.AiGateway},
		{"application", "Application", p.Application},
		{"invoice", "Invoice", p.Invoice},
		{"monitoring", "Monitoring", p.Monitoring},
		{"openstack", "OpenStack", p.Openstack},
		{"users", "Users", p.Users},
	}
}

func privilegeAreas(p api.CommonUserLoginPrivileges) []privilegeArea {
	all := allPrivilegeAreas(p)
	set := all[:0]
	for _, area := range all {
		if area.p != nil {
			set = append(set, area)
		}
	}
	return set
}

// rolesSummary renders a compact privilege overview for the list column:
// area:type pairs ("openstack:project(3)" counts per-project grants),
// compressed to "full (all areas)" when every area is full access, or "-"
// when no privileges are set. The admin flag has its own column.
func rolesSummary(p api.CommonUserLoginPrivileges) string {
	areas := privilegeAreas(p)
	if len(areas) == 0 {
		return "-"
	}
	allFull := len(areas) == len(allPrivilegeAreas(p))
	for _, area := range areas {
		if area.p.Type != api.Full {
			allFull = false
			break
		}
	}
	if allFull {
		return "full (all areas)"
	}
	parts := make([]string, 0, len(areas))
	for _, area := range areas {
		entry := area.name + ":" + string(area.p.Type)
		if area.p.Type == api.Project && area.p.ProjectPrivileges != nil {
			entry = fmt.Sprintf("%s:project(%d)", area.name, len(*area.p.ProjectPrivileges))
		}
		parts = append(parts, entry)
	}
	return strings.Join(parts, " ")
}

// privilegeLabel renders one area's access level in the Control Panel's
// vocabulary.
func privilegeLabel(p *api.CommonUserLoginPrivilege) string {
	switch p.Type {
	case api.Full:
		return "Full Access"
	case api.Read:
		return "Read Access"
	case api.Project:
		n := 0
		if p.ProjectPrivileges != nil {
			n = len(*p.ProjectPrivileges)
		}
		return fmt.Sprintf("Project Access (%d projects)", n)
	default:
		return string(p.Type)
	}
}

func newUserCreateCommand(opts *globalOptions) *cobra.Command {
	var email, firstName, lastName string
	var ipRestrictions []string
	cmd := &cobra.Command{
		Use:   "create <username>",
		Short: "Create a user in the account",
		Long: `Create a Cleura account user. The password is read from a no-echo prompt, or
from stdin when piped — it is never passed on the command line.

The new user gets no privileges. Grant them with 'cleura user privilege set'.

Account-admin rights cannot be set through the API.`,
		Example: `  cleura user create johndoe --email john.doe@example.org
  cleura user create johndoe --email john.doe@example.org --first-name John --last-name Doe
  printf '%s' "$PASSWORD" | cleura user create ci-bot --email ci@example.org   # non-interactive (CI)`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: noFileComp,
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			if strings.TrimSpace(username) == "" {
				return fmt.Errorf("the username must not be empty")
			}
			// MarkFlagRequired only checks presence, so an unset $EMAIL that
			// expands to "" still has to be rejected here.
			if err := requireNonEmpty("email", email); err != nil {
				return err
			}
			if err := validateCIDRs(ipRestrictions); err != nil {
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

			// Password never comes from a flag: no-echo prompt on a TTY, piped
			// stdin otherwise. Mirrors 'cleura login' and 'openstack user create'.
			password, err := newPrompter(cmd).secret(cmd.Context(), "Password for the new user")
			if err != nil {
				return err
			}
			if password == "" {
				return fmt.Errorf("password must not be empty")
			}

			body := api.IdentityCreateUserJSONRequestBody{
				Username: username,
				Email:    email,
				Password: password,
			}
			if firstName != "" {
				body.Firstname = &firstName
			}
			if lastName != "" {
				body.Lastname = &lastName
			}
			if len(ipRestrictions) > 0 {
				body.IpRestrictions = &ipRestrictions
			}

			resp, err := client.IdentityCreateUserWithResponse(cmd.Context(), body)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}
			// Note: create answers 200, not 201.
			if resp.JSON200 == nil {
				return userAuthError("creating user", settings, resp.HTTPResponse, resp.Body)
			}

			user := resp.JSON200
			opts.infof(cmd, "Created user %q (ID %d)", user.Name, user.Id)
			return output.Render(cmd.OutOrStdout(), opts.output, user, func(w io.Writer) error {
				return renderUserDetail(w, user)
			})
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Email address (required)")
	cmd.Flags().StringVar(&firstName, "first-name", "", "First name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "Last name")
	cmd.Flags().StringArrayVar(&ipRestrictions, "ip-restriction", nil, "Restrict logins to a CIDR (repeatable)")
	_ = cmd.MarkFlagRequired("email")
	addOutputFlag(cmd, opts)
	return cmd
}

func newUserEditCommand(opts *globalOptions) *cobra.Command {
	var username, email, firstName, lastName string
	var ipRestrictions []string
	var clearIPRestrictions, setPassword, yes bool
	cmd := &cobra.Command{
		Use:               "edit <user-id or username>",
		Short:             "Change a user's details, privileges or password",
		ValidArgsFunction: completeAccountUsers(opts),
		Long: `Change a Cleura account user's details or password. Only the flags you pass
are sent, so an unset field is never overwritten.

--ip-restriction replaces the whole set; --clear-ip-restrictions removes it.
--set-password prompts for a new password (no-echo, or piped stdin).

Privileges are managed separately, with 'cleura user privilege'. Account-admin
rights cannot be changed through the API.`,
		Example: `  cleura user edit johndoe --email new.address@example.org
  cleura user edit johndoe --first-name Jonathan
  cleura user edit 4763 --set-password
  cleura user edit ci-bot --ip-restriction 203.0.113.0/24`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			changedFlags := []string{"username", "email", "first-name", "last-name", "ip-restriction", "clear-ip-restrictions", "set-password"}
			anyChanged := false
			for _, f := range changedFlags {
				if cmd.Flags().Changed(f) {
					anyChanged = true
					break
				}
			}
			if !anyChanged {
				return fmt.Errorf("nothing to change: pass at least one of --%s", strings.Join(changedFlags, ", --"))
			}
			if cmd.Flags().Changed("ip-restriction") && clearIPRestrictions {
				return fmt.Errorf("--ip-restriction and --clear-ip-restrictions are mutually exclusive")
			}
			if err := validateCIDRs(ipRestrictions); err != nil {
				return err
			}
			// The edit body types firstname/lastname as plain strings whose
			// pattern needs at least one character — unlike create, where both
			// are nullable. So a name can be set and never unset; fail here
			// instead of sending a value the API will reject (wishlist #33).
			for _, f := range []struct{ flag, val string }{{"first-name", firstName}, {"last-name", lastName}} {
				if cmd.Flags().Changed(f.flag) && strings.TrimSpace(f.val) == "" {
					return fmt.Errorf("--%s cannot be cleared: the API has no way to unset a name once it is set", f.flag)
				}
			}

			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}

			// Resolve name-or-ID first, so a typo fails before anything is sent.
			found, err := lookupUser(cmd, settings, client, args[0])
			if err != nil {
				return err
			}
			userID := strconv.Itoa(found.Id)
			current := found

			body := api.IdentityEditUserJSONRequestBody{}
			if cmd.Flags().Changed("username") {
				if err := requireNonEmpty("username", username); err != nil {
					return err
				}
				body.Username = &username
			}
			if cmd.Flags().Changed("email") {
				if err := requireNonEmpty("email", email); err != nil {
					return err
				}
				body.Email = &email
			}
			if cmd.Flags().Changed("first-name") {
				body.Firstname = &firstName
			}
			if cmd.Flags().Changed("last-name") {
				body.Lastname = &lastName
			}
			if cmd.Flags().Changed("ip-restriction") {
				body.IpRestrictions = &ipRestrictions
			}
			if clearIPRestrictions {
				empty := []string{}
				body.IpRestrictions = &empty
			}

			// Renaming the account you are logged in as invalidates the username
			// half of the stored credential pair, so say so before doing it.
			if body.Username != nil && *body.Username != current.Name && current.Name == settings.Username && !yes {
				ok, err := newPrompter(cmd).confirm(cmd.Context(), fmt.Sprintf(
					"Rename %q — the account this profile is logged in as — to %q? The profile still stores the old username", current.Name, *body.Username))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted; rerun with --yes to confirm")
				}
			}

			if setPassword {
				password, err := newPrompter(cmd).secret(cmd.Context(), "New password")
				if err != nil {
					return err
				}
				if password == "" {
					return fmt.Errorf("password must not be empty")
				}
				body.Password = &password
			}

			resp, err := client.IdentityEditUserWithResponse(cmd.Context(), userID, body)
			if err != nil {
				return fmt.Errorf("editing user: %w", err)
			}
			if resp.JSON200 == nil {
				return userAuthError("editing user", settings, resp.HTTPResponse, resp.Body)
			}

			user := resp.JSON200
			opts.infof(cmd, "Updated user %q (ID %d)", user.Name, user.Id)
			if user.Name != current.Name && current.Name == settings.Username {
				opts.warnf(cmd, "profile %q still stores the username %q; run 'cleura config profile set username %s' (or 'cleura login') before the next command", settings.ProfileName, current.Name, user.Name)
			}
			if p := strDeref(user.PendingEmail); p != "" {
				opts.infof(cmd, "Email change to %s is pending verification", p)
			}
			return output.Render(cmd.OutOrStdout(), opts.output, user, func(w io.Writer) error {
				return renderUserDetail(w, user)
			})
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "New username")
	cmd.Flags().StringVar(&email, "email", "", "New email address (may require verification)")
	cmd.Flags().StringVar(&firstName, "first-name", "", "New first name")
	cmd.Flags().StringVar(&lastName, "last-name", "", "New last name")
	cmd.Flags().StringArrayVar(&ipRestrictions, "ip-restriction", nil, "Replace the login CIDR restrictions (repeatable)")
	cmd.Flags().BoolVar(&clearIPRestrictions, "clear-ip-restrictions", false, "Remove all login CIDR restrictions")
	cmd.Flags().BoolVar(&setPassword, "set-password", false, "Prompt for a new password (read no-echo, or from piped stdin)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation shown when renaming the account this profile is logged in as")
	addOutputFlag(cmd, opts)
	return cmd
}

func newUserDeleteCommand(opts *globalOptions) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:               "delete <user-id or username>",
		Short:             "Delete a user from the account",
		ValidArgsFunction: completeAccountUsers(opts),
		Long: `Delete a Cleura account user, given by numeric ID or exact username. This is
irreversible. The command asks for confirmation and refuses on a
non-interactive terminal unless --yes is given.

The account you are logged in as cannot be deleted here.`,
		Example: `  cleura user delete johndoe
  cleura user delete 4763 --yes`,
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
			// Resolve first so the confirmation names the user and a typo'd
			// argument fails before we prompt to delete anything.
			user, err := lookupUser(cmd, settings, client, args[0])
			if err != nil {
				return err
			}
			if user.Name == settings.Username {
				return fmt.Errorf("refusing to delete %q: it is the account this profile is logged in as", user.Name)
			}

			if !yes {
				ok, err := newPrompter(cmd).confirm(cmd.Context(), fmt.Sprintf("Delete user %q (ID %d)? This cannot be undone", user.Name, user.Id))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted; rerun with --yes to confirm")
				}
			}

			resp, err := client.IdentityDeleteUserWithResponse(cmd.Context(), strconv.Itoa(user.Id))
			if err != nil {
				return fmt.Errorf("deleting user: %w", err)
			}
			// Success is a body-less 204, so check the status code.
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return userAuthError("deleting user", settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Deleted user %q (ID %d)", user.Name, user.Id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt (required on a non-interactive terminal)")
	return cmd
}

// validateCIDRs rejects a malformed --ip-restriction before the request, so a
// typo names itself instead of arriving as an opaque API 400.
func validateCIDRs(cidrs []string) error {
	for _, c := range cidrs {
		if _, err := netip.ParsePrefix(strings.TrimSpace(c)); err != nil {
			return fmt.Errorf("--ip-restriction %q is not a CIDR block (want e.g. 203.0.113.0/24): %w", c, err)
		}
	}
	return nil
}

// completeAccountUsers completes a user argument with the account's usernames.
func completeAccountUsers(opts *globalOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		_, settings, err := opts.settings()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client, err := opts.authenticatedClient(settings)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		resp, err := client.IdentityListUsersWithResponse(cmd.Context())
		if err != nil || resp.JSON200 == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(*resp.JSON200))
		for _, u := range *resp.JSON200 {
			if name := displayName(u.Firstname, u.Lastname); name != "" {
				out = append(out, u.Name+"\t"+name)
				continue
			}
			out = append(out, u.Name)
		}
		slices.Sort(out)
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
