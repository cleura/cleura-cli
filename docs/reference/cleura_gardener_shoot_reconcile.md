## cleura gardener shoot reconcile

Trigger a reconciliation of a shoot cluster

### Synopsis

Ask Gardener to run its reconciliation loop for this shoot now, instead of
waiting for the periodic cycle — it applies pending changes and can recover a
cluster stuck after a transient error. A region and project must be selected
(--region/--project-id, CLEURA_REGION/CLEURA_PROJECT_ID, or stored at login).

```
cleura gardener shoot reconcile <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot reconcile prod
```

### Options

```
  -h, --help   help for reconcile
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

