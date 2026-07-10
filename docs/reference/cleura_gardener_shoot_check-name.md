## cleura gardener shoot check-name

Check whether a shoot name is available in the cloud

### Synopsis

Check whether a shoot name is already taken in the selected cloud.

This is cloud-wide: it needs a cloud (--cloud, or the profile's cloud), but no
region or project.

With --exit-code the command is a script predicate: it exits 0 if the name
is available, 2 if it is taken, and 1 (or another non-zero) if the check
itself failed — so 'taken' is never confused with an error.

```
cleura gardener shoot check-name <shoot-name> [flags]
```

### Examples

```
  cleura gardener shoot check-name prod
  cleura gardener shoot check-name prod --exit-code && echo name is free
```

### Options

```
      --exit-code       For scripting: exit 0 if available, 2 if taken, 1 if the check failed
  -h, --help            help for check-name
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

* [cleura gardener shoot](cleura_gardener_shoot.md)	 - Manage shoot clusters

