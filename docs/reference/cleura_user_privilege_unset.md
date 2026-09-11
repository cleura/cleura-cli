## cleura user privilege unset

Remove a user's access to an area or a project

### Synopsis

Remove access, either to a whole area or to one OpenStack project.

  --area <area>            remove all access to that area
  --project-id <project>   remove one project grant, leaving the others

Removing the last per-project grant removes the area itself, since an empty
project list grants nothing.

```
cleura user privilege unset <user-id or username> [flags]
```

### Examples

```
  cleura user privilege unset johndoe --area invoice
  cleura user privilege unset johndoe --project-id team-alpha
```

### Options

```
      --area string         Privilege area to remove access to
  -h, --help                help for unset
  -o, --output string       Output format: table, json, yaml (default "table")
      --project-id string   Project to remove access on, by ID or name (mutually exclusive with --area)
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

* [cleura user privilege](cleura_user_privilege.md)	 - View and change a user's privileges

