## cleura config profile

Manage named profiles (list, use, set, rename, delete)

```
cleura config profile [flags]
```

### Options

```
  -h, --help   help for profile
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
* [cleura config profile current](cleura_config_profile_current.md)	 - Print the profile the next command would use
* [cleura config profile delete](cleura_config_profile_delete.md)	 - Remove a profile from the configuration
* [cleura config profile list](cleura_config_profile_list.md)	 - List configured profiles
* [cleura config profile rename](cleura_config_profile_rename.md)	 - Rename a profile
* [cleura config profile set](cleura_config_profile_set.md)	 - Set a profile value without logging in again
* [cleura config profile use](cleura_config_profile_use.md)	 - Set the current profile

