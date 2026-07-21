package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func newOpenstackCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "openstack",
		Short: "Manage OpenStack identity resources",
		Long: `Manage OpenStack (Keystone) identity resources — domains, projects, users, and
role assignments.

These are distinct from Cleura account users ('cleura user'): OpenStack users
authenticate against OpenStack itself. Almost every command is scoped to a domain
and needs --domain-id <id>; an account usually has several domains (one per region),
so it is normally required — the CLI auto-selects only when there is exactly one.
'domain list' and 'project list' take no --domain-id; use them to find the IDs.`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newOpenstackDomainCommand(opts),
		newOpenstackProjectCommand(opts),
		newOpenstackRoleCommand(opts),
		newOpenstackUserCommand(opts),
	)
	return cmd
}

func newOpenstackDomainCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "View OpenStack domains",
		Long: `View the OpenStack (Keystone) domains available to your account. A domain ID
identifies where projects and users live; it is what 'cleura openstack project
create --domain-id' expects.`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(newOpenstackDomainListCommand(opts))
	return cmd
}

func newOpenstackDomainListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the OpenStack domains for the account",
		Long:    "List the OpenStack (Keystone) domains available to your account, with the domain ID used by 'cleura openstack project create --domain-id'.",
		Example: "  cleura openstack domain list\n  cleura openstack domain list -o json",
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

			resp, err := client.OpenStackIdentityListDomainsWithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing domains: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing domains", settings, resp.HTTPResponse, resp.Body)
			}

			domains := *resp.JSON200
			header := []string{"ID", "NAME", "AREA", "STATUS"}
			return output.Render(cmd.OutOrStdout(), opts.output, domains, func(w io.Writer) error {
				if len(domains) == 0 {
					opts.infof(cmd, "No OpenStack domains available")
					return output.Table(w, header, nil)
				}
				// Sort the table (by name, then area, then id — names are often
				// non-unique); -o json/yaml keep API order.
				sorted := append([]api.CommonOpenStackDomain(nil), domains...)
				slices.SortFunc(sorted, func(a, b api.CommonOpenStackDomain) int {
					if c := strings.Compare(strDeref(a.Name), strDeref(b.Name)); c != 0 {
						return c
					}
					if c := strings.Compare(a.Area.Name, b.Area.Name); c != 0 {
						return c
					}
					return strings.Compare(a.Id, b.Id)
				})
				rows := make([][]string, 0, len(sorted))
				for _, d := range sorted {
					rows = append(rows, []string{d.Id, strDeref(d.Name), d.Area.Name, d.Status})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackProjectCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage OpenStack projects",
		Long:  "Manage OpenStack projects, which live under a domain.",
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(
		newOpenstackProjectListCommand(opts),
		newOpenstackProjectCreateCommand(opts),
		newOpenstackProjectEditCommand(opts),
	)
	return cmd
}

func newOpenstackProjectListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your OpenStack projects",
		Long: `List the OpenStack projects you can access, across regions.

This lists the caller's own projects — there is no account-wide project listing,
so a project created for another user will not appear here.`,
		Example: "  cleura openstack project list\n  cleura openstack project list -o json",
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

			resp, err := client.OpenStackIdentityListRegionsWithProjectsWithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing projects: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing projects", settings, resp.HTTPResponse, resp.Body)
			}

			// The API groups projects by region; flatten to one row per project
			// with the region tag attached, so -o json is a flat, scriptable
			// array that matches the table (view-model-before-render).
			type projectListView struct {
				api.OpenStackIdentityProject
				Region string `json:"region" yaml:"region"`
			}
			views := make([]projectListView, 0)
			for _, rp := range *resp.JSON200 {
				for _, p := range rp.Projects {
					views = append(views, projectListView{OpenStackIdentityProject: p, Region: rp.Region.Tag})
				}
			}

			header := []string{"ID", "NAME", "REGION", "DOMAIN", "ENABLED"}
			return output.Render(cmd.OutOrStdout(), opts.output, views, func(w io.Writer) error {
				if len(views) == 0 {
					opts.infof(cmd, "No projects found")
					return output.Table(w, header, nil)
				}
				// Sort the table by region then name; -o json/yaml keep API order.
				sorted := append([]projectListView(nil), views...)
				slices.SortFunc(sorted, func(a, b projectListView) int {
					if c := strings.Compare(a.Region, b.Region); c != 0 {
						return c
					}
					return strings.Compare(a.Name, b.Name)
				})
				rows := make([][]string, 0, len(sorted))
				for _, v := range sorted {
					rows = append(rows, []string{v.Id, v.Name, v.Region, v.DomainId, yesNo(v.Enabled)})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackProjectCreateCommand(opts *globalOptions) *cobra.Command {
	var domain, description string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an OpenStack project in a domain",
		Long: `Create an OpenStack project. The project is created in the account's domain;
when the account has more than one domain, choose one with --domain-id (list them
with 'cleura openstack domain list').

The created project — including its ID — is printed; use -o json to capture the
ID for scripting (e.g. to grant access or launch resources in it).`,
		Example: `  cleura openstack project create my-project
  cleura openstack project create my-project --description "team sandbox"
  cleura openstack project create my-project --domain-id <domain-id> -o json`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: noFileComp,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}

			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}

			body := api.OpenStackIdentityCreateProjectJSONRequestBody{Name: name}
			if description != "" {
				body.Description = &description
			}
			resp, err := client.OpenStackIdentityCreateProjectWithResponse(cmd.Context(), domainID, body)
			if err != nil {
				return fmt.Errorf("creating project: %w", err)
			}
			if resp.JSON201 == nil {
				return apiAuthError("creating project", settings, resp.HTTPResponse, resp.Body)
			}

			project := *resp.JSON201
			opts.infof(cmd, "Created project %q (ID %s) in domain %s", project.Name, project.Id, project.DomainId)
			return output.Render(cmd.OutOrStdout(), opts.output, project, func(w io.Writer) error {
				return renderProjectKV(w, project)
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID to create the project in (required unless the account has a single domain)")
	cmd.Flags().StringVar(&description, "description", "", "Optional project description")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackProjectEditCommand(opts *globalOptions) *cobra.Command {
	var domain, name, description string
	var enable, disable bool
	cmd := &cobra.Command{
		Use:   "edit <project-id>",
		Short: "Update an OpenStack project",
		Long: `Update an OpenStack project's name, description, or enabled state. Only the
flags you pass are changed.

The API has no project delete; --disable is the closest equivalent — the project
stays but is turned off.`,
		Example: `  cleura openstack project edit <project-id> --name new-name
  cleura openstack project edit <project-id> --description "archived"
  cleura openstack project edit <project-id> --disable`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: noFileComp,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			// Build the PATCH body from only the flags actually set, so an
			// unspecified field is left unchanged (and --description "" can clear
			// it). Validate this usage error before any auth or network work.
			body := api.OpenStackIdentityEditProjectJSONRequestBody{}
			if cmd.Flags().Changed("name") {
				if err := requireNonEmpty("name", name); err != nil {
					return err
				}
				body.Name = &name
			}
			if cmd.Flags().Changed("description") {
				body.Description = &description
			}
			if enable {
				t := true
				body.Enabled = &t
			}
			if disable {
				f := false
				body.Enabled = &f
			}
			if body.Name == nil && body.Description == nil && body.Enabled == nil {
				return fmt.Errorf("nothing to update: pass --name, --description, --enable or --disable")
			}

			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}

			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}

			resp, err := client.OpenStackIdentityEditProjectWithResponse(cmd.Context(), domainID, projectID, body)
			if err != nil {
				return fmt.Errorf("updating project: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("updating project", settings, resp.HTTPResponse, resp.Body)
			}

			project := *resp.JSON200
			opts.infof(cmd, "Updated project %q (ID %s)", project.Name, project.Id)
			return output.Render(cmd.OutOrStdout(), opts.output, project, func(w io.Writer) error {
				return renderProjectKV(w, project)
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID the project is in (required unless the account has a single domain)")
	cmd.Flags().StringVar(&name, "name", "", "New project name")
	cmd.Flags().StringVar(&description, "description", "", "New project description (pass \"\" to clear)")
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable the project")
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable the project (the closest thing to deletion)")
	cmd.MarkFlagsMutuallyExclusive("enable", "disable")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackRoleCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "View OpenStack roles and manage role assignments",
		Long:  "View the OpenStack (Keystone) roles that can be granted to users on projects, and manage role assignments (see 'cleura openstack role assignment').",
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(
		newOpenstackRoleListCommand(opts),
		newOpenstackRoleAssignmentCommand(opts),
	)
	return cmd
}

func newOpenstackRoleListCommand(opts *globalOptions) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List the assignable OpenStack roles",
		Long:    "List the OpenStack (Keystone) roles available in a domain — the names accepted by 'cleura openstack role assignment create --role'.",
		Example: "  cleura openstack role list\n  cleura openstack role list -o json",
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
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			resp, err := client.OpenStackIdentityListRolesWithResponse(cmd.Context(), domainID)
			if err != nil {
				return fmt.Errorf("listing roles: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing roles", settings, resp.HTTPResponse, resp.Body)
			}

			roles := *resp.JSON200
			header := []string{"ID", "NAME"}
			return output.Render(cmd.OutOrStdout(), opts.output, roles, func(w io.Writer) error {
				if len(roles) == 0 {
					opts.infof(cmd, "No roles available")
					return output.Table(w, header, nil)
				}
				sorted := append([]api.OpenStackIdentityProjectRole(nil), roles...)
				slices.SortFunc(sorted, func(a, b api.OpenStackIdentityProjectRole) int {
					return strings.Compare(a.Name, b.Name)
				})
				rows := make([][]string, 0, len(sorted))
				for _, r := range sorted {
					rows = append(rows, []string{r.Id, r.Name})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID (required unless the account has a single domain)")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackRoleAssignmentCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "assignment",
		Short: "Manage role assignments (a user's roles on a project)",
		Long: `Manage OpenStack role assignments — the binding of a user, a project, and one
or more roles. 'create' grants access, 'delete' revokes it, and 'list' shows a
user's assignments across projects.`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newOpenstackRoleAssignmentCreateCommand(opts),
		newOpenstackRoleAssignmentListCommand(opts),
		newOpenstackRoleAssignmentDeleteCommand(opts),
	)
	return cmd
}

func newOpenstackRoleAssignmentCreateCommand(opts *globalOptions) *cobra.Command {
	var domain, user, project string
	var roles []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Grant a user roles on a project",
		Long: `Grant an OpenStack user one or more roles on a project (an additive role
assignment). The user is given by name or ID and the roles by name (resolved
against the domain's roles — see 'cleura openstack role list'); the project is
an ID.`,
		Example: `  cleura openstack role assignment create --user alice --role member --project-id <id>
  cleura openstack role assignment create --user alice --role member,load-balancer_member --project-id <id>`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// MarkFlagRequired only checks presence, so guard empty values too.
			if err := requireNonEmpty("user", user); err != nil {
				return err
			}
			if err := requireNonEmpty("project-id", project); err != nil {
				return err
			}
			if len(roles) == 0 {
				return fmt.Errorf("--role must name at least one role")
			}
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd, settings, client, domainID, user)
			if err != nil {
				return err
			}
			roleIDs, err := resolveRoleIDs(cmd, settings, client, domainID, roles)
			if err != nil {
				return err
			}

			body := api.OpenStackIdentityGrantProjectAccessJSONRequestBody{
				Projects: []api.OpenStackIdentityProjectAccessRequest{
					{ProjectId: project, Roles: roleIDs},
				},
			}
			resp, err := client.OpenStackIdentityGrantProjectAccessWithResponse(cmd.Context(), domainID, u.Id, body)
			if err != nil {
				return fmt.Errorf("creating role assignment: %w", err)
			}
			// Success is a body-less 2xx, so check the status code.
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return apiAuthError("creating role assignment", settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Granted %s on project %s to user %q", strings.Join(roles, ", "), project, u.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID (required unless the account has a single domain)")
	cmd.Flags().StringVar(&user, "user", "", "User name or ID to grant access to")
	cmd.Flags().StringVar(&project, "project-id", "", "Project ID to grant access on")
	cmd.Flags().StringSliceVar(&roles, "role", nil, "Role name(s) to grant, comma-separated (see 'cleura openstack role list')")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.MarkFlagRequired("project-id")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	return cmd
}

func newOpenstackRoleAssignmentListCommand(opts *globalOptions) *cobra.Command {
	var domain, user string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List a user's role assignments across projects",
		Long: `List the projects an OpenStack user can access and the roles they hold on each.
The user is given by name or ID with --user.`,
		Example: `  cleura openstack role assignment list --user alice
  cleura openstack role assignment list --user alice -o json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireNonEmpty("user", user); err != nil {
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
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd, settings, client, domainID, user)
			if err != nil {
				return err
			}
			resp, err := client.OpenStackIdentityListUserProjectsWithResponse(cmd.Context(), domainID, u.Id)
			if err != nil {
				return fmt.Errorf("listing role assignments: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing role assignments", settings, resp.HTTPResponse, resp.Body)
			}

			memberships := *resp.JSON200
			header := []string{"PROJECT ID", "PROJECT", "ROLES"}
			return output.Render(cmd.OutOrStdout(), opts.output, memberships, func(w io.Writer) error {
				if len(memberships) == 0 {
					opts.infof(cmd, "User %q has no role assignments in domain %s", u.Name, domainID)
					return output.Table(w, header, nil)
				}
				opts.infof(cmd, "Role assignments for user %q in domain %s:", u.Name, domainID)
				sorted := append([]api.OpenStackIdentityProjectMembership(nil), memberships...)
				slices.SortFunc(sorted, func(a, b api.OpenStackIdentityProjectMembership) int {
					return strings.Compare(a.Name, b.Name)
				})
				rows := make([][]string, 0, len(sorted))
				for _, m := range sorted {
					rows = append(rows, []string{m.Id, m.Name, roleNames(m.Roles)})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID (required unless the account has a single domain)")
	cmd.Flags().StringVar(&user, "user", "", "User name or ID whose assignments to list")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackRoleAssignmentDeleteCommand(opts *globalOptions) *cobra.Command {
	var domain, user, project, role string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Revoke a user's role on a project",
		Long: `Revoke one role from an OpenStack user on a project. The user is given by name
or ID and the role by name; the project is an ID. Reversible with 'create'.`,
		Example: "  cleura openstack role assignment delete --user alice --role member --project-id <id>",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireNonEmpty("user", user); err != nil {
				return err
			}
			if err := requireNonEmpty("project-id", project); err != nil {
				return err
			}
			if err := requireNonEmpty("role", role); err != nil {
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
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			u, err := resolveUser(cmd, settings, client, domainID, user)
			if err != nil {
				return err
			}
			roleIDs, err := resolveRoleIDs(cmd, settings, client, domainID, []string{role})
			if err != nil {
				return err
			}

			resp, err := client.OpenStackIdentityRevokeProjectAccessWithResponse(cmd.Context(), domainID, u.Id, project, roleIDs[0])
			if err != nil {
				return fmt.Errorf("deleting role assignment: %w", err)
			}
			// Success is a body-less 2xx, so check the status code.
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return apiAuthError("deleting role assignment", settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Revoked %s on project %s from user %q", role, project, u.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID (required unless the account has a single domain)")
	cmd.Flags().StringVar(&user, "user", "", "User name or ID to revoke access from")
	cmd.Flags().StringVar(&project, "project-id", "", "Project ID to revoke access on")
	cmd.Flags().StringVar(&role, "role", "", "Role name to revoke (see 'cleura openstack role list')")
	_ = cmd.MarkFlagRequired("user")
	_ = cmd.MarkFlagRequired("project-id")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	return cmd
}

func newOpenstackUserCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage OpenStack users",
		Long: `Manage OpenStack (Keystone) users, which live under a domain.

These are distinct from Cleura account users ('cleura user'): OpenStack users
authenticate against OpenStack itself and are granted access to projects via
roles ('cleura openstack role assignment create').`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newOpenstackUserListCommand(opts),
		newOpenstackUserCreateCommand(opts),
		newOpenstackUserDeleteCommand(opts),
	)
	return cmd
}

func newOpenstackUserListCommand(opts *globalOptions) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List OpenStack users in a domain",
		Long:    "List the OpenStack (Keystone) users in a domain.",
		Example: "  cleura openstack user list\n  cleura openstack user list -o json",
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
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			resp, err := client.OpenStackIdentityListUsersWithResponse(cmd.Context(), domainID)
			if err != nil {
				return fmt.Errorf("listing users: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing users", settings, resp.HTTPResponse, resp.Body)
			}

			users := *resp.JSON200
			// The list is scoped to one domain, so the domain is shown once as
			// context (like kubectl's namespace) rather than as a column repeating
			// the same value on every row. -o json/yaml keep domain_id per user.
			header := []string{"ID", "NAME", "ENABLED"}
			return output.Render(cmd.OutOrStdout(), opts.output, users, func(w io.Writer) error {
				if len(users) == 0 {
					opts.infof(cmd, "No OpenStack users in domain %s", domainID)
					return output.Table(w, header, nil)
				}
				opts.infof(cmd, "OpenStack users in domain %s:", domainID)
				sorted := append([]api.OpenStackIdentityUser(nil), users...)
				slices.SortFunc(sorted, func(a, b api.OpenStackIdentityUser) int {
					return strings.Compare(a.Name, b.Name)
				})
				rows := make([][]string, 0, len(sorted))
				for _, u := range sorted {
					rows = append(rows, []string{u.Id, u.Name, yesNo(u.Enabled)})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID (required unless the account has a single domain)")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackUserCreateCommand(opts *globalOptions) *cobra.Command {
	var domain, description string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create an OpenStack user in a domain",
		Long: `Create an OpenStack user. The password is read from a no-echo prompt, or from
stdin when piped — it is never passed on the command line. Grant the new user
access to projects afterwards with 'cleura openstack role assignment create'.`,
		Example: `  cleura openstack user create alice
  printf '%s' "$PASSWORD" | cleura openstack user create alice   # non-interactive (CI)
  cleura openstack user create svc --description "CI account" -o json`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: noFileComp,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}

			// Password never comes from a flag: no-echo prompt on a TTY, piped
			// stdin otherwise (secret handles both). Mirrors 'cleura login'.
			password, err := newPrompter(cmd).secret(cmd.Context(), "Password for the new user")
			if err != nil {
				return err
			}
			if password == "" {
				return fmt.Errorf("password must not be empty")
			}

			body := api.OpenStackIdentityCreateUserJSONRequestBody{Name: name, Password: password}
			if description != "" {
				body.Description = &description
			}
			resp, err := client.OpenStackIdentityCreateUserWithResponse(cmd.Context(), domainID, body)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}
			if resp.JSON201 == nil {
				return apiAuthError("creating user", settings, resp.HTTPResponse, resp.Body)
			}

			user := *resp.JSON201
			opts.infof(cmd, "Created OpenStack user %q (ID %s) in domain %s", user.Name, user.Id, user.DomainId)
			return output.Render(cmd.OutOrStdout(), opts.output, user, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("ID", user.Id)
				kv.Row("Name", user.Name)
				kv.Row("Domain", user.DomainId)
				kv.Row("Enabled", yesNo(user.Enabled))
				if d := strDeref(user.Description); d != "" {
					kv.Row("Description", d)
				}
				return kv.Flush()
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID to create the user in (required unless the account has a single domain)")
	cmd.Flags().StringVar(&description, "description", "", "Optional user description")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
}

func newOpenstackUserDeleteCommand(opts *globalOptions) *cobra.Command {
	var domain string
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <user>",
		Short: "Delete an OpenStack user",
		Long: `Delete an OpenStack user, given by name or ID. This is irreversible. The
command asks for confirmation and refuses on a non-interactive terminal unless
--yes is given.`,
		Example: `  cleura openstack user delete alice
  cleura openstack user delete <user-id> --yes`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: noFileComp,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}
			client, err := opts.authenticatedClient(settings)
			if err != nil {
				return err
			}
			domainID, err := resolveDomain(cmd, settings, client, domain)
			if err != nil {
				return err
			}
			// Resolve name-or-ID first so the confirmation can name the user, and
			// a typo'd user fails before we prompt to delete anything.
			user, err := resolveUser(cmd, settings, client, domainID, args[0])
			if err != nil {
				return err
			}

			if !yes {
				ok, err := newPrompter(cmd).confirm(cmd.Context(), fmt.Sprintf("Delete OpenStack user %q (ID %s)? This cannot be undone", user.Name, user.Id))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted; rerun with --yes to confirm")
				}
			}

			resp, err := client.OpenStackIdentityDeleteUserWithResponse(cmd.Context(), domainID, user.Id)
			if err != nil {
				return fmt.Errorf("deleting user: %w", err)
			}
			// Success is a body-less 2xx, so check the status code.
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return apiAuthError("deleting user", settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Deleted OpenStack user %q (ID %s)", user.Name, user.Id)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain-id", "", "OpenStack domain ID the user is in (required unless the account has a single domain)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt (required on a non-interactive terminal)")
	_ = cmd.RegisterFlagCompletionFunc("domain-id", completeOpenstackDomains(opts))
	return cmd
}

// resolveRoleIDs maps role names to their IDs for a domain (grant takes role
// UUIDs, not names), reading the domain's roles and delegating the mapping.
func resolveRoleIDs(cmd *cobra.Command, settings config.Settings, client *cleura.Client, domainID string, names []string) ([]string, error) {
	resp, err := client.OpenStackIdentityListRolesWithResponse(cmd.Context(), domainID)
	if err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, apiAuthError("listing roles", settings, resp.HTTPResponse, resp.Body)
	}
	return roleIDsByName(*resp.JSON200, names)
}

// roleIDsByName resolves role names to IDs, erroring on an unknown name with the
// available names listed. Pure, so the mapping is unit-testable without a client.
func roleIDsByName(roles []api.OpenStackIdentityProjectRole, names []string) ([]string, error) {
	byName := make(map[string]string, len(roles))
	available := make([]string, 0, len(roles))
	for _, r := range roles {
		byName[r.Name] = r.Id
		available = append(available, r.Name)
	}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		id, ok := byName[n]
		if !ok {
			slices.Sort(available)
			return nil, fmt.Errorf("unknown role %q; available roles: %s", n, strings.Join(available, ", "))
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// resolveUser resolves a user name-or-ID to the user in a domain, listing the
// domain's users and matching by ID or exact name. There is no single-user GET,
// so a list is the only way to look one up (and it validates existence).
func resolveUser(cmd *cobra.Command, settings config.Settings, client *cleura.Client, domainID, arg string) (*api.OpenStackIdentityUser, error) {
	resp, err := client.OpenStackIdentityListUsersWithResponse(cmd.Context(), domainID)
	if err != nil {
		return nil, fmt.Errorf("looking up user %q: %w", arg, err)
	}
	if resp.JSON200 == nil {
		return nil, apiAuthError("looking up user", settings, resp.HTTPResponse, resp.Body)
	}
	return userByNameOrID(*resp.JSON200, arg)
}

// userByNameOrID finds a user by exact ID or name. Pure, so it is unit-testable.
func userByNameOrID(users []api.OpenStackIdentityUser, arg string) (*api.OpenStackIdentityUser, error) {
	for i := range users {
		if users[i].Id == arg || users[i].Name == arg {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("no OpenStack user with name or ID %q in this domain", arg)
}

// roleNames joins a membership's role names for the table, or "-" when none.
func roleNames(roles *[]api.OpenStackIdentityProjectRole) string {
	if roles == nil || len(*roles) == 0 {
		return "-"
	}
	names := make([]string, 0, len(*roles))
	for _, r := range *roles {
		names = append(names, r.Name)
	}
	slices.Sort(names) // deterministic order for the table and for scripts
	return strings.Join(names, ", ")
}

// requireNonEmpty rejects an empty value for a flag cobra marked required.
// cobra's MarkFlagRequired only checks that a flag was set, so `--flag ""` still
// passes; this closes that gap with a clear usage error before any network work.
func requireNonEmpty(name, val string) error {
	if strings.TrimSpace(val) == "" {
		return fmt.Errorf("--%s must not be empty", name)
	}
	return nil
}

// domainLabel is a human label for a domain in disambiguation/completion output.
// Domain names are frequently not unique (several "CCP_Domain_NNNN" per account),
// so the area (region) is what actually tells them apart.
func domainLabel(d api.CommonOpenStackDomain) string {
	name, area := strDeref(d.Name), d.Area.Name
	switch {
	case name != "" && area != "":
		return name + " — " + area
	case area != "":
		return area
	default:
		return name
	}
}

// renderProjectKV writes a project as key/value rows for the default table
// output; -o json/yaml render the project struct itself.
func renderProjectKV(w io.Writer, p api.OpenStackIdentityProject) error {
	kv := output.NewKVWriter(w)
	kv.Row("ID", p.Id)
	kv.Row("Name", p.Name)
	kv.Row("Domain", p.DomainId)
	kv.Row("Enabled", yesNo(p.Enabled))
	if d := strDeref(p.Description); d != "" {
		kv.Row("Description", d)
	}
	return kv.Flush()
}

// resolveDomain returns the domain ID to operate in. An explicit --domain-id wins;
// otherwise it lists the account's domains and uses the sole one, erroring when
// there is not exactly one so the caller must disambiguate with --domain-id.
func resolveDomain(cmd *cobra.Command, settings config.Settings, client *cleura.Client, flagDomain string) (string, error) {
	if flagDomain != "" {
		return flagDomain, nil
	}
	resp, err := client.OpenStackIdentityListDomainsWithResponse(cmd.Context())
	if err != nil {
		return "", fmt.Errorf("resolving domain: %w", err)
	}
	if resp.JSON200 == nil {
		return "", apiAuthError("resolving domain", settings, resp.HTTPResponse, resp.Body)
	}
	return chooseSoleDomain(*resp.JSON200)
}

// chooseSoleDomain picks the only domain, or returns an actionable error naming
// the choices when there is not exactly one. Kept separate from the API call so
// the selection logic is unit-testable.
func chooseSoleDomain(domains []api.CommonOpenStackDomain) (string, error) {
	switch len(domains) {
	case 0:
		return "", fmt.Errorf("no OpenStack domains available for this account")
	case 1:
		return domains[0].Id, nil
	default:
		choices := make([]string, 0, len(domains))
		for _, d := range domains {
			if label := domainLabel(d); label != "" {
				choices = append(choices, fmt.Sprintf("%s (%s)", d.Id, label))
			} else {
				choices = append(choices, d.Id)
			}
		}
		return "", fmt.Errorf("the account has %d OpenStack domains; select one with --domain-id: %s", len(domains), strings.Join(choices, ", "))
	}
}

// completeOpenstackDomains offers domain IDs (with the name as description) for
// the --domain-id flag.
func completeOpenstackDomains(opts *globalOptions) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		_, settings, err := opts.settings()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		client, err := opts.authenticatedClient(settings)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		resp, err := client.OpenStackIdentityListDomainsWithResponse(cmd.Context())
		if err != nil || resp.JSON200 == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		out := make([]string, 0, len(*resp.JSON200))
		for _, d := range *resp.JSON200 {
			if label := domainLabel(d); label != "" {
				out = append(out, d.Id+"\t"+label)
			} else {
				out = append(out, d.Id)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
