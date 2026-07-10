## cleura gardener shoot get

Show detailed information about a shoot cluster

### Synopsis

Show detailed information about a single shoot cluster — worker groups,
maintenance window, hibernation schedule, networking, high availability and
status — that the summary 'shoot list' omits.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot get <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot get prod
  cleura gardener shoot get prod -o json
```

### Options

```
  -h, --help            help for get
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

* [cleura gardener shoot](cleura_gardener_shoot.md)	 - Manage shoot clusters

