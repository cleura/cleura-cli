## cleura config set

Set a profile value without logging in again

### Synopsis

Set one value in the selected profile. Keys: api_url, cloud, project_id, region, username.
An empty value ("") removes the stored value. Tokens cannot be set here; use
'cleura login' (or 'cleura login --token-stdin').

```
cleura config set <key> <value> [flags]
```

### Examples

```
  cleura config set region kna1
  cleura config set project_id a1b2c3
  cleura config set --profile acme api_url https://rest.cloud.acme.example
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura config](cleura_config.md)	 - Manage CLI configuration and profiles

