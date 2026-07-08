package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cleura/cleura-cli/internal/config"
	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/cleura/cleura-client-go/cleura"
	"github.com/spf13/cobra"
)

func newGardenerCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gardener",
		Short: "Manage Gardener Kubernetes clusters",
		// A typo'd subcommand must fail loudly (exit 1 + suggestions), not
		// print help to stdout with exit 0 — CI pipelines key off exit codes.
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}

	shoot := &cobra.Command{
		Use:   "shoot",
		Short: "Manage shoot clusters",
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	shoot.AddCommand(
		newShootListCommand(opts),
		newShootKubeconfigCommand(opts),
		newShootActionCommand(opts, shootAction{
			use:       "wake <shoot-name>",
			short:     "Wake a shoot cluster up from hibernation",
			op:        "waking shoot",
			confirmed: "Requested wake-up of shoot %q; watch progress with 'cleura gardener shoot list'",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerWakeUpShootWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootActionCommand(opts, shootAction{
			use:       "hibernate <shoot-name>",
			short:     "Hibernate a shoot cluster (scales workloads and control plane down)",
			op:        "hibernating shoot",
			confirmed: "Requested hibernation of shoot %q; watch progress with 'cleura gardener shoot list'",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerHibernateShootWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootActionCommand(opts, shootAction{
			use:       "reconcile <shoot-name>",
			short:     "Trigger a reconciliation of a shoot cluster",
			op:        "reconciling shoot",
			confirmed: "Requested reconcile of shoot %q; watch progress with 'cleura gardener shoot list'",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerReconcileShootWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
	)

	cmd.AddCommand(shoot)
	return cmd
}

// gardenerContext resolves settings and returns an authenticated client after
// validating the project-scoped requirements shared by all gardener commands.
func gardenerContext(opts *globalOptions) (config.Settings, *cleura.Client, error) {
	_, settings, err := opts.settings()
	if err != nil {
		return settings, nil, err
	}
	if err := requireProjectContext(settings); err != nil {
		return settings, nil, err
	}
	client, err := opts.authenticatedClient(settings)
	return settings, client, err
}

func newShootListCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List shoot clusters in a project",
		Long: `List shoot clusters in a project. A region and project must be selected:
via --region/--project-id, CLEURA_REGION/CLEURA_PROJECT_ID, or stored in the
profile at login.`,
		Example: "  cleura gardener shoot list\n  cleura gardener shoot list --region sto1 --project-id a1b2c3 -o json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerListShootsWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID)
			if err != nil {
				return fmt.Errorf("listing shoots: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing shoots", settings, resp.HTTPResponse, resp.Body)
			}
			shoots := *resp.JSON200

			header := []string{"NAME", "REGION", "K8S", "UPGRADE", "WORKERS", "STATUS"}
			return output.Render(cmd.OutOrStdout(), opts.output, shoots, func(w io.Writer) error {
				if len(shoots) == 0 {
					// Header to stdout, notice to stderr: piped consumers see an
					// empty table, humans see why.
					opts.infof(cmd, "No shoot clusters in project %s (region %s)", settings.ProjectID, settings.Region)
					return output.Table(w, header, nil)
				}

				// The UPGRADE column needs the cloud profiles' available
				// Kubernetes versions. Degrade gracefully if they cannot be
				// fetched — the listing is more important than the column.
				profileVersions := map[string][]api.GardenerCloudProfileKubernetesVersion{}
				if profResp, err := client.GardenerListCloudProfilesWithResponse(cmd.Context(), settings.Cloud); err == nil && profResp.JSON200 != nil {
					for _, p := range *profResp.JSON200 {
						profileVersions[p.Name] = p.Kubernetes.Versions
					}
				} else {
					opts.warnf(cmd, "could not fetch cloud profiles; UPGRADE column unavailable")
				}

				rows := make([][]string, 0, len(shoots))
				for _, s := range shoots {
					// An operation under way (or one that stopped short) beats the
					// steady-state hibernation flag: is_hibernated stays true for
					// the whole duration of a wake-up, and would mask its progress.
					status := ""
					if op := s.LastOperation; op != nil && op.Progress < 100 {
						status = fmt.Sprintf("%s (%s %d%%)", op.State, op.Type, op.Progress)
					} else if s.Status != nil && s.Status.IsHibernated {
						status = "hibernated"
					} else if op := s.LastOperation; op != nil {
						status = string(op.State)
					}

					upgrade := "?"
					if versions, ok := profileVersions[s.CloudProfileName]; ok {
						upgrade = upgradeAvailable(s.Kubernetes.Version, versions)
					}

					rows = append(rows, []string{
						s.Name,
						s.Region,
						s.Kubernetes.Version,
						upgrade,
						strconv.Itoa(len(s.ShootProvider.Workers)),
						status,
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
}

func newShootKubeconfigCommand(opts *globalOptions) *cobra.Command {
	var expiration time.Duration
	var file string

	cmd := &cobra.Command{
		Use:   "kubeconfig <shoot-name>",
		Short: "Create a time-limited admin kubeconfig for a shoot cluster",
		Long: `Create an admin kubeconfig for a shoot cluster and print it to stdout,
or write it to a file with --file. The credential expires after --expiration
(the API may cap the allowed validity).`,
		Example: `  cleura gardener shoot kubeconfig prod > prod.kubeconfig
  cleura gardener shoot kubeconfig prod --expiration 8h -f ~/.kube/prod.yaml
  KUBECONFIG=$(pwd)/prod.kubeconfig kubectl get nodes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if expiration <= 0 {
				return fmt.Errorf("--expiration must be positive, got %s", expiration)
			}

			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerCreateShootAdminKubeConfigWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name,
				api.GardenerCreateShootAdminKubeConfigJSONRequestBody{
					ExpirationSeconds: int(expiration.Seconds()),
				})
			if err != nil {
				return fmt.Errorf("creating kubeconfig: %w", err)
			}
			// The generated YAML201 field is never populated for this endpoint:
			// the server's Content-Type does not match the spec's text/yaml, and
			// a kubeconfig (a YAML mapping) cannot unmarshal into the declared
			// string schema anyway. Read the raw body, like the terraform
			// provider does.
			if resp.StatusCode() != http.StatusCreated {
				return apiAuthError("creating kubeconfig", settings, resp.HTTPResponse, resp.Body)
			}
			if len(resp.Body) == 0 {
				return fmt.Errorf("creating kubeconfig: API returned an empty body")
			}
			kubeconfig := string(resp.Body)

			if file == "" {
				fmt.Fprint(cmd.OutOrStdout(), kubeconfig)
				return nil
			}
			if err := writeSecretFile(file, []byte(kubeconfig)); err != nil {
				return err
			}
			opts.infof(cmd, "Wrote admin kubeconfig for shoot %q to %s (valid %s)", name, file, expiration)
			return nil
		},
	}

	cmd.Flags().DurationVar(&expiration, "expiration", time.Hour, "How long the kubeconfig stays valid (e.g. 30m, 6h)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Write the kubeconfig to this path instead of stdout (created with mode 0600)")

	return cmd
}

// shootAction describes a shoot operation that takes no request body and
// reports success via the HTTP status.
type shootAction struct {
	use, short, op, confirmed string
	call                      func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error)
}

func newShootActionCommand(opts *globalOptions, action shootAction) *cobra.Command {
	return &cobra.Command{
		Use:   action.use,
		Short: action.short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, body, err := action.call(cmd.Context(), client, settings, name)
			if err != nil {
				return fmt.Errorf("%s: %w", action.op, err)
			}
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return apiAuthError(action.op, settings, resp, body)
			}
			opts.infof(cmd, action.confirmed, name)
			return nil
		},
	}
}

// upgradeAvailable returns the newest generally-available Kubernetes version
// that is newer than current: the version string, or "-" when current is the
// newest. Preview and deprecated versions are not offered as targets.
func upgradeAvailable(current string, versions []api.GardenerCloudProfileKubernetesVersion) string {
	best := ""
	for _, v := range versions {
		if v.Classification != nil && *v.Classification != api.Supported {
			continue
		}
		if versionLess(current, v.Version) && (best == "" || versionLess(best, v.Version)) {
			best = v.Version
		}
	}
	if best == "" {
		return "-"
	}
	return best
}

// versionLess compares two dotted numeric versions (e.g. "1.35.6"). Segments
// that are not numeric fall back to string comparison.
func versionLess(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, aerr := strconv.Atoi(as[i])
		bi, berr := strconv.Atoi(bs[i])
		if aerr != nil || berr != nil {
			if as[i] != bs[i] {
				return as[i] < bs[i]
			}
			continue
		}
		if ai != bi {
			return ai < bi
		}
	}
	return len(as) < len(bs)
}

// writeSecretFile writes credential material to path with 0600 permissions,
// atomically replacing any existing file. os.WriteFile alone would keep a
// pre-existing file's permissions — possibly world-readable — while writing
// new secrets into it.
func writeSecretFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cleura-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// requireProjectContext validates the settings needed by project-scoped commands.
func requireProjectContext(s config.Settings) error {
	if s.Region == "" {
		return fmt.Errorf("no region selected; use --region or CLEURA_REGION, or log in with --region to store it in the profile")
	}
	if s.ProjectID == "" {
		return fmt.Errorf("no project selected; use --project-id or CLEURA_PROJECT_ID, or log in with --project-id to store it in the profile")
	}
	return nil
}
