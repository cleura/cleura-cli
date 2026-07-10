package cli

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

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
				if groups := groupVersions(classifyK8s(profile.Kubernetes.Versions)); len(groups) > 0 {
					kv.Row("Kubernetes versions", "")
					for _, g := range groups {
						kv.Row("  "+g.label, strings.Join(g.versions, ", "))
					}
				}
				if len(profile.MachineTypes) > 0 {
					kv.Row("Machine types", "")
					for _, mt := range profile.MachineTypes {
						kv.Row("  "+mt.Name, machineTypeLabel(mt))
					}
				}
				// Fold the image name into the header for the common single-image
				// case; nest per-image only when a profile offers several.
				writeImageGroups := func(indent string, img api.GardenerCloudProfileMachineImage) {
					for _, g := range groupVersions(classifyImages(img.Versions)) {
						kv.Row(indent+g.label, strings.Join(g.versions, ", "))
					}
				}
				switch len(profile.MachineImages) {
				case 0:
				case 1:
					img := profile.MachineImages[0]
					kv.Row("Machine images ("+img.Name+")", "")
					writeImageGroups("  ", img)
				default:
					kv.Row("Machine images", "")
					for _, img := range profile.MachineImages {
						kv.Row("  "+img.Name, "")
						writeImageGroups("    ", img)
					}
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

// versionInfo is a version with its non-supported classification (empty when
// supported) and optional expiration; used to group cloud-profile version
// lists for display.
type versionInfo struct {
	version string
	class   string
	expiry  *time.Time
}

type versionGroup struct {
	label    string
	versions []string
}

func classifyK8s(versions []api.GardenerCloudProfileKubernetesVersion) []versionInfo {
	out := make([]versionInfo, 0, len(versions))
	for _, v := range versions {
		out = append(out, versionInfo{v.Version, nonSupportedClass(v.Classification), v.ExpirationDate})
	}
	return out
}

func classifyImages(versions []api.GardenerCloudProfileMachineImageVersion) []versionInfo {
	out := make([]versionInfo, 0, len(versions))
	for _, v := range versions {
		out = append(out, versionInfo{v.Version, nonSupportedClass(v.Classification), v.ExpirationDate})
	}
	return out
}

// nonSupportedClass returns "" for a nil or "supported" classification, else
// the classification string (e.g. "preview", "deprecated").
func nonSupportedClass(c *api.K8sVersionClassification) string {
	if c == nil || *c == api.Supported {
		return ""
	}
	return string(*c)
}

// groupVersions buckets versions by (classification, expiration date) so
// versions sharing a status and deadline share one line. Groups are ordered
// supported-first, then by soonest expiry (undated last); the label is the
// classification, plus " (expires <date>)" when the group expires. Version
// order within a group is preserved (the API lists them ascending).
func groupVersions(versions []versionInfo) []versionGroup {
	type bucket struct {
		class    string // "" == supported
		date     string // "" or YYYY-MM-DD
		versions []string
	}
	order := []string{}
	buckets := map[string]*bucket{}
	for _, v := range versions {
		date := ""
		if v.expiry != nil {
			date = v.expiry.Format("2006-01-02")
		}
		key := v.class + "|" + date
		b := buckets[key]
		if b == nil {
			b = &bucket{class: v.class, date: date}
			buckets[key] = b
			order = append(order, key)
		}
		b.versions = append(b.versions, v.version)
	}

	bs := make([]*bucket, 0, len(order))
	for _, key := range order {
		bs = append(bs, buckets[key])
	}
	sort.SliceStable(bs, func(i, j int) bool {
		a, b := bs[i], bs[j]
		if aSup, bSup := a.class == "", b.class == ""; aSup != bSup {
			return aSup // supported groups first
		}
		if (a.date == "") != (b.date == "") {
			return a.date != "" // dated (more urgent) before undated
		}
		if a.date != b.date {
			return a.date < b.date // soonest expiry first (YYYY-MM-DD sorts chronologically)
		}
		return a.class < b.class
	})

	groups := make([]versionGroup, 0, len(bs))
	for _, b := range bs {
		label := b.class
		if label == "" {
			label = "supported"
		}
		if b.date != "" {
			label += " (expires " + b.date + ")"
		}
		groups = append(groups, versionGroup{label, b.versions})
	}
	return groups
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
