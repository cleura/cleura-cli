## cleura user edit

Change a user's details, privileges or password

### Synopsis

Change a Cleura account user's details or password. Only the flags you pass
are sent, so an unset field is never overwritten.

--ip-restriction replaces the whole set; --clear-ip-restrictions removes it.
--set-password prompts for a new password (no-echo, or piped stdin).

Privileges are managed separately, with 'cleura user privilege'. Account-admin
rights cannot be changed through the API.

```
cleura user edit <user-id or username> [flags]
```

### Examples

```
  cleura user edit johndoe --email new.address@example.org
  cleura user edit johndoe --first-name Jonathan
  cleura user edit 4763 --set-password
  cleura user edit ci-bot --ip-restriction 203.0.113.0/24
```

### Options

```
      --clear-ip-restrictions        Remove all login CIDR restrictions
      --email string                 New email address (may require verification)
      --first-name string            New first name
  -h, --help                         help for edit
      --ip-restriction stringArray   Replace the login CIDR restrictions (repeatable)
      --last-name string             New last name
  -o, --output string                Output format: table, json, yaml (default "table")
      --set-password                 Prompt for a new password (read no-echo, or from piped stdin)
      --username string              New username
  -y, --yes                          Skip the confirmation shown when renaming the account this profile is logged in as
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

