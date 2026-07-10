## cleura gardener shoot monitoring credentials

Show the shoot's monitoring (Prometheus/Plutono) credentials

### Synopsis

Show the Prometheus and Plutono dashboard URLs and logins for a shoot.

Passwords are masked in table output; reveal them with --show-secrets or
-o json/yaml.

A region and project must be selected for gardener commands: pass
--region/--project-id, set CLEURA_REGION/CLEURA_PROJECT_ID, or store them in the
profile with 'cleura login'.

```
cleura gardener shoot monitoring credentials <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot monitoring credentials prod
  cleura gardener shoot monitoring credentials prod --show-secrets
```

### Options

```
  -h, --help            help for credentials
  -o, --output string   Output format: table, json, yaml (default "table")
      --show-secrets    Show passwords in table output instead of masking them
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

