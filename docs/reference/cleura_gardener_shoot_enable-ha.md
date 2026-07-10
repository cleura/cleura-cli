## cleura gardener shoot enable-ha

Enable a highly available (multi-zone) control plane

### Synopsis

Enable a highly available control plane for a shoot, spreading it across
failure domains.

This cannot be undone through the CLI and increases cost.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot enable-ha <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot enable-ha prod --yes
```

### Options

```
  -h, --help   help for enable-ha
  -y, --yes    Skip the confirmation prompt (required on a non-interactive terminal)
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

