## cleura config profile delete

Remove a profile from the configuration

### Synopsis

Remove a profile from the configuration file. A stored token is revoked
server-side first (best effort — the profile is deleted even when revocation
fails, and the warning says so).

```
cleura config profile delete <name> [flags]
```

### Examples

```
  cleura config profile delete old-test
```

### Options

```
  -h, --help   help for delete
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud: public, compliant, or a private cloud's name (with --api-url) [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura config profile](cleura_config_profile.md)	 - Manage named profiles (list, use, set, rename, delete)

