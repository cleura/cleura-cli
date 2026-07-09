## cleura gardener shoot

Manage shoot clusters

### Synopsis

Manage Gardener shoot (Kubernetes) clusters.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot [flags]
```

### Options

```
  -h, --help   help for shoot
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
* [cleura gardener shoot hibernate](cleura_gardener_shoot_hibernate.md)	 - Hibernate a shoot cluster (scales workloads and control plane down)
* [cleura gardener shoot kubeconfig](cleura_gardener_shoot_kubeconfig.md)	 - Create a time-limited admin kubeconfig for a shoot cluster
* [cleura gardener shoot list](cleura_gardener_shoot_list.md)	 - List shoot clusters in a project
* [cleura gardener shoot reconcile](cleura_gardener_shoot_reconcile.md)	 - Trigger a reconciliation of a shoot cluster
* [cleura gardener shoot wake](cleura_gardener_shoot_wake.md)	 - Wake a shoot cluster up from hibernation

