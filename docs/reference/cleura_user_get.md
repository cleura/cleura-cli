## cleura user get

Show one user with the full privilege breakdown

### Synopsis

Show one user (by numeric ID or exact username) with the full privilege
breakdown. Viewing another user requires the 'users' privilege or account
administrator rights; to see your own account without that privilege, use
'cleura whoami'.

```
cleura user get <user-id or username> [flags]
```

### Examples

```
  cleura user get 4763
  cleura user get johndoe
```

### Options

```
  -h, --help            help for get
  -o, --output string   Output format: table, json, yaml (default "table")
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

* [cleura user](cleura_user.md)	 - View users in the Cleura account

