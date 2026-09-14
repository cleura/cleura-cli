## cleura user

Manage users in the Cleura account

### Synopsis

View and manage the users in your Cleura account and their privileges. This
needs the 'users' privilege or account-admin rights; to see your own account use
'cleura whoami'.

Privileges are their own noun: 'cleura user privilege' grants and revokes
access, per area or per OpenStack project. Account-admin rights (the ADMIN
column) cannot be granted or revoked through the API — the create and edit
request bodies have no such field. Use the Control Panel for that.

```
cleura user [flags]
```

### Options

```
  -h, --help   help for user
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

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud
* [cleura user create](cleura_user_create.md)	 - Create a user in the account
* [cleura user delete](cleura_user_delete.md)	 - Delete a user from the account
* [cleura user edit](cleura_user_edit.md)	 - Change a user's details, privileges or password
* [cleura user get](cleura_user_get.md)	 - Show one user with the full privilege breakdown
* [cleura user list](cleura_user_list.md)	 - List the users in the account
* [cleura user privilege](cleura_user_privilege.md)	 - View and change a user's privileges

