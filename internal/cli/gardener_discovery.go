package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/cleura/cleura-cli/internal/output"
	api "github.com/cleura/cleura-client-go/api"
	"github.com/spf13/cobra"
)

// cloudOnlyHelp states that a command needs only a cloud, not a region/project.
// Cloud profiles are a cloud-wide catalog; 'project bootstrap' is the
// project-scoped discovery/enablement command and uses projectScopedHelp.
const cloudOnlyHelp = `This is cloud-wide: it needs a cloud (--cloud, or the profile's cloud), but no
region or project.`

func newCloudProfileCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud-profile",
		Short: "Inspect Gardener cloud profiles (versions, machine types, images)",
		Long: "Inspect the Gardener cloud profiles available in a cloud: the Kubernetes\n" +
			"versions, machine types, machine images, regions/zones and volume types you\n" +
			"can use when creating shoots.\n\n" + cloudOnlyHelp,
		Args: cobra.NoArgs,
		RunE: groupHelp,
	}
	cmd.AddCommand(newCloudProfileListCommand(opts), newCloudProfileShowCommand(opts))
	return cmd
}

func newCloudProfileListCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the cloud profiles in the cloud",
		Long: "List the Gardener cloud profiles in the selected cloud, with a count of the\n" +
			"supported Kubernetes versions, usable machine types and regions each offers.\n\n" + cloudOnlyHelp,
		Example: "  cleura gardener cloud-profile list\n  cleura gardener cloud-profile list -o json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rejectProjectContextFlags(cmd); err != nil {
				return err
			}
			settings, client, err := gardenerCloudContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerListCloudProfilesWithResponse(cmd.Context(), settings.Cloud)
			if err != nil {
				return fmt.Errorf("listing cloud profiles: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("listing cloud profiles", settings, resp.HTTPResponse, resp.Body)
			}

			profiles := *resp.JSON200
			header := []string{"NAME", "K8S VERSIONS", "MACHINE TYPES", "REGIONS"}
			return output.Render(cmd.OutOrStdout(), opts.output, profiles, func(w io.Writer) error {
				if len(profiles) == 0 {
					opts.infof(cmd, "No cloud profiles in cloud %q", settings.Cloud)
					return output.Table(w, header, nil)
				}
				rows := make([][]string, 0, len(profiles))
				for _, p := range profiles {
					rows = append(rows, []string{
						p.Name,
						strconv.Itoa(len(supportedK8sVersions(p.Kubernetes.Versions))),
						strconv.Itoa(usableMachineTypeCount(p.MachineTypes)),
						strconv.Itoa(len(p.Regions)),
					})
				}
				return output.Table(w, header, rows)
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

func newCloudProfileShowCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <profile-name>",
		Short: "Show a cloud profile in detail",
		Long: "Show one Gardener cloud profile in detail: its Kubernetes versions (with any\n" +
			"non-supported ones flagged), machine types, machine images, regions/zones and\n" +
			"volume types. Find the profile names with 'cleura gardener cloud-profile list'.\n\n" + cloudOnlyHelp,
		Example: "  cleura gardener cloud-profile list\n  cleura gardener cloud-profile show <name> -o json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := rejectProjectContextFlags(cmd); err != nil {
				return err
			}
			settings, client, err := gardenerCloudContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerListCloudProfilesWithResponse(cmd.Context(), settings.Cloud)
			if err != nil {
				return fmt.Errorf("getting cloud profile: %w", err)
			}
			if resp.JSON200 == nil {
				return apiAuthError("getting cloud profile", settings, resp.HTTPResponse, resp.Body)
			}

			// The API only lists cloud profiles; select the requested one
			// client-side and report the available names when it is missing.
			profiles := *resp.JSON200
			var profile *api.GardenerCloudProfileCloudProfile
			names := make([]string, 0, len(profiles))
			for i := range profiles {
				names = append(names, profiles[i].Name)
				if profiles[i].Name == name {
					profile = &profiles[i]
				}
			}
			if profile == nil {
				return fmt.Errorf("cloud profile %q does not exist (available: %s)", name, strings.Join(names, ", "))
			}

			return output.Render(cmd.OutOrStdout(), opts.output, profile, func(w io.Writer) error {
				kv := output.NewKVWriter(w)
				kv.Row("Name", profile.Name)
				kv.Row("Type", profile.Type)
				kv.Row("Kubernetes versions", k8sVersionList(profile.Kubernetes.Versions))
				if len(profile.MachineTypes) > 0 {
					kv.Row("Machine types", "")
					for _, mt := range profile.MachineTypes {
						kv.Row("  "+mt.Name, machineTypeLabel(mt))
					}
				}
				if images := machineImageList(profile.MachineImages); images != "" {
					kv.Row("Machine images", images)
				}
				if regions := regionList(profile.Regions); regions != "" {
					kv.Row("Regions", regions)
				}
				if volumes := volumeTypeList(profile.VolumeTypes); volumes != "" {
					kv.Row("Volume types", volumes)
				}
				return kv.Flush()
			})
		},
	}
	addOutputFlag(cmd, opts)
	return cmd
}

