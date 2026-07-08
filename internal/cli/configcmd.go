package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration and profiles",
		// Fail loudly on typo'd subcommands instead of exit-0 help output.
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(
		newConfigViewCommand(opts),
		newConfigSetCommand(opts),
		newConfigPathCommand(),
		newUseProfileCommand(opts),
		newListProfilesCommand(opts),
		newDeleteProfileCommand(opts),
	)
	return cmd
}

func newConfigViewCommand(opts *globalOptions) *cobra.Command {
	// settingRow is one resolved value with its origin; the same shape renders
	// as the table and as json/yaml. The token value is always redacted.
	type settingRow struct {
		Setting string `json:"setting" yaml:"setting"`
		Value   string `json:"value" yaml:"value"`
		Source  string `json:"source" yaml:"source"`
	}

	return &cobra.Command{
		Use:   "view",
		Short: "Show the effective settings and where each value comes from",
		Long: `Show the settings the next command would use, resolved with the usual
precedence (flags > environment > profile > defaults), and the source of each
value. Values set by environment variables that shadow something stored in the
selected profile are pointed out on stderr. The token value is never shown.`,
		Example: "  cleura config view\n  cleura config view --profile compliant -o json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, s, err := opts.settings()
			if err != nil {
				return err
			}

			endpoint, err := s.ResolveURL()
			if err != nil {
				endpoint = "(" + err.Error() + ")"
			}

			token := "(not set)"
			if s.Token != "" {
				token = "(set)"
			}

			path, pathSource, err := config.PathWithSource()
			if err != nil {
				return err
			}

			rows := []settingRow{
				{"profile", s.ProfileName, s.Sources.Profile},
				{"endpoint", endpoint, s.Sources.Endpoint},
				{"cloud", s.Cloud, s.Sources.Cloud},
				{"api_url", s.APIURL, s.Sources.APIURL},
				{"username", s.Username, s.Sources.Username},
				{"token", token, s.Sources.Token},
				{"region", s.Region, s.Sources.Region},
				{"project_id", s.ProjectID, s.Sources.ProjectID},
				{"config_file", path, pathSource},
			}

			// Point out environment values that shadow something stored in the
			// profile — the failure mode that motivated this command.
			if profile := cfg.Profiles[s.ProfileName]; profile != nil {
				shadowed := []struct{ source, stored string }{
					{s.Sources.Cloud, profile.Cloud},
					{s.Sources.APIURL, profile.APIURL},
					{s.Sources.Username, profile.Username},
					{s.Sources.Token, profile.Token},
					{s.Sources.Region, profile.Region},
					{s.Sources.ProjectID, profile.ProjectID},
				}
				for _, sh := range shadowed {
					if strings.HasPrefix(sh.source, "$") && sh.stored != "" {
						fmt.Fprintf(cmd.ErrOrStderr(), "note: %s overrides a value stored in profile %q\n", sh.source, s.ProfileName)
					}
				}
			}

			return output.Render(cmd.OutOrStdout(), opts.output, rows, func(w io.Writer) error {
				display := make([][]string, 0, len(rows))
				for _, r := range rows {
					value := r.Value
					if value == "" {
						value = "(not set)"
					}
					display = append(display, []string{r.Setting, value, r.Source})
				}
				return output.Table(w, []string{"SETTING", "VALUE", "SOURCE"}, display)
			})
		},
	}
}

// settableKeys are the profile fields "config set" may write. The token is
// excluded on purpose: it is set by login, which validates it.
var settableKeys = map[string]func(p *config.Profile, value string){
	"cloud":      func(p *config.Profile, v string) { p.Cloud = v },
	"api_url":    func(p *config.Profile, v string) { p.APIURL = v },
	"username":   func(p *config.Profile, v string) { p.Username = v },
	"region":     func(p *config.Profile, v string) { p.Region = v },
	"project_id": func(p *config.Profile, v string) { p.ProjectID = v },
}

func newConfigSetCommand(opts *globalOptions) *cobra.Command {
	keys := make([]string, 0, len(settableKeys))
	for k := range settableKeys {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a profile value without logging in again",
		Long: fmt.Sprintf(`Set one value in the selected profile. Keys: %s.
An empty value ("") removes the stored value. Tokens cannot be set here; use
'cleura login' (or 'cleura login --token-stdin').`, strings.Join(keys, ", ")),
		Example: "  cleura config set region kna1\n  cleura config set project_id a1b2c3\n  cleura config set --profile acme api_url https://rest.cloud.acme.example",
		Args:    cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return keys, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if key == "token" {
				return fmt.Errorf("tokens cannot be set directly; run 'cleura login' or 'cleura login --token-stdin'")
			}
			set, ok := settableKeys[key]
			if !ok {
				return fmt.Errorf("unknown key %q (expected one of: %s)", key, strings.Join(keys, ", "))
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := cfg.ProfileName(config.Flags{Profile: opts.profile})
			set(cfg.Profile(name), value)
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Set %s = %q in profile %q", key, value, name)
			return nil
		},
	}
}

func newConfigPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the configuration file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newUseProfileCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:               "use-profile <name>",
		Short:             "Set the current profile",
		Example:           "  cleura config use-profile compliant",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNameArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			profile, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q does not exist (create it with 'cleura login --profile %s')%s",
					name, name, availableProfiles(cfg))
			}
			if profile == nil { // hand-edited config: a profile key with no value
				profile = &config.Profile{}
			}
			cfg.CurrentProfile = name
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Switched to profile %q (%s)", name, endpointLabel(profile.Cloud, profile.APIURL))
			return nil
		},
	}
}

func newListProfilesCommand(opts *globalOptions) *cobra.Command {
	// profileView is what list-profiles exposes; deliberately without the token.
	type profileView struct {
		Name      string `json:"name" yaml:"name"`
		Current   bool   `json:"current" yaml:"current"`
		Cloud     string `json:"cloud,omitempty" yaml:"cloud,omitempty"`
		APIURL    string `json:"api_url,omitempty" yaml:"api_url,omitempty"`
		Username  string `json:"username,omitempty" yaml:"username,omitempty"`
		Region    string `json:"region,omitempty" yaml:"region,omitempty"`
		ProjectID string `json:"project_id,omitempty" yaml:"project_id,omitempty"`
		LoggedIn  bool   `json:"logged_in" yaml:"logged_in"`
	}

	return &cobra.Command{
		Use:   "list-profiles",
		Short: "List configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			current := cfg.ProfileName(config.Flags{Profile: opts.profile})

			if cfg.CurrentProfile != "" && cfg.Profiles[cfg.CurrentProfile] == nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: current_profile %q does not exist in the config file\n", cfg.CurrentProfile)
			}

			names := cfg.ProfileNames()

			views := make([]profileView, 0, len(names))
			for _, name := range names {
				p := cfg.Profiles[name]
				if p == nil { // hand-edited config: a profile key with no value
					p = &config.Profile{}
				}
				views = append(views, profileView{
					Name:      name,
					Current:   name == current,
					Cloud:     p.Cloud,
					APIURL:    p.APIURL,
					Username:  p.Username,
					Region:    p.Region,
					ProjectID: p.ProjectID,
					LoggedIn:  p.Token != "",
				})
			}

			header := []string{"CURRENT", "NAME", "ENDPOINT", "USERNAME", "LOGGED IN"}
			return output.Render(cmd.OutOrStdout(), opts.output, views, func(w io.Writer) error {
				if len(views) == 0 {
					opts.infof(cmd, "No profiles configured yet; run 'cleura login' to create one.")
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(views))
				for _, v := range views {
					marker := ""
					if v.Current {
						marker = "*"
					}
					loggedIn := "no"
					if v.LoggedIn {
						loggedIn = "yes"
					}
					rows = append(rows, []string{marker, v.Name, endpointLabel(v.Cloud, v.APIURL), v.Username, loggedIn})
				}
				return output.Table(w, header, rows)
			})
		},
	}
}

func newDeleteProfileCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:               "delete-profile <name>",
		Short:             "Remove a profile from the configuration",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfileNameArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			profile, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q does not exist%s", name, availableProfiles(cfg))
			}
			// Best-effort revocation before the token is lost with the
			// profile; afterwards nothing could revoke it anymore.
			if profile != nil && profile.Token != "" {
				if err := revokeProfileToken(cmd.Context(), opts, profile); err != nil {
					opts.warnf(cmd, "could not revoke the profile's token (%v); deleting the profile anyway — the token stays valid until it expires", err)
				} else {
					opts.infof(cmd, "Revoked the stored token")
				}
			}
			delete(cfg.Profiles, name)
			if cfg.CurrentProfile == name {
				cfg.CurrentProfile = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Deleted profile %q", name)
			return nil
		},
	}
}

// completeProfileNameArg completes the first positional argument with the
// configured profile names.
func completeProfileNameArg(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeProfileNames(cmd, args, toComplete)
}

// endpointLabel describes which API a profile points at, for human output.
func endpointLabel(cloud, apiURL string) string {
	switch {
	case apiURL != "":
		return apiURL
	case cloud != "":
		return cloud
	default:
		return "public (default)"
	}
}

func availableProfiles(cfg *config.Config) string {
	if len(cfg.Profiles) == 0 {
		return ""
	}
	return "; available: " + strings.Join(cfg.ProfileNames(), ", ")
}
