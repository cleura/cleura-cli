## cleura config use-profile

Set the current profile

```
cleura config use-profile <name> [flags]
```

### Examples

```
  cleura config use-profile compliant
```

### Options

```
  -h, --help   help for use-profile
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

