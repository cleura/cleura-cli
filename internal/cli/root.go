// Package cli implements the cleura command tree.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

// requestTimeout bounds every API call so a hung endpoint cannot stall the
// CLI indefinitely.
const requestTimeout = 60 * time.Second

type globalOptions struct {
	version   string
	profile   string
	cloud     string
	apiURL    string
	region    string
	projectID string
	output    string
	debug     bool
	quiet     bool
}

func NewRootCommand(version string) *cobra.Command {
	// output defaults to table; commands that don't render never bind the
	// -o flag, so this is the value they carry (harmlessly) too.
	opts := &globalOptions{version: version, output: "table"}

	root := &cobra.Command{
		Use:   "cleura",
		Short: "Command-line interface for Cleura Cloud",
		Long: `cleura is the command-line interface for Cleura Cloud.

Run 'cleura login' first: it stores an API token in a profile in
~/.config/cleura/config.yaml (override with $CLEURA_CONFIG). Settings resolve
with the precedence flags > environment variables > profile > defaults; run
'cleura config view' to see the effective values and where each one comes
from.`,
		Version: version,
		// Errors are printed by main. Usage is shown for parse and argument
		// errors (which happen before PersistentPreRunE) but suppressed for
		// runtime errors, where it would be noise.
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			if !output.Valid(opts.output) {
				return fmt.Errorf("unknown output format %q (expected one of: %s)", opts.output, output.Formats)
			}
			return nil
		},
	}

	// Genuinely global: every command reads a profile and resolves an
	// endpoint; --debug/--quiet are cross-cutting. --region/--project-id and
	// -o are NOT global — they are attached only to the commands that use
	// them (gardener + login store the former; renderers take the latter),
	// so they never appear on commands that would silently ignore them.
	pf := root.PersistentFlags()
	pf.StringVar(&opts.profile, "profile", "", "Configuration profile to use [$CLEURA_PROFILE] (default from config, or \"default\")")
	pf.StringVar(&opts.cloud, "cloud", "", "Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]")
	pf.StringVar(&opts.apiURL, "api-url", "", "Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]")
	pf.BoolVar(&opts.debug, "debug", false, "Log HTTP requests and responses to stderr (credentials redacted)")
	pf.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress informational messages; errors and requested output are still shown")

	// Define --version ourselves so cobra does not claim the -v shorthand,
	// which stays reserved for a possible --verbose.
	root.Flags().Bool("version", false, "Show the cleura version")

	root.AddCommand(
		newLoginCommand(opts),
		newLogoutCommand(opts),
		newWhoamiCommand(opts),
		newUserCommand(opts),
		newConfigCommand(opts),
		newGardenerCommand(opts),
		newVersionCommand(opts),
	)

	_ = root.RegisterFlagCompletionFunc("cloud", cobra.FixedCompletions([]string{"public", "compliant"}, cobra.ShellCompDirectiveNoFileComp))
	_ = root.RegisterFlagCompletionFunc("profile", completeProfileNames)

	return root
}

// addOutputFlag attaches -o/--output (and its completion) to a command that
// renders structured output. It is deliberately not global: action-only
// commands ignore output formatting, and offering the flag there is
// misleading. All render commands bind the same opts.output field.
func addOutputFlag(cmd *cobra.Command, opts *globalOptions) {
	cmd.Flags().StringVarP(&opts.output, "output", "o", "table", "Output format: "+output.Formats)
	_ = cmd.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(output.FormatList, cobra.ShellCompDirectiveNoFileComp))
}

// addProjectContextFlags attaches --region/--project-id to a command. They
// are not global: only gardener (persistent, inherited by its subcommands)
// uses them at call time, and login (local) stores them in the profile.
func addProjectContextFlags(cmd *cobra.Command, opts *globalOptions, persistent bool) {
	fs := cmd.Flags()
	if persistent {
		fs = cmd.PersistentFlags()
	}
	fs.StringVar(&opts.region, "region", "", "OpenStack region (e.g. sto1) [$CLEURA_REGION]")
	fs.StringVar(&opts.projectID, "project-id", "", "OpenStack project ID [$CLEURA_PROJECT_ID]")
}

func newVersionCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the cleura version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := struct {
				Version string `json:"version" yaml:"version"`
			}{opts.version}
			return output.Render(cmd.OutOrStdout(), opts.output, v, func(w io.Writer) error {
				fmt.Fprintf(w, "cleura version %s\n", v.Version)
				return nil
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

// groupHelp is the RunE for parent commands that only route to subcommands.
// Making them runnable is what activates their Args validation: without it,
// cobra prints help with exit 0 for typo'd subcommands, which would let CI
// pipelines pass silently. Paired with Args: cobra.NoArgs, a bare invocation
// shows help and an unknown subcommand fails with exit 1.
func groupHelp(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

// infof prints an informational message to stderr unless --quiet is set.
// Informational output never goes to stdout, which carries only data.
func (o *globalOptions) infof(cmd *cobra.Command, format string, a ...any) {
	if o.quiet {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", a...)
}

// warnf prints a warning to stderr. Unlike infof, warnings are not silenced
// by --quiet: they signal degraded results that scripts should still see.
func (o *globalOptions) warnf(cmd *cobra.Command, format string, a ...any) {
	fmt.Fprintf(cmd.ErrOrStderr(), "Warning: "+format+"\n", a...)
}

// settings loads the config file and resolves effective settings for this
// invocation (flags > environment > profile > defaults).
func (o *globalOptions) settings() (*config.Config, config.Settings, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, config.Settings{}, err
	}
	s := cfg.Resolve(config.Flags{
		Profile:   o.profile,
		Cloud:     o.cloud,
		APIURL:    o.apiURL,
		Region:    o.region,
		ProjectID: o.projectID,
	})
	return cfg, s, nil
}

// clientOptions returns the API client options every command uses: a request
// timeout, a User-Agent identifying the CLI and version, and the --debug
// transport when enabled.
func (o *globalOptions) clientOptions() []api.ClientOption {
	httpClient := &http.Client{Timeout: requestTimeout}
	if o.debug {
		httpClient.Transport = newDebugTransport(os.Stderr)
	}
	userAgent := "cleura-cli/" + o.version
	return []api.ClientOption{
		api.WithHTTPClient(httpClient),
		api.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			req.Header.Set("User-Agent", userAgent)
			return nil
		}),
	}
}

// authenticatedClient returns a client with credentials from the settings.
func (o *globalOptions) authenticatedClient(s config.Settings) (*cleura.Client, error) {
	if s.Token == "" {
		if !s.ProfileExists && s.ProfileName != "default" {
			return nil, fmt.Errorf("profile %q does not exist; run 'cleura config list-profiles' to see profiles, or 'cleura login --profile %s' to create it", s.ProfileName, s.ProfileName)
		}
		return nil, fmt.Errorf("not logged in (profile %q); run 'cleura login' or set CLEURA_API_USERNAME and CLEURA_API_TOKEN", s.ProfileName)
	}
	if s.Username == "" {
		return nil, fmt.Errorf("a token is set but no username; set CLEURA_API_USERNAME to the account the token belongs to (the API requires both)")
	}
	url, err := s.ResolveURL()
	if err != nil {
		return nil, err
	}
	return cleura.NewClientWithCredentials(url, s.Username, s.Token, o.clientOptions()...)
}

// completeProfileNames offers the configured profile names, for --profile and
// profile-name arguments.
func completeProfileNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return cfg.ProfileNames(), cobra.ShellCompDirectiveNoFileComp
}

// apiError builds an error from a non-success API response, using the API's
// error message when the body contains one.
func apiError(op string, resp *http.Response, body []byte) error {
	var e api.FrameworkHttpErrorResponse
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return fmt.Errorf("%s: %s (HTTP %d)", op, e.Error.Message, resp.StatusCode)
	}
	return fmt.Errorf("%s: unexpected response %s", op, resp.Status)
}

// apiAuthError is apiError for calls made with stored/env credentials: a
// 401/403 gets a recovery hint. Not for use inside the login flow itself,
// where "run 'cleura login'" would be circular.
func apiAuthError(op string, s config.Settings, resp *http.Response, body []byte) error {
	err := apiError(op, resp, body)
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		return err
	}
	// A 403 can also mean missing privileges ("No access: ..."), where
	// re-logging in cannot help; hint only when the API blames the token
	// itself, or when the body carries no message to judge by.
	var e api.FrameworkHttpErrorResponse
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" && !strings.Contains(strings.ToLower(e.Error.Message), "token") {
		return err
	}
	hint := fmt.Sprintf("the token may be expired or revoked — run 'cleura login' to refresh profile %q", s.ProfileName)
	if s.Sources.Token == "profile" && !s.TokenStoredAt.IsZero() {
		hint = fmt.Sprintf("the token was stored %s ago and tokens are short-lived — run 'cleura login' to refresh profile %q", humanAge(time.Since(s.TokenStoredAt)), s.ProfileName)
	}
	if s.Sources.Token == "$CLEURA_API_TOKEN" {
		hint += "; note that CLEURA_API_TOKEN is set in your environment and overrides the profile's stored token"
	}
	return fmt.Errorf("%w\n%s", err, hint)
}

// humanAge renders a duration as a rough human quantity for diagnostics.
func humanAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
