## cleura user create

Create a user in the account

### Synopsis

Create a Cleura account user. The password is read from a no-echo prompt, or
from stdin when piped — it is never passed on the command line.

The new user gets no privileges. Grant them with 'cleura user privilege set'.

Account-admin rights cannot be set through the API.

```
cleura user create <username> [flags]
```

### Examples

```
  cleura user create johndoe --email john.doe@example.org
  cleura user create johndoe --email john.doe@example.org --first-name John --last-name Doe
  printf '%s' "$PASSWORD" | cleura user create ci-bot --email ci@example.org   # non-interactive (CI)
```

### Options

```
      --email string                 Email address (required)
      --first-name string            First name
  -h, --help                         help for create
      --ip-restriction stringArray   Restrict logins to a CIDR (repeatable)
      --last-name string             Last name
  -o, --output string                Output format: table, json, yaml (default "table")
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

* [cleura user](cleura_user.md)	 - Manage users in the Cleura account

