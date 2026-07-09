## cleura config profile current

Print the profile the next command would use

### Synopsis

Print the name of the profile that commands resolve to right now
(--profile / $CLEURA_PROFILE / current_profile / "default"), without a network
call — the quick answer to "which profile am I on?".

```
cleura config profile current [flags]
```

### Examples

```
  cleura config profile current
```

### Options

```
  -h, --help   help for current
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

* [cleura config profile](cleura_config_profile.md)	 - Manage named profiles (list, use, set, rename, delete)

