## cleura

Command-line interface for Cleura Cloud

### Synopsis

cleura is the command-line interface for Cleura Cloud.

Run 'cleura login' first: it stores an API token in a profile in
~/.config/cleura/config.yaml (override with $CLEURA_CONFIG). Settings resolve
with the precedence flags > environment variables > profile > defaults; run
'cleura config view' to see the effective values and where each one comes
from.

### Options

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud: public, compliant, or a private cloud's name (with --api-url) [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
  -h, --help             help for cleura
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
      --version          Show the cleura version
```

### SEE ALSO

* [cleura completion](cleura_completion.md)	 - Generate the autocompletion script for the specified shell
* [cleura config](cleura_config.md)	 - Manage CLI configuration and profiles
* [cleura gardener](cleura_gardener.md)	 - Manage Gardener Kubernetes clusters
* [cleura login](cleura_login.md)	 - Log in to Cleura Cloud and store an API token
* [cleura logout](cleura_logout.md)	 - Revoke the profile's stored API token and remove it from the configuration
* [cleura user](cleura_user.md)	 - View users in the Cleura account
* [cleura version](cleura_version.md)	 - Show the cleura version
* [cleura whoami](cleura_whoami.md)	 - Show the currently authenticated user

