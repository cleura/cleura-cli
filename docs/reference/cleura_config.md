## cleura config

Manage CLI configuration and profiles

```
cleura config [flags]
```

### Options

```
  -h, --help   help for config
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

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud
* [cleura config current](cleura_config_current.md)	 - Print the profile the next command would use
* [cleura config delete-profile](cleura_config_delete-profile.md)	 - Remove a profile from the configuration
* [cleura config get-credentials](cleura_config_get-credentials.md)	 - Print the effective credentials as JSON, for tool integration
* [cleura config list-profiles](cleura_config_list-profiles.md)	 - List configured profiles
* [cleura config path](cleura_config_path.md)	 - Print the configuration file path
* [cleura config rename-profile](cleura_config_rename-profile.md)	 - Rename a profile
* [cleura config set](cleura_config_set.md)	 - Set a profile value without logging in again
* [cleura config use-profile](cleura_config_use-profile.md)	 - Set the current profile
* [cleura config view](cleura_config_view.md)	 - Show the effective settings and where each value comes from

