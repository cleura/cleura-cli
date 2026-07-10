## cleura gardener shoot monitoring nodes

Show per-node resource usage for a worker group

### Synopsis

Show a per-node snapshot of CPU (used/idle), memory (used/free) and pod
count for a worker group. With --names-only, print just the node names
(useful for scripts).

The table shows cores (CPU) and GiB (memory); -o json reports the raw API values (cores and bytes).

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot monitoring nodes <shoot-name> <worker-group> [flags]
```

### Examples

```
  cleura gardener shoot monitoring nodes prod default
  cleura gardener shoot monitoring nodes prod default --names-only
```

### Options

```
  -h, --help            help for nodes
      --names-only      Print only the node names
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

