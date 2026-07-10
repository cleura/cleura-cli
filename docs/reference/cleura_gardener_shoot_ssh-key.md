## cleura gardener shoot ssh-key

Fetch the shoot's node SSH private key

### Synopsis

Fetch the SSH private key for a shoot's nodes.

This is a secret: write it to a 0600 file with --file, or print it to
stdout with --stdout (exactly one is required).

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot ssh-key <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot ssh-key prod -f ~/.ssh/prod-nodes.pem
  cleura gardener shoot ssh-key prod --stdout
```

### Options

```
  -f, --file string   Write the key to this path (created with mode 0600)
  -h, --help          help for ssh-key
      --stdout        Print the key to stdout instead of a file
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

