## cleura gardener shoot monitoring

Inspect a shoot's monitoring data and credentials

### Synopsis

Inspect a shoot's observability data: dashboard credentials and node /
worker-group resource metrics.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot monitoring [flags]
```

### Options

```
  -h, --help   help for monitoring
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

* [cleura gardener shoot](cleura_gardener_shoot.md)	 - Manage shoot clusters
* [cleura gardener shoot monitoring credentials](cleura_gardener_shoot_monitoring_credentials.md)	 - Show a shoot's monitoring (Prometheus/Plutono) credentials
* [cleura gardener shoot monitoring node](cleura_gardener_shoot_monitoring_node.md)	 - Show detailed metrics for one node
* [cleura gardener shoot monitoring nodes](cleura_gardener_shoot_monitoring_nodes.md)	 - Show per-node resource usage for a worker group
* [cleura gardener shoot monitoring worker-group](cleura_gardener_shoot_monitoring_worker-group.md)	 - Show aggregate metrics for a worker group

