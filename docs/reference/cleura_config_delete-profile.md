## cleura config delete-profile

Remove a profile from the configuration

### Synopsis

Remove a profile from the configuration file. A stored token is revoked
server-side first (best effort — the profile is deleted even when revocation
fails, and the warning says so).

```
cleura config delete-profile <name> [flags]
```

### Examples

```
  cleura config delete-profile old-test
```

### Options

```
  -h, --help   help for delete-profile
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

