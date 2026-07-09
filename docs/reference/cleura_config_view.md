## cleura config view

Show the effective settings and where each value comes from

### Synopsis

Show the settings the next command would use, resolved with the usual
precedence (flags > environment > profile > defaults), and the source of each
value. Values set by environment variables that shadow something stored in the
selected profile are pointed out on stderr. The token value is never shown.

```
cleura config view [flags]
```

### Examples

```
  cleura config view
  cleura config view --profile compliant -o json
```

### Options

```
  -h, --help            help for view
  -o, --output string   Output format: table, json, yaml (default "table")
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

