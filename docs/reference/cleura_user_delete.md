## cleura user delete

Delete a user from the account

### Synopsis

Delete a Cleura account user, given by numeric ID or exact username. This is
irreversible. The command asks for confirmation and refuses on a
non-interactive terminal unless --yes is given.

The account you are logged in as cannot be deleted here.

```
cleura user delete <user-id or username> [flags]
```

### Examples

```
  cleura user delete johndoe
  cleura user delete 4763 --yes
```

### Options

```
  -h, --help   help for delete
  -y, --yes    Skip the confirmation prompt (required on a non-interactive terminal)
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

