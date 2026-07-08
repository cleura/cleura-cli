package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/cleura/cleura-cli/internal/config"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func newLoginCommand(opts *globalOptions) *cobra.Command {
	var username string
	var tokenStdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Cleura Cloud and store an API token",
		Long: `Log in to Cleura Cloud with username and password and store the resulting
API token in the configuration file. The profile you log in to becomes the
current profile.

Logging in again with the same identity refreshes the profile's token. When
the login would replace a different identity (another username, cloud or
endpoint) in an existing profile, confirmation is required — use --profile
to keep identities in separate profiles instead.

SMS is the only two-factor method the CLI supports; accounts with SMS 2FA are
prompted for the code and must log in from an interactive terminal. WebAuthn
accounts can create an API token in the Control Panel instead and store it
with --token-stdin.

For non-interactive use (CI), set CLEURA_API_PASSWORD in the environment — no
prompt, no secrets on the command line (single-factor accounts only). The
password can also be piped on stdin. Alternatively, store a pre-created API
token with --token-stdin (validated before storing).`,
		Example: `  cleura login
  cleura login --profile compliant --cloud compliant
  CLEURA_API_PASSWORD=... cleura login -u johndoe     # CI: set as a masked variable
  echo "$TOKEN" | cleura login -u johndoe --token-stdin`,
		// Secrets are never accepted as arguments; catch the natural mistake
		// of passing the token on the command line with a purpose-built hint.
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("login takes no arguments (got %q) — secrets are never passed on the command line; pipe them instead: echo \"$TOKEN\" | cleura login --token-stdin", args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			cfg, settings, err := opts.settings()
			if err != nil {
				return err
			}
			url, err := settings.ResolveURL()
			if err != nil {
				return err
			}

			prompt := newPrompter(cmd)

			if username == "" {
				username = settings.Username
			}
			if username == "" {
				if username, err = prompt.line(ctx, "Username"); err != nil {
					return err
				}
			}
			if username == "" {
				return fmt.Errorf("username must not be empty")
			}

			// Same identity = token refresh, silent. A different identity
			// (user or endpoint) would destroy the stored login — ask first,
			// and before any password/SMS round-trip. Non-interactive runs
			// refuse instead of prompting.
			newCloud, newAPIURL := persistedEndpoint(settings)
			replacing := false
			if existing := cfg.Profiles[settings.ProfileName]; existing != nil && existing.Token != "" &&
				(existing.Username != username || existing.Cloud != newCloud || existing.APIURL != newAPIURL) {
				replacing = true
				situation := fmt.Sprintf("profile %q is currently logged in as %s on %s; this login is %s on %s",
					settings.ProfileName,
					existing.Username, endpointLabel(existing.Cloud, existing.APIURL),
					username, endpointLabel(newCloud, newAPIURL))
				ok, err := prompt.confirm(ctx, situation+".\nReplace it?")
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("%s — refusing to overwrite; use --profile <name> to log in to a separate profile", situation)
				}
			}

			var token string
			if tokenStdin {
				token, err = loginWithProvidedToken(ctx, opts, prompt, url, username)
			} else {
				token, err = loginWithPassword(ctx, opts, prompt, url, username)
			}
			if err != nil {
				return err
			}

			profile := cfg.Profile(settings.ProfileName)
			profile.Username = username
			profile.Token = token
			// Tokens are short-lived and the API exposes no expiry; the
			// storage time is the only basis for staleness diagnostics.
			profile.TokenStoredAt = time.Now().UTC().Truncate(time.Second)
			// Record the endpoint the token was created against, however it
			// was selected, so later commands reach the same API.
			profile.Cloud, profile.APIURL = persistedEndpoint(settings)
			// A confirmed identity replacement resets the contextual fields:
			// the old identity's region/project do not apply to the new one.
			if replacing {
				profile.Region, profile.ProjectID = "", ""
			}
			// Convenience context worth remembering, but never blank stored
			// values on a plain re-login.
			if settings.Region != "" {
				profile.Region = settings.Region
			}
			if settings.ProjectID != "" {
				profile.ProjectID = settings.ProjectID
			}
			// The profile you log in to becomes the current profile — the
			// az/gcloud/kubectl convention: acquiring credentials activates
			// them. One-off use of another profile is what --profile is for.
			previous := cfg.CurrentProfile
			cfg.CurrentProfile = settings.ProfileName
			if err := cfg.Save(); err != nil {
				return err
			}

			opts.infof(cmd, "Logged in as %s (profile %q, %s)", username, settings.ProfileName, url)
			if previous != "" && previous != settings.ProfileName {
				opts.infof(cmd, "Current profile is now %q (was %q); switch back with 'cleura config use-profile %s'", settings.ProfileName, previous, previous)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "Username to log in with [$CLEURA_API_USERNAME]")
	cmd.Flags().BoolVar(&tokenStdin, "token-stdin", false, "Read an existing API token from stdin and store it instead of logging in with a password")

	return cmd
}

