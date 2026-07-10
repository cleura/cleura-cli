## cleura gardener

Manage Gardener Kubernetes clusters

### Synopsis

Manage Gardener Kubernetes clusters.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener [flags]
```

### Options

```
  -h, --help                help for gardener
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud: public, compliant, or a private cloud's name (with --api-url) [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud
* [cleura gardener cloud-profile](cleura_gardener_cloud-profile.md)	 - Inspect Gardener cloud profiles (versions, machine types, images)
* [cleura gardener project](cleura_gardener_project.md)	 - Manage Gardener enablement for a project
* [cleura gardener shoot](cleura_gardener_shoot.md)	 - Manage shoot clusters

