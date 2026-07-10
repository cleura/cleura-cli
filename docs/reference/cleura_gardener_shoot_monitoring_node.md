## cleura gardener shoot monitoring node

Show detailed metrics for one node

### Synopsis

Show detailed metrics for a single node. The table shows the latest
snapshot; -o json/yaml carry the full time series.

Values are in cores (CPU), GiB (memory and filesystem) and percent (utilization).

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot monitoring node <shoot-name> <node-name> [flags]
```

### Examples

```
  cleura gardener shoot monitoring node prod prod-default-abc12
  cleura gardener shoot monitoring node prod prod-default-abc12 -o json
```

### Options

```
  -h, --help            help for node
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

* [cleura gardener shoot monitoring](cleura_gardener_shoot_monitoring.md)	 - Inspect a shoot's monitoring data and credentials