// supportedK8sVersions returns the versions safe to create with: a nil
// classification or the explicit "supported" one (preview/deprecated excluded).
func supportedK8sVersions(versions []api.GardenerCloudProfileKubernetesVersion) []string {
	out := make([]string, 0, len(versions))
	for _, v := range versions {
		if v.Classification == nil || *v.Classification == api.Supported {
			out = append(out, v.Version)
		}
	}
	return out
}

func usableMachineTypeCount(types []api.GardenerCloudProfileMachineType) int {
	n := 0
	for _, mt := range types {
		if mt.Usable {
			n++
		}
	}
	return n
}

// k8sVersionList lists every version, flagging any that is not "supported"
// (e.g. preview or deprecated) so it is clear which are safe create targets.
func k8sVersionList(versions []api.GardenerCloudProfileKubernetesVersion) string {
	out := make([]string, 0, len(versions))
	for _, v := range versions {
		s := v.Version
		if v.Classification != nil && *v.Classification != api.Supported {
			s += " (" + string(*v.Classification) + ")"
		}
		out = append(out, s)
	}
	return strings.Join(out, ", ")
}

func machineTypeLabel(mt api.GardenerCloudProfileMachineType) string {
	label := fmt.Sprintf("%d vCPU, %s RAM", mt.Cpu, mt.Memory)
	if mt.Gpu > 0 {
		label += fmt.Sprintf(", %d GPU", mt.Gpu)
	}
	if mt.Architecture != "" {
		label += ", " + mt.Architecture
	}
	if !mt.Usable {
		label += " (unusable)"
	}
	return label
}

func machineImageList(images []api.GardenerCloudProfileMachineImage) string {
	out := make([]string, 0, len(images))
	for _, img := range images {
		versions := make([]string, 0, len(img.Versions))
		for _, v := range img.Versions {
			versions = append(versions, v.Version)
		}
		out = append(out, fmt.Sprintf("%s (%s)", img.Name, strings.Join(versions, ", ")))
	}
	return strings.Join(out, "; ")
}

func regionList(regions []api.GardenerCloudProfileRegion) string {
	out := make([]string, 0, len(regions))
	for _, r := range regions {
		zones := make([]string, 0, len(r.Zones))
		for _, z := range r.Zones {
			zones = append(zones, z.Name)
		}
		out = append(out, fmt.Sprintf("%s (%s)", r.Name, strings.Join(zones, ", ")))
	}
	return strings.Join(out, "; ")
}

func volumeTypeList(types []api.GardenerCloudProfileVolumeType) string {
	out := make([]string, 0, len(types))
	for _, vt := range types {
		label := vt.Name
		if vt.Class != "" {
			label += " (" + vt.Class + ")"
		}
		out = append(out, label)
	}
	return strings.Join(out, ", ")
}

func newGardenerProjectCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage Gardener enablement for a project",
		Long:  "Manage Gardener for the selected OpenStack project.\n\n" + projectScopedHelp,
		Args:  cobra.NoArgs,
		RunE:  groupHelp,
	}
	cmd.AddCommand(newProjectBootstrapCommand(opts))
	return cmd
}

func newProjectBootstrapCommand(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Enable Gardener for the selected project",
		Long: "Enable (bootstrap) Gardener for the selected OpenStack project — a one-time\n" +
			"setup required before shoots can be created in it.\n\n" + projectScopedHelp,
		Example: "  cleura gardener project bootstrap --region sto1 --project-id a1b2c3",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, client, err := gardenerContext(opts)
			if err != nil {
				return err
			}

			resp, err := client.GardenerCommunicationBootstrapWithResponse(cmd.Context(),
				settings.Cloud, settings.Region, settings.ProjectID)
			if err != nil {
				return fmt.Errorf("bootstrapping project: %w", err)
			}
			// Status-only (204 on success). A 404 usually means the wrong
			// region/project or that Gardener is not offered there, so the op
			// string carries both to make the message actionable.
			if resp.StatusCode() < 200 || resp.StatusCode() > 299 {
				return apiAuthError(fmt.Sprintf("bootstrapping project %s in region %s", settings.ProjectID, settings.Region),
					settings, resp.HTTPResponse, resp.Body)
			}
			opts.infof(cmd, "Requested Gardener bootstrap for project %s (region %s)", settings.ProjectID, settings.Region)
			return nil
		},
	}
	return cmd
}
