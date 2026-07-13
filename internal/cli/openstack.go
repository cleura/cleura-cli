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
		Short: "Manage OpenStack identity resources (domains, projects)",
		Long: `Manage OpenStack (Keystone) identity resources — domains and projects.

These are distinct from Cleura account users ('cleura user'): OpenStack projects
live under a domain and are addressed by an opaque domain ID. Most accounts have a
single domain, which the CLI selects automatically; pass --domain when there is
more than one (list them with 'cleura openstack domain list').`,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newOpenstackDomainCommand(opts),
		newOpenstackProjectCommand(opts),
	)
	return cmd
}

func newOpenstackDomainCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "View OpenStack domains",
		Long: `View the OpenStack (Keystone) domains available to your account. A domain ID
identifies where projects and users live; it is what 'cleura openstack project
create --domain' expects.`,
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
		Long:    "List the OpenStack (Keystone) domains available to your account, with the domain ID used by 'cleura openstack project create --domain'.",
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
				rows := make([][]string, 0, len(domains))
				for _, d := range domains {
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
when the account has more than one domain, choose one with --domain (list them
with 'cleura openstack domain list').

The created project — including its ID — is printed; use -o json to capture the
ID for scripting (e.g. to grant access or launch resources in it).`,
		Example: `  cleura openstack project create my-project
  cleura openstack project create my-project --description "team sandbox"
  cleura openstack project create my-project --domain <domain-id> -o json`,
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
	cmd.Flags().StringVar(&domain, "domain", "", "OpenStack domain ID to create the project in (default: the account's only domain)")
	cmd.Flags().StringVar(&description, "description", "", "Optional project description")
	_ = cmd.RegisterFlagCompletionFunc("domain", completeOpenstackDomains(opts))
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
	cmd.Flags().StringVar(&domain, "domain", "", "OpenStack domain ID the project is in (default: the account's only domain)")
	cmd.Flags().StringVar(&name, "name", "", "New project name")
	cmd.Flags().StringVar(&description, "description", "", "New project description (pass \"\" to clear)")
	cmd.Flags().BoolVar(&enable, "enable", false, "Enable the project")
	cmd.Flags().BoolVar(&disable, "disable", false, "Disable the project (the closest thing to deletion)")
	cmd.MarkFlagsMutuallyExclusive("enable", "disable")
	_ = cmd.RegisterFlagCompletionFunc("domain", completeOpenstackDomains(opts))
	addOutputFlag(cmd, opts)
	return cmd
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

// resolveDomain returns the domain ID to operate in. An explicit --domain wins;
// otherwise it lists the account's domains and uses the sole one, erroring when
// there is not exactly one so the caller must disambiguate with --domain.
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
			if n := strDeref(d.Name); n != "" {
				choices = append(choices, fmt.Sprintf("%s (%s)", d.Id, n))
			} else {
				choices = append(choices, d.Id)
			}
		}
		return "", fmt.Errorf("the account has %d OpenStack domains; select one with --domain: %s", len(domains), strings.Join(choices, ", "))
	}
}

// completeOpenstackDomains offers domain IDs (with the name as description) for
// the --domain flag.
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
			if n := strDeref(d.Name); n != "" {
				out = append(out, d.Id+"\t"+n)
			} else {
				out = append(out, d.Id)
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}
