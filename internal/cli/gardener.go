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
		newShootGetCommand(opts),
		newShootCheckNameCommand(opts),
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
		newShootActionCommand(opts, shootAction{
			use:   "maintain <shoot-name>",
			short: "Run a shoot's maintenance operation now",
			long: "Run the shoot's maintenance operation immediately instead of waiting for its\n" +
				"scheduled window. It applies pending Kubernetes/OS updates and may roll nodes.\n\n" + projectScopedHelp,
			op:        "triggering maintenance",
			confirmed: "Requested maintenance of shoot %q; watch progress with 'cleura gardener shoot list'",
			example:   "  cleura gardener shoot maintain prod",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerTriggerShootMaintenanceWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootActionCommand(opts, shootAction{
			use:   "retry <shoot-name>",
			short: "Retry a shoot's last failed operation",
			long: "Retry the shoot's last failed operation — for example after a transient\n" +
				"infrastructure error — instead of waiting for the next reconciliation.\n\n" + projectScopedHelp,
			op:        "retrying shoot operation",
			confirmed: "Requested retry of shoot %q; watch progress with 'cleura gardener shoot list'",
			example:   "  cleura gardener shoot retry prod",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerRetryFailedShootOperationWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootActionCommand(opts, shootAction{
			use:   "enable-ha <shoot-name>",
			short: "Enable a highly available (multi-zone) control plane",
			long: "Enable a highly available control plane for a shoot, spreading it across\n" +
				"failure domains.\n\nThis cannot be undone through the CLI and increases cost.\n\n" + projectScopedHelp,
			op:            "enabling highly available control plane",
			confirmed:     "Requested HA control plane for shoot %q; watch progress with 'cleura gardener shoot list'",
			destructive:   true,
			confirmPrompt: "Enable an HA control plane for shoot %q? This cannot be undone via the CLI and increases cost",
			example:       "  cleura gardener shoot enable-ha prod --yes",
			call: func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error) {
				resp, err := client.GardenerEnableHighlyAvailableControlPlaneWithResponse(ctx, s.Cloud, s.Region, s.ProjectID, name)
				if err != nil {
					return nil, nil, err
				}
				return resp.HTTPResponse, resp.Body, nil
			},
		}),
		newShootCACommand(opts),
		newShootMonitoringCommand(opts),
		newShootSSHKeyCommand(opts),
		newWorkerGroupCommand(opts),
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

// gardenerCloudContext resolves settings and an authenticated client for the
// cloud-only gardener endpoints (e.g. shoot check-name) that take the Gardener
// region tag — the cloud name — but no OpenStack region/project. It
// deliberately skips requireProjectContext. Those commands inherit
// --region/--project-id from the gardener parent but do not use them, so they
// reject the flags if explicitly passed (see rejectProjectContextFlags) rather
// than silently ignoring them.
func gardenerCloudContext(opts *globalOptions) (config.Settings, *cleura.Client, error) {
	_, settings, err := opts.settings()
	if err != nil {
		return settings, nil, err
	}
	client, err := opts.authenticatedClient(settings)
	if err != nil {
		return settings, nil, err
	}
	return settings, client, nil
}

// rejectProjectContextFlags errors if --region/--project-id were explicitly
// passed to a cloud-only command. The flags are inherited (persistent on the
// gardener parent) but do not apply here; rejecting them beats silently
// ignoring an inherited flag, which the CLI treats as a bug elsewhere.
func rejectProjectContextFlags(cmd *cobra.Command) error {
	for _, f := range []string{"region", "project-id"} {
		if cmd.Flags().Changed(f) {
			return fmt.Errorf("--%s does not apply to this command (shoot names are checked cloud-wide)", f)
		}
	}
	return nil
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

			header := []string{"NAME", "REGION", "K8S", "UPGRADE", "GROUPS", "STATUS"}
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

// shootDetailView is a shoot plus the CLI-computed fields shown by 'shoot get'.
// The full API shoot is embedded so -o json/yaml keep every raw field; the
// computed fields are added alongside (built before output.Render).
type shootDetailView struct {
	api.GardenerShootShoot
	StatusSummary    string `json:"status_summary" yaml:"status_summary"`
	HighAvailability string `json:"high_availability" yaml:"high_availability"`
	WorkerGroups     int    `json:"worker_groups" yaml:"worker_groups"`
	UpgradeAvailable string `json:"upgrade_available,omitempty" yaml:"upgrade_available,omitempty"`
}

func newShootDetailView(s api.GardenerShootShoot) shootDetailView {
	return shootDetailView{
		GardenerShootShoot: s,
		StatusSummary:      shootStatusSummary(s),
		HighAvailability:   shootHA(s),
		WorkerGroups:       len(s.ShootProvider.Workers),
	}
}

// shootHA reports the high-availability failure tolerance of a shoot's control
// plane. There is no literal HA boolean in the API: HA is enabled iff the
// (optional) ControlPlane block is present, and its tolerance is the zone/node
// scope it survives.
func shootHA(s api.GardenerShootShoot) string {
	if s.ControlPlane == nil {
		return "none"
	}
	if t := string(s.ControlPlane.HighAvailability.FailureTolerance.Type); t != "" {
		return t
	}
	return "enabled"
}

// shootUpgradeTarget reports the newest Kubernetes upgrade available for a
// shoot via its cloud profile. ok is false only when the cloud-profile fetch
// fails, so callers can degrade gracefully; a shoot already on the newest
// version returns ("-", true).
func shootUpgradeTarget(ctx context.Context, client *cleura.Client, cloud string, s api.GardenerShootShoot) (target string, ok bool) {
	profResp, err := client.GardenerListCloudProfilesWithResponse(ctx, cloud)
	if err != nil || profResp.JSON200 == nil {
		return "", false
	}
	for _, p := range *profResp.JSON200 {
		if p.Name == s.CloudProfileName {
			return upgradeAvailable(s.Kubernetes.Version, p.Kubernetes.Versions), true
		}
	}
	return "-", true
}

func hibernationScheduleText(sc api.GardenerShootHibernationSchedule) string {
	parts := make([]string, 0, 3)
	if start := strDeref(sc.Start); start != "" {
		parts = append(parts, "down "+start)
	}
	if end := strDeref(sc.End); end != "" {
		parts = append(parts, "up "+end)
	}
	if loc := strDeref(sc.Location); loc != "" {
		parts = append(parts, "("+loc+")")
	}
	return strings.Join(parts, " ")
}

func newShootGetCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <shoot-name>",
		Short: "Show detailed information about a shoot cluster",
		Long: "Show detailed information about a single shoot cluster — worker groups,\n" +
			"maintenance window, hibernation schedule, networking, high availability and\n" +
			"status — that the summary 'shoot list' omits.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot get prod\n  cleura gardener shoot get prod -o json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name)
			if err != nil {
				return fmt.Errorf("getting shoot: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting shoot", settings, resp.HTTPResponse, resp.Body)
			}

			view := newShootDetailView(*resp.JSON200)
			// Best-effort upgrade hint, mirroring 'shoot list'. A profile-fetch
			// failure degrades the field silently rather than failing the get.
			if target, ok := shootUpgradeTarget(cmd.Context(), client, settings.Cloud, *resp.JSON200); ok && target != "-" && target != "" {
				view.UpgradeAvailable = target
			}

			return output.Render(cmd.OutOrStdout(), opts.output, view, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("Name", view.Name)
				kv.Row("Region", view.Region)
				if view.Purpose != "" {
					kv.Row("Purpose", view.Purpose)
				}
				kv.Row("Cloud profile", view.CloudProfileName)
				kv.Row("Kubernetes", view.Kubernetes.Version)
				if view.UpgradeAvailable != "" {
					kv.Row("Upgrade available", view.UpgradeAvailable)
				}
				kv.Row("Status", view.StatusSummary)
				kv.Row("High availability", view.HighAvailability)
				kv.Row("Networking", fmt.Sprintf("%s (nodes %s)", view.Networking.Type, view.Networking.Nodes))
				if view.Maintenance.TimeWindow.Begin != "" {
					kv.Row("Maintenance window", view.Maintenance.TimeWindow.Begin+"-"+view.Maintenance.TimeWindow.End)
				}
				if view.AllowedCidrs != nil && len(*view.AllowedCidrs) > 0 {
					kv.Row("Allowed CIDRs", strings.Join(*view.AllowedCidrs, ", "))
				}
				if view.Hibernation != nil && len(view.Hibernation.Schedules) > 0 {
					schedules := make([]string, 0, len(view.Hibernation.Schedules))
					for _, sc := range view.Hibernation.Schedules {
						schedules = append(schedules, hibernationScheduleText(sc))
					}
					kv.Row("Hibernation", strings.Join(schedules, "; "))
				}
				kv.Row("Worker groups", strconv.Itoa(view.WorkerGroups))
				for _, wg := range view.ShootProvider.Workers {
					kv.Row("  "+wg.Name, fmt.Sprintf("%s, %d-%d nodes, zones %s",
						wg.Machine.Type, wg.Minimum, wg.Maximum, strings.Join(wg.Zones, ",")))
				}
				return kv.Flush()
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

// checkNameView reports shoot-name availability with a positive framing
// (Available) while preserving the raw API field (IsTaken) via the embed.
type checkNameView struct {
	api.GardenerIsShootNameTaken
	Name      string `json:"name" yaml:"name"`
	Available bool   `json:"available" yaml:"available"`
}

func newShootCheckNameCommand(opts *globalOptions) *cobra.Command {
	var exitCode bool
	cmd := &cobra.Command{
		Use:   "check-name <shoot-name>",
		Short: "Check whether a shoot name is available in the cloud",
		Long: "Check whether a shoot name is already taken in the selected cloud.\n\n" +
			"This is a cloud-wide check: it needs a cloud (--cloud/profile) but no\n" +
			"region or project. With --exit-code the command exits 1 when the name is\n" +
			"taken, so it can be used as a script predicate.",
		Example: "  cleura gardener shoot check-name prod\n" +
			"  cleura gardener shoot check-name prod --exit-code && echo name is free",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := rejectProjectContextFlags(cmd); err != nil {
				return err
			}
			settings, client, err := gardenerCloudContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerIsShootNameTakenWithResponse(cmd.Context(), settings.Cloud, name)
			if err != nil {
				return fmt.Errorf("checking shoot name: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("checking shoot name", settings, resp.HTTPResponse, resp.Body)
			}

			view := checkNameView{
				GardenerIsShootNameTaken: *resp.JSON200,
				Name:                     name,
				Available:                !resp.JSON200.IsTaken,
			}
			if err := output.Render(cmd.OutOrStdout(), opts.output, view, func(w io.Writer) error {
				return output.Table(w, []string{"NAME", "AVAILABLE"},
					[][]string{{view.Name, yesNo(view.Available)}})
			}); err != nil {
				return err
			}
			// Opt-in scripting predicate: taken -> exit 1, with no error text
			// (a taken name is a valid answer, not a failure).
			if exitCode && view.IsTaken {
				return &ExitCodeError{Code: 1}
			}
			return nil
		},
	}
	addOutputFlag(cmd, opts)
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Exit 1 if the name is taken (for scripting)")
	return cmd
}

// workerGroupView is a worker group (node pool) plus CLI-computed columns.
type workerGroupView struct {
	api.GardenerShootWorker
	MachineType string `json:"machine_type" yaml:"machine_type"`
	Image       string `json:"image,omitempty" yaml:"image,omitempty"`
	zonesText   string
}

func newWorkerGroupView(w api.GardenerShootWorker) workerGroupView {
	image := ""
	if w.Machine.Image.Name != "" {
		image = strings.TrimSpace(w.Machine.Image.Name + " " + w.Machine.Image.Version)
	}
	return workerGroupView{
		GardenerShootWorker: w,
		MachineType:         w.Machine.Type,
		Image:               image,
		zonesText:           strings.Join(w.Zones, ","),
	}
}

func newWorkerGroupCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker-group",
		Short: "Inspect a shoot's worker groups (node pools)",
		Long:  "Inspect the worker groups (node pools) of a shoot cluster.\n\n" + projectScopedHelp,
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(newWorkerGroupListCommand(opts))
	return cmd
}

func newWorkerGroupListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <shoot-name>",
		Short: "List a shoot's worker groups",
		Long: "List the worker groups (node pools) of a shoot: machine type, node-count\n" +
			"range, rolling-update surge and zones.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot worker-group list prod\n  cleura gardener shoot worker-group list prod -o json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name)
			if err != nil {
				return fmt.Errorf("listing worker groups: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing worker groups", settings, resp.HTTPResponse, resp.Body)
			}

			workers := resp.JSON200.ShootProvider.Workers
			views := make([]workerGroupView, 0, len(workers))
			for _, w := range workers {
				views = append(views, newWorkerGroupView(w))
			}

			header := []string{"NAME", "MACHINE", "MIN", "MAX", "SURGE", "ZONES"}
			return output.Render(cmd.OutOrStdout(), opts.output, views, func(w io.Writer) error {
				if len(views) == 0 {
					opts.infof(cmd, "Shoot %q has no worker groups", name)
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(views))
				for _, v := range views {
					rows = append(rows, []string{
						v.Name,
						v.MachineType,
						strconv.Itoa(v.Minimum),
						strconv.Itoa(v.Maximum),
						strconv.Itoa(v.MaxSurge),
						v.zonesText,
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
// reports success via the HTTP status. When destructive is set, the command
// gains a --yes flag and confirms first (confirmPrompt is formatted with the
// shoot name); confirmation refuses on a non-TTY, so CI must pass --yes.
type shootAction struct {
	use, short, long, op, confirmed, example string
	destructive                              bool
	confirmPrompt                            string
	call                                     func(ctx context.Context, client *cleura.Client, s config.Settings, name string) (*http.Response, []byte, error)
}

func newShootActionCommand(opts *globalOptions, action shootAction) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
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

			if action.destructive && !yes {
				ok, err := newPrompter(cmd).confirm(cmd.Context(), fmt.Sprintf(action.confirmPrompt, name))
				if err != nil {
					return err
				}
				if !ok {
					// Declined, or non-TTY without --yes: fail closed.
					return fmt.Errorf("aborted; rerun with --yes to confirm")
				}
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
	if action.destructive {
		cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt (required on a non-interactive terminal)")
	}
	return cmd
}

func newShootCACommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ca",
		Short: "Manage a shoot's certificate-authority rotation",
		Long:  "Manage the two-phase certificate-authority (CA) rotation of a shoot cluster.\n\n" + projectScopedHelp,
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(newShootCARotateCommand(opts), newShootCAStatusCommand(opts))
	return cmd
}

// caRotateStage maps the user-facing --stage value to the API enum. Only the
// two write stages are accepted ("start" aliases "prepare"); the read-only
// states are rejected. The SDK enum's Valid() accepts all seven, so this
// deliberately does not use it.
func caRotateStage(stage string) (api.K8sCaRotationStage, bool) {
	switch strings.ToLower(stage) {
	case "prepare", "start":
		return api.Prepare, true
	case "complete":
		return api.Complete, true
	default:
		return "", false
	}
}

func newShootCARotateCommand(opts *globalOptions) *cobra.Command {
	var stage string
	var yes bool
	cmd := &cobra.Command{
		Use:   "rotate <shoot-name> --stage prepare|complete",
		Short: "Rotate a shoot's certificate authority",
		Long: "Rotate a shoot's certificate authorities in two stages:\n" +
			"  --stage prepare    start the rotation (new CAs are issued alongside the old)\n" +
			"  --stage complete   finish it (the old CAs are dropped)\n\n" +
			"A rotation cannot be aborted through the CLI once prepared, and completing it\n" +
			"invalidates kubeconfigs issued before the rotation (re-issue them with\n" +
			"'cleura gardener shoot kubeconfig'). Track progress with 'shoot ca status'.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot ca rotate prod --stage prepare\n  cleura gardener shoot ca rotate prod --stage complete --yes",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// Only the two write stages are valid input; the read-only enum
			// states (Preparing/Prepared/...) belong to 'ca status'.
			wire, ok := caRotateStage(stage)
			if !ok {
				return fmt.Errorf("unknown stage %q (expected one of: prepare, complete)", stage)
			}
			prompt := fmt.Sprintf("Start CA rotation for shoot %q? It cannot be aborted through the CLI once started", name)
			if wire == api.Complete {
				prompt = fmt.Sprintf("Complete CA rotation for shoot %q? Kubeconfigs issued before the rotation will stop working", name)
			}

			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}
			if !yes {
				ok, err := newPrompter(cmd).confirm(cmd.Context(), prompt)
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted; rerun with --yes to confirm")
				}
			}

			resp, err := client.GardenerRotateShootCaWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name,
				api.GardenerRotateShootCaJSONRequestBody{Stage: wire})
			if err != nil {
				return fmt.Errorf("rotating shoot CA: %w", err)
			}
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return apiAuthError("rotating shoot CA", settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Requested CA rotation (%s) for shoot %q; check with 'cleura gardener shoot ca status %s'",
				strings.ToLower(stage), name, name)
			return nil
		},
	}
	cmd.Flags().StringVar(&stage, "stage", "", "Rotation stage: prepare or complete")
	_ = cmd.MarkFlagRequired("stage")
	_ = cmd.RegisterFlagCompletionFunc("stage", cobra.FixedCompletions([]string{"prepare", "complete"}, cobra.ShellCompDirectiveNoFileComp))
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt (required on a non-interactive terminal)")
	return cmd
}

// caStatusView is the CA rotation stage plus a computed next-action hint. The
// raw API stage is embedded so -o json/yaml keep it.
type caStatusView struct {
	api.GardenerCaRotationStage
	NextAction string `json:"next_action" yaml:"next_action"`
}

// caNextAction maps a rotation stage to the action the operator should take.
func caNextAction(name string, stage api.K8sCaRotationStage) string {
	switch stage {
	case api.NotInitiated:
		return fmt.Sprintf("not started; run 'cleura gardener shoot ca rotate %s --stage prepare'", name)
	case api.Preparing:
		return "prepare in progress; check again shortly"
	case api.Prepared:
		return fmt.Sprintf("prepared; run 'cleura gardener shoot ca rotate %s --stage complete'", name)
	case api.Completing:
		return "completion in progress; check again shortly"
	case api.Completed:
		return "rotation complete; no action needed"
	case api.Prepare:
		return "prepare accepted; expect Preparing then Prepared"
	case api.Complete:
		return "complete accepted; expect Completing then Completed"
	default:
		return string(stage)
	}
}

func newShootCAStatusCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status <shoot-name>",
		Short:   "Show a shoot's CA rotation stage",
		Long:    "Show the current certificate-authority rotation stage of a shoot and the next action to take.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot ca status prod",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootCaRotationStageWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name)
			if err != nil {
				return fmt.Errorf("getting CA rotation stage: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting CA rotation stage", settings, resp.HTTPResponse, resp.Body)
			}

			view := caStatusView{
				GardenerCaRotationStage: *resp.JSON200,
				NextAction:              caNextAction(name, resp.JSON200.Stage),
			}
			return output.Render(cmd.OutOrStdout(), opts.output, view, func(w io.Writer) error {
				return output.Table(w, []string{"STAGE", "NEXT ACTION"},
					[][]string{{string(view.Stage), view.NextAction}})
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

// Metric units differ by endpoint (none are declared in the spec; these are
// Cleura's fixed conventions):
//   - the details endpoints (node, worker-group) report memory and filesystem
//     in GiB and utilization as a percentage;
//   - the nodes-overview endpoint reports memory in BYTES (converted to GiB for
//     the table by bytesToGiB).
//
// CPU is in cores everywhere.
const metricsUnitsHelp = "Values are in cores (CPU), GiB (memory and filesystem) and percent (utilization)."

// overviewUnitsHelp documents the nodes-overview table, which shows plain
// usage/idle values; memory arrives in bytes and is shown as GiB, while -o json
// keeps the raw API values.
const overviewUnitsHelp = "The table shows cores (CPU) and GiB (memory); -o json reports the raw API values (cores and bytes)."

// fmtFloat renders a float32 metric without trailing zeros (used for counts
// like pods). The unit, if any, is carried by the column header.
func fmtFloat(f float32) string {
	return strconv.FormatFloat(float64(f), 'f', -1, 32)
}

// fmtCores renders a CPU-cores value to three decimals.
func fmtCores(f float32) string {
	return fmt.Sprintf("%.3f", f)
}

// bytesToGiB renders a byte count as GiB to two decimals. The nodes-overview
// endpoint reports memory in bytes; the details endpoints already use GiB.
func bytesToGiB(b float32) string {
	return fmt.Sprintf("%.2f", float64(b)/(1<<30))
}

// fmtPercent renders an already-percentage value (e.g. the details endpoints'
// *Utilization fields) to one decimal with a % sign.
func fmtPercent(f float32) string {
	return fmt.Sprintf("%.1f%%", f)
}

// latestSample returns the newest sample's raw value, or "-" when empty. The
// API declares no sample ordering; we assume the last element is newest (flip
// here if that proves wrong).
func latestSample(samples []api.GardenerSample) string {
	if len(samples) == 0 {
		return "-"
	}
	return samples[len(samples)-1].Value
}

// latestWithUnit is latestSample with the unit appended (nothing appended when
// the series is empty). Percent takes no space ("4.4%"); other units do
// ("0.02 cores", "1.25 GiB").
func latestWithUnit(samples []api.GardenerSample, unit string) string {
	v := latestSample(samples)
	if v == "-" {
		return v
	}
	if unit == "%" {
		return v + "%"
	}
	return v + " " + unit
}

func newShootMonitoringCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitoring",
		Short: "Inspect a shoot's monitoring data and credentials",
		Long:  "Inspect a shoot's observability data: dashboard credentials and node /\nworker-group resource metrics.\n\n" + projectScopedHelp,
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(
		newShootMonitoringCredentialsCommand(opts),
		newShootMonitoringNodesCommand(opts),
		newShootMonitoringNodeCommand(opts),
		newShootMonitoringWorkerGroupCommand(opts),
	)
	return cmd
}

func newShootMonitoringCredentialsCommand(opts *globalOptions) *cobra.Command {
	var showSecrets bool
	cmd := &cobra.Command{
		Use:   "credentials <shoot-name>",
		Short: "Show a shoot's monitoring (Prometheus/Plutono) credentials",
		Long: "Show the Prometheus and Plutono dashboard URLs and logins for a shoot.\n\n" +
			"Passwords are masked in table output; reveal them with --show-secrets or\n-o json/yaml.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot monitoring credentials prod\n  cleura gardener shoot monitoring credentials prod --show-secrets",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootMonitoringCredentialsWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name)
			if err != nil {
				return fmt.Errorf("getting monitoring credentials: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting monitoring credentials", settings, resp.HTTPResponse, resp.Body)
			}

			creds := *resp.JSON200
			return output.Render(cmd.OutOrStdout(), opts.output, creds, func(w io.Writer) error {
				mask := func(pw string) string {
					if showSecrets {
						return pw
					}
					return "********"
				}
				return output.Table(w, []string{"SERVICE", "URL", "USERNAME", "PASSWORD"}, [][]string{
					{"prometheus", creds.Prometheus.Url, creds.Prometheus.Username, mask(creds.Prometheus.Password)},
					{"plutono", creds.Plutono.Url, creds.Plutono.Username, mask(creds.Plutono.Password)},
				})
			})
		},
	}
	addOutputFlag(cmd, opts)
	cmd.Flags().BoolVar(&showSecrets, "show-secrets", false, "Show passwords in table output instead of masking them")
	return cmd
}

func newShootMonitoringNodesCommand(opts *globalOptions) *cobra.Command {
	var namesOnly bool
	cmd := &cobra.Command{
		Use:   "nodes <shoot-name> <worker-group>",
		Short: "Show per-node resource usage for a worker group",
		Long: "Show a per-node snapshot of CPU (used/idle), memory (used/free) and pod\n" +
			"count for a worker group. With --names-only, print just the node names\n" +
			"(useful for scripts).\n\n" + overviewUnitsHelp + "\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot monitoring nodes prod default\n  cleura gardener shoot monitoring nodes prod default --names-only",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, group := args[0], args[1]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			if namesOnly {
				resp, err := client.GardenerGetShootNodeNamesWithResponse(cmd.Context(),
					settings.Cloud, settings.Region, settings.ProjectID, name, group)
				if err != nil {
					return fmt.Errorf("getting node names: %w", err)
				}
				if resp.JSON200 == nil {
					return apiAuthError("getting node names", settings, resp.HTTPResponse, resp.Body)
				}
				names := *resp.JSON200
				return output.Render(cmd.OutOrStdout(), opts.output, names, func(w io.Writer) error {
					if len(names.NodeNames) == 0 {
						opts.infof(cmd, "Worker group %q of shoot %q has no nodes", group, name)
						return output.Table(w, []string{"NODE"}, nil)
					}
					rows := make([][]string, 0, len(names.NodeNames))
					for _, n := range names.NodeNames {
						rows = append(rows, []string{n})
					}
					return output.Table(w, []string{"NODE"}, rows)
				})
			}

			resp, err := client.GardenerGetShootWorkerGroupNodesOverviewWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name, group)
			if err != nil {
				return fmt.Errorf("getting worker group nodes overview: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting worker group nodes overview", settings, resp.HTTPResponse, resp.Body)
			}
			nodes := *resp.JSON200
			header := []string{"NODE", "WORKER-GROUP", "CPU-USED (cores)", "CPU-IDLE (cores)", "MEM-USED (GiB)", "MEM-FREE (GiB)", "PODS"}
			return output.Render(cmd.OutOrStdout(), opts.output, nodes, func(w io.Writer) error {
				if len(nodes) == 0 {
					opts.infof(cmd, "Worker group %q of shoot %q has no nodes", group, name)
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(nodes))
				for _, n := range nodes {
					// Memory arrives in bytes on this endpoint; CPU in cores.
					rows = append(rows, []string{
						n.NodeName, n.WorkerGroup,
						fmtCores(n.CpuUsage), fmtCores(n.IdleCpu),
						bytesToGiB(n.MemoryUsage), bytesToGiB(n.UnusedMemory),
						fmtFloat(n.Pods),
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	cmd.Flags().BoolVar(&namesOnly, "names-only", false, "Print only the node names")
	return cmd
}

func newShootMonitoringNodeCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node <shoot-name> <node-name>",
		Short: "Show detailed metrics for one node",
		Long: "Show detailed metrics for a single node. The table shows the latest\n" +
			"snapshot; -o json/yaml carry the full time series.\n\n" + metricsUnitsHelp + "\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot monitoring node prod prod-default-abc12\n  cleura gardener shoot monitoring node prod prod-default-abc12 -o json",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, node := args[0], args[1]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootNodeDetailsWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name, node)
			if err != nil {
				return fmt.Errorf("getting node details: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting node details", settings, resp.HTTPResponse, resp.Body)
			}

			d := *resp.JSON200
			return output.Render(cmd.OutOrStdout(), opts.output, d, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("Node", d.NodeName)
				kv.Row("CPU utilization", fmtPercent(d.CpuUtilization))
				kv.Row("Memory utilization", fmtPercent(d.MemoryUtilization))
				kv.Row("CPU used (latest)", latestWithUnit(d.CpuUsage.Used, "cores"))
				kv.Row("CPU allocatable (latest)", latestWithUnit(d.CpuUsage.Allocatable, "cores"))
				kv.Row("Memory used (latest)", latestWithUnit(d.MemoryUsage.Used, "GiB"))
				kv.Row("Memory allocatable (latest)", latestWithUnit(d.MemoryUsage.Allocatable, "GiB"))
				kv.Row("Filesystem size (latest)", latestWithUnit(d.FilesystemSize.Size, "GiB"))
				kv.Row("Filesystem free (latest)", latestWithUnit(d.FilesystemSize.Free, "GiB"))
				return kv.Flush()
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newShootMonitoringWorkerGroupCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker-group <shoot-name> <worker-group>",
		Short: "Show aggregate metrics for a worker group",
		Long: "Show aggregate CPU/memory/node/pod metrics for a worker group. The table\n" +
			"shows the latest sample; -o json/yaml carry the full time series.\n\n" +
			metricsUnitsHelp + "\n\nManage worker groups with 'cleura gardener shoot worker-group'.\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot monitoring worker-group prod default",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, group := args[0], args[1]
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootWorkerGroupDetailsWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name, group)
			if err != nil {
				return fmt.Errorf("getting worker group details: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting worker group details", settings, resp.HTTPResponse, resp.Body)
			}

			d := *resp.JSON200
			return output.Render(cmd.OutOrStdout(), opts.output, d, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("Worker group", d.WorkerName)
				kv.Row("CPU usage (latest)", latestWithUnit(d.CpuUsage, "cores"))
				kv.Row("CPU utilization (latest)", latestWithUnit(d.CpuUtilization, "%"))
				kv.Row("Memory usage (latest)", latestWithUnit(d.MemoryUsage, "GiB"))
				kv.Row("Memory utilization (latest)", latestWithUnit(d.MemoryUtilization, "%"))
				kv.Row("Nodes (latest)", latestSample(d.NumberOfNodes))
				kv.Row("Pods (latest)", latestSample(d.NumberOfPods))
				return kv.Flush()
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newShootSSHKeyCommand(opts *globalOptions) *cobra.Command {
	var file string
	var stdout bool
	cmd := &cobra.Command{
		Use:   "ssh-key <shoot-name>",
		Short: "Fetch a shoot's node SSH private key",
		Long: "Fetch the SSH private key for a shoot's nodes.\n\n" +
			"This is a secret: write it to a 0600 file with --file, or print it to\nstdout with --stdout (exactly one is required).\n\n" + projectScopedHelp,
		Example: "  cleura gardener shoot ssh-key prod -f ~/.ssh/prod-nodes.pem\n  cleura gardener shoot ssh-key prod --stdout",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			// A private key must not be echoed by accident: require an explicit
			// destination, exactly one of --file or --stdout.
			if (file != "") == stdout {
				return fmt.Errorf("pass exactly one of --file or --stdout")
			}

			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerGetShootSshPrivateKeyWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID, name)
			if err != nil {
				return fmt.Errorf("getting ssh private key: %w", err)
			}
			// Raw-body endpoint (like kubeconfig) but success is 200, not 201.
			if resp.StatusCode() != http.StatusOK {
				return apiAuthError("getting ssh private key", settings, resp.HTTPResponse, resp.Body)
			}
			if len(resp.Body) == 0 {
				return fmt.Errorf("getting ssh private key: API returned an empty body")
			}

			if stdout {
				fmt.Fprint(cmd.OutOrStdout(), string(resp.Body))
				return nil
			}
			if err := writeSecretFile(file, resp.Body); err != nil {
				return err
			}
			opts.infof(cmd, "Wrote SSH private key for shoot %q to %s (mode 0600)", name, file)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Write the key to this path (created with mode 0600)")
	cmd.Flags().BoolVar(&stdout, "stdout", false, "Print the key to stdout instead of a file")
	return cmd
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
