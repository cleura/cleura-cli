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

// projectScopedHelp states the region/project requirement. It is appended to
// the help of every gardener command so the prerequisite is stated wherever a
// user looks (the parent group and each leaf), not only on 'shoot list'.
const projectScopedHelp = `A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.`

func newGardenerCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gardener",
		Short: "Manage Gardener Kubernetes clusters",
		Long:  "Manage Gardener Kubernetes clusters.\n\n" + projectScopedHelp,
		// A typo'd subcommand must fail loudly (exit 1 + suggestions), not
		// print help to stdout with exit 0 — CI pipelines key off exit codes.
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}

	shoot := &cobra.Command{
		Use:   "shoot",
		Short: "Manage shoot clusters",
		Long:  "Manage Gardener shoot (Kubernetes) clusters.\n\n" + projectScopedHelp,
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	// Region/project are needed by every gardener call; persistent here so
	// all shoot subcommands inherit them (and they no longer clutter
	// unrelated commands as global flags).
	addProjectContextFlags(cmd, opts, true)

	shoot.AddCommand(
		newShootListCommand(opts),
		newShootKubeconfigCommand(opts),
		newShootActionCommand(opts, shootAction{
			use:       "wake <shoot-name>",
			short:     "Wake a shoot cluster up from hibernation",
			long:      "Wake a shoot cluster up from hibernation.\n\n" + projectScopedHelp,
			op:        "waking shoot",
			confirmed: "Requested wake-up of shoot %q; watch progress with 'cleura gardener shoot list'",
			example:   "  cleura gardener shoot wake prod",
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
			long:      "Hibernate a shoot cluster: scale its workloads and control plane down to save cost. Reversible with 'cleura gardener shoot wake'.\n\n" + projectScopedHelp,
			op:        "hibernating shoot",
			confirmed: "Requested hibernation of shoot %q; watch progress with 'cleura gardener shoot list'",
			example:   "  cleura gardener shoot hibernate staging   # reversible: wake it with 'shoot wake'",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerHibernateShootWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootActionCommand(opts, shootAction{
			use:   "reconcile <shoot-name>",
			short: "Trigger a reconciliation of a shoot cluster",
			long: "Ask Gardener to run its reconciliation loop for this shoot now, instead of\n" +
				"waiting for the periodic cycle — it applies pending changes and can recover a\n" +
				"cluster stuck after a transient error.\n\n" + projectScopedHelp,
			op:        "reconciling shoot",
			confirmed: "Requested reconcile of shoot %q; watch progress with 'cleura gardener shoot list'",
			example:   "  cleura gardener shoot reconcile prod",
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
	// Auth is the most fundamental prerequisite: check it before region/project
	// so a not-logged-in user is told to log in, not to set a region first.
	client, err := opts.authenticatedClient(settings)
	if err != nil {
		return settings, nil, err
	}
	if err := requireProjectContext(settings); err != nil {
		return settings, nil, err
	}
	return settings, client, nil
}

func newShootListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List shoot clusters in a project",
		Long:    "List shoot clusters in a project.\n\n" + projectScopedHelp,
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

			// Build the view model BEFORE rendering so -o json/yaml carry the
			// CLI-computed columns too (status summary, upgrade availability),
			// not just the raw API shape. The full API shoot is embedded, so
			// no raw field is lost.
			profileVersions := map[string][]api.GardenerCloudProfileKubernetesVersion{}
			profilesFetched := false
			if len(shoots) > 0 {
				if profResp, err := client.GardenerListCloudProfilesWithResponse(cmd.Context(), settings.Cloud); err == nil && profResp.JSON200 != nil {
					profilesFetched = true
					for _, p := range *profResp.JSON200 {
						profileVersions[p.Name] = p.Kubernetes.Versions
					}
				} else {
					opts.warnf(cmd, "could not fetch cloud profiles; UPGRADE column shows \"?\"")
				}
			}

			views := make([]shootView, 0, len(shoots))
			for _, s := range shoots {
				upgradeDisplay, upgradeVersion := "?", ""
				if versions, ok := profileVersions[s.CloudProfileName]; ok {
					v := upgradeAvailable(s.Kubernetes.Version, versions) // "-" (current) or a version
					upgradeDisplay = v
					if v != "-" {
						upgradeVersion = v
					}
				} else if profilesFetched {
					opts.warnf(cmd, "cloud profile %q for shoot %q was not found; UPGRADE unknown", s.CloudProfileName, s.Name)
				}
				views = append(views, shootView{
					GardenerShootShoot: s,
					StatusSummary:      shootStatusSummary(s),
					UpgradeAvailable:   upgradeVersion,
					upgradeDisplay:     upgradeDisplay,
				})
			}

			header := []string{"NAME", "REGION", "K8S", "UPGRADE", "POOLS", "STATUS"}
			return output.Render(cmd.OutOrStdout(), opts.output, views, func(w io.Writer) error {
				if len(views) == 0 {
					// Header to stdout, notice to stderr: piped consumers see an
					// empty table, humans see why.
					opts.infof(cmd, "No shoot clusters in project %s (region %s)", settings.ProjectID, settings.Region)
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(views))
				for _, v := range views {
					rows = append(rows, []string{
						v.Name,
						v.Region,
						v.Kubernetes.Version,
						v.upgradeDisplay,
						strconv.Itoa(len(v.ShootProvider.Workers)),
						v.StatusSummary,
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newShootKubeconfigCommand(opts *globalOptions) *cobra.Command {
	var expiration time.Duration
	var file string

	cmd := &cobra.Command{
		Use:   "kubeconfig <shoot-name>",
		Short: "Create a time-limited admin kubeconfig for a shoot cluster",
		Long: "Create an admin kubeconfig for a shoot cluster and print it to stdout,\n" +
			"or write it to a file with --file. The credential expires after --expiration\n" +
			"(the API may cap the allowed validity).\n\n" + projectScopedHelp,
		Example: `  cleura gardener shoot kubeconfig prod > prod.kubeconfig
  cleura gardener shoot kubeconfig prod --expiration 8h -f ~/.kube/prod.yaml
  KUBECONFIG=$(pwd)/prod.kubeconfig kubectl get nodes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// Validate the integer seconds actually sent: a sub-second
			// duration is >0 but truncates to 0, which would slip past a
			// naive `expiration <= 0` check.
			expirationSeconds := int(expiration.Seconds())
			if expirationSeconds < 1 {
				return fmt.Errorf("--expiration must be at least 1s, got %s", expiration)
			}

			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerCreateShootAdminKubeConfigWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name,
				api.GardenerCreateShootAdminKubeConfigJSONRequestBody{
					ExpirationSeconds: expirationSeconds,
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
			opts.infof(cmd, "Wrote admin kubeconfig for shoot %q to %s (requested validity %s; the API may cap it)", name, file, expiration)
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
	use, short, long, op, confirmed, example string
	call                                     func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error)
}

func newShootActionCommand(opts *globalOptions, action shootAction) *cobra.Command {
	return &cobra.Command{
		Use:     action.use,
		Short:   action.short,
		Long:    action.long,
		Example: action.example,
		Args:    cobra.ExactArgs(1),
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

// shootView is a shoot plus the CLI-computed columns. The full API shoot is
// embedded so -o json/yaml keep every raw field; the computed fields are
// added alongside. upgradeDisplay is table-only (unexported → not marshaled).
type shootView struct {
	api.GardenerShootShoot
	StatusSummary    string `json:"status_summary" yaml:"status_summary"`
	UpgradeAvailable string `json:"upgrade_available,omitempty" yaml:"upgrade_available,omitempty"`
	upgradeDisplay   string
}

// shootStatusSummary is the human status word for a shoot: an in-flight
// operation (with progress) beats the steady-state hibernation flag, which
// stays true for the whole duration of a wake-up; a completed, non-hibernated
// cluster reads as "ready" rather than the raw Gardener "Succeeded".
func shootStatusSummary(s api.GardenerShootShoot) string {
	if op := s.LastOperation; op != nil && op.Progress < 100 {
		return fmt.Sprintf("%s (%s %d%%)", op.State, op.Type, op.Progress)
	}
	if s.Status != nil && s.Status.IsHibernated {
		return "hibernated"
	}
	if op := s.LastOperation; op != nil {
		if string(op.State) == "Succeeded" {
			return "ready"
		}
		return string(op.State)
	}
	return ""
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
