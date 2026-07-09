package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

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
		newConfigPathCommand(),
		newGetCredentialsCommand(opts),
		newProfileCommand(opts),
	)
	return cmd
}

// newProfileCommand groups the profile-management verbs under
// 'cleura config profile', matching the noun-verb shape of 'gardener shoot'
// and 'user'. (Whole-config commands — view, path, get-credentials — stay at
// the config level.)
func newProfileCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage named profiles (list, use, set, rename, delete)",
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(
		newListProfilesCommand(opts),
		newConfigCurrentCommand(opts),
		newUseProfileCommand(opts),
		newConfigSetCommand(opts),
		newRenameProfileCommand(opts),
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

	cmd := &cobra.Command{
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
			// The diagnostic command should not silently hide a typo'd
			// --profile behind a defaults table.
			if !s.ProfileExists && s.ProfileName != "default" {
				fmt.Fprintf(cmd.ErrOrStderr(), "note: profile %q does not exist; showing defaults\n", s.ProfileName)
			}

			endpoint, err := s.ResolveURL()
			if err != nil {
				endpoint = "(" + err.Error() + ")"
			}

			token := "(not set)"
			if s.Token != "" {
				token = "(set)"
			}
			tokenStoredAt, tokenStoredAtSource := "", ""
			if !s.TokenStoredAt.IsZero() {
				tokenStoredAt = s.TokenStoredAt.UTC().Format(time.RFC3339)
				tokenStoredAtSource = "profile"
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
				{"token_stored_at", tokenStoredAt, tokenStoredAtSource},
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
	addOutputFlag(cmd, opts)
	// config view explains how settings resolve, so it accepts every
	// resolution input to preview it — including --region/--project-id,
	// which are otherwise gardener/login-scoped. (--profile/--cloud/--api-url
	// are already global.)
	addProjectContextFlags(cmd, opts, false)
	return cmd
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
		Example: "  cleura config profile set region kna1\n  cleura config profile set project_id a1b2c3\n  cleura config profile set --profile acme api_url https://rest.cloud.acme.example",
		Args:    cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return keys, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Accept flag spelling too: --project-id the flag, project_id the key.
			key, value := strings.ReplaceAll(args[0], "-", "_"), args[1]
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
			existing, exists := cfg.Profiles[name]
			if value == "" && !exists {
				// Unsetting a value on a profile that does not exist must not
				// materialize an empty one.
				opts.infof(cmd, "Profile %q does not exist; nothing to unset", name)
				return nil
			}
			// Changing the username on a profile that holds a token desyncs
			// the pair (the token still belongs to the old username).
			if key == "username" && existing != nil && existing.Token != "" && value != existing.Username {
				opts.warnf(cmd, "profile %q has a stored token for %q; changing the username may leave the token mismatched — run 'cleura login' to refresh it", name, existing.Username)
			}
			set(cfg.Profile(name), value)
			if err := cfg.Save(); err != nil {
				return err
			}
			// A typo in --profile would otherwise mint a phantom silently.
			if !exists {
				opts.infof(cmd, "Created profile %q", name)
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
		Long: `Print the configuration file path: $CLEURA_CONFIG if set, otherwise
$XDG_CONFIG_HOME/cleura/config.yaml, otherwise ~/.config/cleura/config.yaml
(the OS config directory on Windows).`,
		Example: "  cleura config path",
		Args:    cobra.NoArgs,
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

func newConfigCurrentCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Print the profile the next command would use",
		Long: `Print the name of the profile that commands resolve to right now
(--profile / $CLEURA_PROFILE / current_profile / "default"), without a network
call — the quick answer to "which profile am I on?".`,
		Example: "  cleura config profile current",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, s, err := opts.settings()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), s.ProfileName)
			return nil
		},
	}
}

func newRenameProfileCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a profile",
		Long: `Rename a profile, keeping its stored token and settings (the token
belongs to the same account, so no re-login is needed). current_profile follows
the rename. Refuses if the new name already exists.`,
		Example:           "  cleura config profile rename default work",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeProfileNameArg,
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			if oldName == newName {
				return fmt.Errorf("old and new names are the same")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, ok := cfg.Profiles[oldName]
			if !ok {
				return fmt.Errorf("profile %q does not exist%s", oldName, availableProfiles(cfg))
			}
			if _, exists := cfg.Profiles[newName]; exists {
				return fmt.Errorf("profile %q already exists; choose another name or delete it first", newName)
			}
			if p == nil { // hand-edited nil entry
				p = &config.Profile{}
			}
			cfg.Profiles[newName] = p
			delete(cfg.Profiles, oldName)
			if cfg.CurrentProfile == oldName {
				cfg.CurrentProfile = newName
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Renamed profile %q to %q", oldName, newName)
			return nil
		},
	}
}

func newUseProfileCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:               "use <name>",
		Short:             "Set the current profile",
		Example:           "  cleura config profile use compliant",
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

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List configured profiles",
		Example: "  cleura config profile list\n  cleura config profile list -o json   # tokens are never included",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			// The CURRENT (*) column marks the stored current_profile — what
			// the config says — not whatever a --profile/env override selects
			// for this one run; that override is surfaced separately below.
			current := cfg.CurrentProfile
			effective := cfg.ProfileName(config.Flags{Profile: opts.profile})

			if _, ok := cfg.Profiles[cfg.CurrentProfile]; cfg.CurrentProfile != "" && !ok {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: current_profile %q does not exist in the config file\n", cfg.CurrentProfile)
			}
			// Only when a real stored current exists and something selected a
			// different one this run — not when current_profile is empty (there
			// is nothing to override then, e.g. right after deleting it).
			if current != "" && effective != current {
				opts.infof(cmd, "selected for this run: %q (overrides current_profile %q)", effective, current)
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
	addOutputFlag(cmd, opts)
	return cmd
}

func newDeleteProfileCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Remove a profile from the configuration",
		Long: `Remove a profile from the configuration file. A stored token is revoked
server-side first (best effort — the profile is deleted even when revocation
fails, and the warning says so).`,
		Example:           "  cleura config profile delete old-test",
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
			clearedCurrent := cfg.CurrentProfile == name
			if clearedCurrent {
				cfg.CurrentProfile = ""
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Deleted profile %q", name)
			// Deleting the current profile leaves no current one; guide the
			// user rather than silently falling back to a nonexistent default.
			if clearedCurrent {
				switch remaining := cfg.ProfileNames(); len(remaining) {
				case 0:
				case 1:
					opts.infof(cmd, "No current profile; select the remaining one with 'cleura config profile use %s'", remaining[0])
				default:
					opts.infof(cmd, "No current profile; select one with 'cleura config profile use <name>' (%s)", strings.Join(remaining, ", "))
				}
			}
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
