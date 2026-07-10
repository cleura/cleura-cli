## cleura gardener shoot monitoring worker-group

Show aggregate metrics for a worker group

### Synopsis

Show aggregate CPU/memory/node/pod metrics for a worker group. The table
shows the latest sample; -o json/yaml carry the full time series.

Values are in cores (CPU), GiB (memory and filesystem) and percent (utilization).

Manage worker groups with 'cleura gardener shoot worker-group'.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot monitoring worker-group <shoot-name> <worker-group> [flags]
```

### Examples

```
  cleura gardener shoot monitoring worker-group prod default
```

### Options

```
  -h, --help            help for worker-group
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

* [cleura gardener shoot monitoring](cleura_gardener_shoot_monitoring.md)	 - Inspect a shoot's monitoring data and credentials

