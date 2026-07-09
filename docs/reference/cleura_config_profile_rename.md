## cleura config profile rename

Rename a profile

### Synopsis

Rename a profile, keeping its stored token and settings (the token
belongs to the same account, so no re-login is needed). current_profile follows
the rename. Refuses if the new name already exists.

```
cleura config profile rename <old> <new> [flags]
```

### Examples

```
  cleura config profile rename default work
```

### Options

```
  -h, --help   help for rename
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

