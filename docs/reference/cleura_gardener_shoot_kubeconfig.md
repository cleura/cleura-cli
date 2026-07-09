## cleura gardener shoot kubeconfig

Create a time-limited admin kubeconfig for a shoot cluster

### Synopsis

Create an admin kubeconfig for a shoot cluster and print it to stdout,
or write it to a file with --file. The credential expires after --expiration
(the API may cap the allowed validity). A region and project must be selected
(--region/--project-id, CLEURA_REGION/CLEURA_PROJECT_ID, or stored at login).

```
cleura gardener shoot kubeconfig <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot kubeconfig prod > prod.kubeconfig
  cleura gardener shoot kubeconfig prod --expiration 8h -f ~/.kube/prod.yaml
  KUBECONFIG=$(pwd)/prod.kubeconfig kubectl get nodes
```

### Options

```
      --expiration duration   How long the kubeconfig stays valid (e.g. 30m, 6h) (default 1h0m0s)
  -f, --file string           Write the kubeconfig to this path instead of stdout (created with mode 0600)
  -h, --help                  help for kubeconfig
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

