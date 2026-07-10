## cleura gardener cloud-profile show

Show a cloud profile in detail

### Synopsis

Show one Gardener cloud profile in detail: its Kubernetes versions (with any
non-supported ones flagged), machine types, machine images, regions/zones and
volume types. Find the profile names with 'cleura gardener cloud-profile list'.

This is cloud-wide: it needs a cloud (--cloud, or the profile's cloud), but no
region or project.

```
cleura gardener cloud-profile show <profile-name> [flags]
```

### Examples

```
  cleura gardener cloud-profile list
  cleura gardener cloud-profile show <name> -o json
```

### Options

```
  -h, --help            help for show
  -o, --output string   Output format: table, json, yaml (default "table")
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud: public, compliant, or a private cloud's name (with --api-url) [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura gardener cloud-profile](cleura_gardener_cloud-profile.md)	 - Inspect Gardener cloud profiles (versions, machine types, images)

