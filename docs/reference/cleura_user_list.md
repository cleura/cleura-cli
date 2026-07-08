## cleura user list

List the users in the account

### Synopsis

List the users in the account with their privileges. The ROLES column
summarizes each privilege area as area:type (types: full, read, or
project(n) for per-project grants); account administrators show as "admin".

Viewing other users requires the users privilege or account administrator
rights on the logged-in account.

```
cleura user list [flags]
```

### Examples

```
  cleura user list
  cleura user list -o json
```

### Options

```
  -h, --help   help for list
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

* [cleura user](cleura_user.md)	 - View users in the Cleura account