// persistedEndpoint is the cloud/api_url pair a login stores. A cloud that
// was only the built-in default must not be stored next to an explicit
// endpoint override: {api_url}-only profiles keep cloud unset, so a later
// --cloud public is never paired with the private URL.
func persistedEndpoint(s config.Settings) (cloud, apiURL string) {
	if s.APIURL != "" && s.Sources.Cloud == "default" {
		return "", s.APIURL
	}
	return s.Cloud, s.APIURL
}

// loginWithPassword runs the token creation flow, including SMS two-factor
// authentication. The password comes from CLEURA_API_PASSWORD when set —
// the recommended CI path, keeping secrets off command lines and out of
// shell pipes — and is prompted for otherwise.
func loginWithPassword(ctx context.Context, opts *globalOptions, prompt *prompter, url, username string) (string, error) {
	password := os.Getenv("CLEURA_API_PASSWORD")
	if password == "" {
		var err error
		if password, err = prompt.secret(ctx, "Password"); err != nil {
			return "", err
		}
	}
	if password == "" {
		return "", fmt.Errorf("password must not be empty")
	}

	client, err := cleura.NewClient(url, opts.clientOptions()...)
	if err != nil {
		return "", err
	}

	body := api.AuthCreateTokenJSONRequestBody{Login: username, Password: password}
	// The flow needs at most two passes: a twofactor_options response is
	// answered by re-posting with a chosen method, which then yields
	// twofactor_required.
	for range 2 {
		resp, err := client.AuthCreateTokenWithResponse(ctx, body)
		if err != nil {
			return "", fmt.Errorf("logging in: %w", err)
		}
		if resp.JSON201 == nil {
			return "", apiError("login", resp.HTTPResponse, resp.Body)
		}
		result := resp.JSON201

		switch result.Result {
		case api.LoginOk:
			if result.Token == nil || *result.Token == "" {
				return "", fmt.Errorf("login: API returned no token")
			}
			return *result.Token, nil

		case api.TwofactorOptions:
			if result.Options == nil || !slices.Contains(*result.Options, api.Sms) {
				return "", fmt.Errorf("login: none of the account's two-factor methods are supported by the CLI yet (only SMS is)")
			}
			method := api.Sms
			body.TwoFactorMethod = &method

		case api.TwofactorRequired:
			if result.Type == nil || *result.Type != api.Sms || result.Verification == nil {
				return "", fmt.Errorf("login: the account requires a two-factor method not supported by the CLI yet (only SMS is)")
			}
			return loginWithSmsCode(ctx, prompt, client, username, *result.Verification)

		default:
			return "", fmt.Errorf("login: unexpected result %q", result.Result)
		}
	}
	return "", fmt.Errorf("login: two-factor negotiation did not converge")
}

// loginWithSmsCode triggers an SMS code and exchanges it for a token.
func loginWithSmsCode(ctx context.Context, prompt *prompter, client *cleura.Client, username, verification string) (string, error) {
	smsResp, err := client.AuthRequestSmsCodeWithResponse(ctx, api.AuthRequestSmsCodeJSONRequestBody{
		Login:        username,
		Verification: verification,
	})
	if err != nil {
		return "", fmt.Errorf("requesting SMS code: %w", err)
	}
	if smsResp.StatusCode() != 204 {
		return "", apiError("login: requesting SMS code", smsResp.HTTPResponse, smsResp.Body)
	}

	codeStr, err := prompt.line(ctx, "SMS code")
	if err != nil {
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("%w; accounts with SMS two-factor authentication must log in from an interactive terminal", err)
		}
		return "", err
	}
	code, err := strconv.Atoi(strings.TrimSpace(codeStr))
	if err != nil {
		return "", fmt.Errorf("SMS code must be numeric")
	}

	resp, err := client.AuthCreateTokenBySmsCodeWithResponse(ctx, api.AuthCreateTokenBySmsCodeJSONRequestBody{
		Login:        username,
		Verification: verification,
		Code:         code,
	})
	if err != nil {
		return "", fmt.Errorf("verifying SMS code: %w", err)
	}
	if resp.JSON200 == nil {
		return "", apiError("login: verifying SMS code", resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200.Token, nil
}

// loginWithProvidedToken reads a pre-created API token and validates it
// against the API before it is stored.
func loginWithProvidedToken(ctx context.Context, opts *globalOptions, prompt *prompter, url, username string) (string, error) {
	token, err := prompt.secret(ctx, "Token")
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("token must not be empty")
	}

	client, err := cleura.NewClientWithCredentials(url, username, token, opts.clientOptions()...)
	if err != nil {
		return "", err
	}
	resp, err := client.IdentityGetCurrentUserWithResponse(ctx)
	if err != nil {
		return "", fmt.Errorf("validating token: %w", err)
	}
	if resp.JSON200 == nil {
		return "", apiError("validating token", resp.HTTPResponse, resp.Body)
	}
	return token, nil
}
