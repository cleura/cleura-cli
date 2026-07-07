package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func newLogoutCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke the profile's stored API token and remove it from the configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, settings, err := opts.settings()
			if err != nil {
				return err
			}

			profile := cfg.Profiles[settings.ProfileName]
			if profile == nil || profile.Token == "" {
				opts.infof(cmd, "No stored token for profile %q", settings.ProfileName)
				return nil
			}

			// Revocation deliberately uses the profile's own credentials, not
			// the resolved settings: an exported CLEURA_API_TOKEN would shadow
			// the stored token and we would revoke the wrong one.
			if env := os.Getenv("CLEURA_API_TOKEN"); env != "" && env != profile.Token {
				opts.warnf(cmd, "CLEURA_API_TOKEN is set in your environment; logout revokes the profile's stored token, the environment token is not touched")
			}
			if err := revokeProfileToken(cmd.Context(), opts, profile); err != nil {
				opts.warnf(cmd, "could not revoke token (%v); removing it locally anyway", err)
			}

			profile.Token = ""
			if err := cfg.Save(); err != nil {
				return err
			}
			opts.infof(cmd, "Logged out (profile %q)", settings.ProfileName)
			return nil
		},
	}
}

// revokeProfileToken revokes the token stored in a profile, authenticating
// with the profile's own credentials against the profile's own endpoint.
func revokeProfileToken(ctx context.Context, opts *globalOptions, p *config.Profile) error {
	url, err := profileEndpoint(p)
	if err != nil {
		return err
	}
	client, err := cleura.NewClientWithCredentials(url, p.Username, p.Token, opts.clientOptions()...)
	if err != nil {
		return err
	}
	resp, err := client.AuthRevokeTokenWithResponse(ctx)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 204 {
		return fmt.Errorf("unexpected response %s", resp.Status())
	}
	return nil
}

// profileEndpoint returns the API base URL a profile's stored credentials
// belong to.
func profileEndpoint(p *config.Profile) (string, error) {
	if p.APIURL != "" {
		return p.APIURL, nil
	}
	cloud := p.Cloud
	if cloud == "" {
		cloud = "public"
	}
	return cleura.DefaultAPIURL(cloud)
}
