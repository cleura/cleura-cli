## cleura gardener project bootstrap

Enable Gardener for the selected project

### Synopsis

Enable (bootstrap) Gardener for the selected OpenStack project — a one-time
setup required before shoots can be created in it.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener project bootstrap [flags]
```

### Examples

```
  cleura gardener project bootstrap --region sto1 --project-id a1b2c3
```

### Options

```
  -h, --help   help for bootstrap
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

* [cleura gardener project](cleura_gardener_project.md)	 - Manage Gardener enablement for a project

