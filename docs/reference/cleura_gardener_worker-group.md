## cleura gardener worker-group

Inspect a shoot's worker groups (node pools)

### Synopsis

Inspect the worker groups (node pools) of a shoot cluster.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener worker-group [flags]
```

### Options

```
  -h, --help   help for worker-group
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

* [cleura gardener](cleura_gardener.md)	 - Manage Gardener Kubernetes clusters
* [cleura gardener worker-group list](cleura_gardener_worker-group_list.md)	 - List a shoot's worker groups

