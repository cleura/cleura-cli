package cli

import (
	"fmt"
	"io"

	"github.com/cleura/cleura-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWhoamiCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "whoami",
		Short:   "Show the currently authenticated user",
		Example: "  cleura whoami\n  cleura whoami -o json",
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

			resp, err := client.IdentityGetCurrentUserWithResponse(cmd.Context())
			if err != nil {
				return fmt.Errorf("fetching current user: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("whoami", settings, resp.HTTPResponse, resp.Body)
			}
			user := resp.JSON200

			return output.Render(cmd.OutOrStdout(), opts.output, user, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("ID", user.Id)
				kv.Row("Username", user.Name)
				kv.Row("Name", displayName(user.Firstname, user.Lastname))
				kv.Row("Email", user.Email)
				kv.Row("Admin", user.Admin)
				kv.Row("Currency", user.Currency.Code)
				kv.Row("Profile", settings.ProfileName)
				return kv.Flush()
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}
