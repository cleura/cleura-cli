// Package cli implements the cleura command tree.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	opts := &globalOptions{version: version}

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

	pf := root.PersistentFlags()
	pf.StringVar(&opts.profile, "profile", "", "Configuration profile to use [$CLEURA_PROFILE] (default from config, or \"default\")")
	pf.StringVar(&opts.cloud, "cloud", "", "Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]")
	pf.StringVar(&opts.apiURL, "api-url", "", "Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]")
	pf.StringVar(&opts.region, "region", "", "OpenStack region (e.g. sto1) [$CLEURA_REGION]")
	pf.StringVar(&opts.projectID, "project-id", "", "OpenStack project ID [$CLEURA_PROJECT_ID]")
	pf.StringVarP(&opts.output, "output", "o", "table", "Output format: "+output.Formats)
	pf.BoolVar(&opts.debug, "debug", false, "Log HTTP requests and responses to stderr (credentials redacted)")
	pf.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress informational messages; errors and requested output are still shown")

	// Define --version ourselves so cobra does not claim the -v shorthand,
	// which stays reserved for a possible --verbose.
	root.Flags().Bool("version", false, "Show the cleura version")

	root.AddCommand(
		newLoginCommand(opts),
		newLogoutCommand(opts),
		newWhoamiCommand(opts),
		newConfigCommand(opts),
		newGardenerCommand(opts),
		newVersionCommand(opts),
	)

	_ = root.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(output.FormatList, cobra.ShellCompDirectiveNoFileComp))
	_ = root.RegisterFlagCompletionFunc("cloud", cobra.FixedCompletions([]string{"public", "compliant"}, cobra.ShellCompDirectiveNoFileComp))
	_ = root.RegisterFlagCompletionFunc("profile", completeProfileNames)

	return root
}

func newVersionCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
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
	hint := fmt.Sprintf("the token may be expired or revoked — run 'cleura login' to refresh profile %q", s.ProfileName)
	if s.Sources.Token == "$CLEURA_API_TOKEN" {
		hint += "; note that CLEURA_API_TOKEN is set in your environment and overrides the profile's stored token"
	}
	return fmt.Errorf("%w\n%s", err, hint)
}
