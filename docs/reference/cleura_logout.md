## cleura logout

Log out and revoke the stored API token

### Synopsis

Revoke the selected profile's stored API token server-side and remove it from
the configuration file. The profile itself (endpoint, username, region,
project) is kept for the next login.

Revocation deliberately targets the profile's own stored token: a token in
CLEURA_API_TOKEN is never touched by logout.

```
cleura logout [flags]
```

### Examples

```
  cleura logout
  cleura logout --profile ci -q   # CI cleanup: quiet, exit 0 even when nothing is stored
```

### Options

```
  -h, --help   help for logout
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

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud

