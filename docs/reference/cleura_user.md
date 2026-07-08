## cleura user

View users in the Cleura account

```
cleura user [flags]
```

### Options

```
  -h, --help   help for user
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
  -o, --output string       Output format: table, json, yaml (default "table")
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud
* [cleura user get](cleura_user_get.md)	 - Show one user with the full privilege breakdown
* [cleura user list](cleura_user_list.md)	 - List the users in the account

