## cleura config path

Print the configuration file path

### Synopsis

Print the configuration file path: $CLEURA_CONFIG if set, otherwise
$XDG_CONFIG_HOME/cleura/config.yaml, otherwise ~/.config/cleura/config.yaml
(the OS config directory on Windows).

```
cleura config path [flags]
```

### Examples

```
  cleura config path
```

### Options

```
  -h, --help   help for path
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

