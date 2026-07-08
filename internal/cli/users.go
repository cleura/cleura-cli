package cli

import (
	"fmt"
	"io"
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
		Short: "View users in the Cleura account",
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(
		newUserListCommand(opts),
		newUserGetCommand(opts),
	)
	return cmd
}

func newUserListCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the users in the account",
		Long: `List the users in the account with their privileges. The ROLES column
summarizes each privilege area as area:type (types: full, read, or
project(n) for per-project grants), compressed to "full (all areas)" when
every area has full access. Use 'cleura user get' for the full breakdown.

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
				return apiAuthError("listing users", settings, resp.HTTPResponse, resp.Body)
			}
			users := *resp.JSON200

			header := []string{"ID", "USERNAME", "NAME", "EMAIL", "ADMIN", "2FA", "ROLES"}
			return output.Render(cmd.OutOrStdout(), opts.output, users, func(w io.Writer) error {
				if len(users) == 0 {
					opts.infof(cmd, "No users in the account")
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(users))
				for _, u := range users {
					rows = append(rows, []string{
						strconv.Itoa(u.Id),
						u.Name,
						displayName(u.Firstname, u.Lastname),
						strDeref(u.Email),
						yesNo(u.Admin),
						yesNo(hasTwoFactor(u.TwoFactorLogin)),
						rolesSummary(u.Privileges),
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
}

func newUserGetCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <user-id or username>",
		Short: "Show one user with the full privilege breakdown",
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
				kv := output.NewKVWriter(w)
				kv.Row("ID", user.Id)
				kv.Row("Username", user.Name)
				kv.Row("Name", displayName(user.Firstname, user.Lastname))
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
					kv.Row("IP restrictions", strconv.Itoa(len(user.IpRestrictions)))
				}
				return kv.Flush()
			})
		},
	}
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

	if _, numErr := strconv.Atoi(arg); numErr != nil {
		// Not an ID — try it as a username.
		list, err := client.IdentityListUsersWithResponse(cmd.Context())
		if err != nil {
			return nil, fmt.Errorf("looking up username %q: %w", arg, err)
		}
		if list.JSON200 != nil {
			for i, u := range *list.JSON200 {
				if u.Name == arg {
					return &(*list.JSON200)[i], nil
				}
			}
			return nil, fmt.Errorf("no user with ID or username %q", arg)
		}
	}
	return nil, apiAuthError("fetching user", settings, resp.HTTPResponse, resp.Body)
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

// hasTwoFactor reports whether any second factor is enrolled.
func hasTwoFactor(t *api.CommonUserLoginTwoFactorLogin) bool {
	if t == nil {
		return false
	}
	return t.Sms != nil || (t.Webauthn != nil && len(*t.Webauthn) > 0)
}

// twoFactorSummary names the enrolled methods, e.g. "sms" or "sms, webauthn (2 keys)".
func twoFactorSummary(t *api.CommonUserLoginTwoFactorLogin) string {
	if !hasTwoFactor(t) {
		return "none"
	}
	var methods []string
	if t.Sms != nil {
		methods = append(methods, "sms")
	}
	if t.Webauthn != nil && len(*t.Webauthn) > 0 {
		methods = append(methods, fmt.Sprintf("webauthn (%d keys)", len(*t.Webauthn)))
	}
	return strings.Join(methods, ", ")
}

type privilegeArea struct {
	name    string // compact token for the list column
	display string // Control Panel wording for user get
	p       *api.CommonUserLoginPrivilege
}

// privilegeAreas lists the set privilege areas in a stable order.
func privilegeAreas(p api.CommonUserLoginPrivileges) []privilegeArea {
	all := []privilegeArea{
		{"account", "Account", p.Account},
		{"ai-gateway", "AI Gateway", p.AiGateway},
		{"application", "Application", p.Application},
		{"invoice", "Invoice", p.Invoice},
		{"monitoring", "Monitoring", p.Monitoring},
		{"openstack", "OpenStack", p.Openstack},
		{"users", "Users", p.Users},
	}
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
	allFull := len(areas) == 7
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
