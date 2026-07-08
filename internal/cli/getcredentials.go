package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// ExitCodeError carries a specific process exit code through cobra to main.
type ExitCodeError struct {
	Code int
	Err  error
}

func (e *ExitCodeError) Error() string { return e.Err.Error() }
func (e *ExitCodeError) Unwrap() error { return e.Err }

// credentialsEnvelope is the stable tool-integration contract emitted by
// "cleura config get-credentials". Compatibility rules: fields are only
// added, never renamed or removed, while Version stays 1; a breaking change
// bumps Version. Consumers must check Version.
type credentialsEnvelope struct {
	Version       int    `json:"version"`
	Profile       string `json:"profile"`
	Cloud         string `json:"cloud"`
	Endpoint      string `json:"endpoint"`
	Username      string `json:"username"`
	Token         string `json:"token"`
	Region        string `json:"region,omitempty"`
	ProjectID     string `json:"project_id,omitempty"`
	TokenStoredAt string `json:"token_stored_at,omitempty"`
}

func newGetCredentialsCommand(opts *globalOptions) *cobra.Command {
	var validate bool

	cmd := &cobra.Command{
		Use:   "get-credentials",
		Short: "Print the effective credentials as JSON, for tool integration",
		Long: `Print the effective credentials (resolved with the usual precedence:
flags > environment > profile) as a single JSON object on stdout. This is the
stable interface for tools that authenticate via the cleura CLI, such as the
Terraform provider — the config file itself is internal and must not be parsed.

Output is always JSON; --output does not apply. The token is printed in the
clear — that is this command's purpose.

Exit codes are part of the contract:
  0  credentials printed
  2  no usable credentials for the selected profile (a JSON {"error": ...}
     object is printed on stdout so callers can fall through to their next
     credential source)
  1  anything else went wrong (unreadable config, validation transport error)

Envelope fields are only added, never renamed or removed, while "version" is 1;
a breaking change bumps it. Consumers must check the version field.`,
		Example: `  cleura config get-credentials
  cleura config get-credentials --profile compliant --validate
  cleura config get-credentials | jq -r .token`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := opts.settings()
			if err != nil {
				return err
			}

			noCredentials := func(reason string, a ...any) error {
				msg := fmt.Sprintf(reason, a...)
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				_ = enc.Encode(map[string]string{"error": msg})
				return &ExitCodeError{Code: 2, Err: fmt.Errorf("%s", msg)}
			}

			if settings.Token == "" {
				return noCredentials("no credentials: profile %q has no token; run 'cleura login'", settings.ProfileName)
			}
			if settings.Username == "" {
				return noCredentials("no credentials: a token is set but no username; the API requires both")
			}
			endpoint, err := settings.ResolveURL()
			if err != nil {
				return err
			}

			if validate {
				client, err := opts.authenticatedClient(settings)
				if err != nil {
					return err
				}
				resp, err := client.AuthValidateTokenWithResponse(cmd.Context())
				if err != nil {
					return fmt.Errorf("validating token: %w", err)
				}
				if resp.StatusCode() != 204 {
					return noCredentials("no credentials: the token was rejected by the API (%s); run 'cleura login'", resp.Status())
				}
			}

			envelope := credentialsEnvelope{
				Version:   1,
				Profile:   settings.ProfileName,
				Cloud:     settings.Cloud,
				Endpoint:  endpoint,
				Username:  settings.Username,
				Token:     settings.Token,
				Region:    settings.Region,
				ProjectID: settings.ProjectID,
			}
			if !settings.TokenStoredAt.IsZero() {
				envelope.TokenStoredAt = settings.TokenStoredAt.UTC().Format(time.RFC3339)
			}

			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(envelope)
		},
	}

	cmd.Flags().BoolVar(&validate, "validate", false, "Verify the token against the API before printing (one extra request)")

	return cmd
}
