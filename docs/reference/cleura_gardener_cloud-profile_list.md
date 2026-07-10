## cleura gardener cloud-profile list

List the cloud profiles in the cloud

### Synopsis

List the Gardener cloud profiles in the selected cloud, with a count of the
supported Kubernetes versions, usable machine types and regions each offers.

This is cloud-wide: it needs a cloud (--cloud, or the profile's cloud), but no
region or project.

```
cleura gardener cloud-profile list [flags]
```

### Examples

```
  cleura gardener cloud-profile list
  cleura gardener cloud-profile list -o json
```

### Options

```
  -h, --help            help for list
  -o, --output string   Output format: table, json, yaml (default "table")
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura gardener cloud-profile](cleura_gardener_cloud-profile.md)	 - Inspect Gardener cloud profiles (versions, machine types, images)

