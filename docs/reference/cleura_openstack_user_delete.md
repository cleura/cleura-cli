## cleura openstack user delete

Delete an OpenStack user

### Synopsis

Delete an OpenStack user, given by name or ID. This is irreversible. The
command asks for confirmation and refuses on a non-interactive terminal unless
--yes is given.

```
cleura openstack user delete <user> [flags]
```

### Examples

```
  cleura openstack user delete alice
  cleura openstack user delete <user-id> --yes
```

### Options

```
      --domain string   OpenStack domain ID the user is in (required unless the account has a single domain)
  -h, --help            help for delete
  -y, --yes             Skip the confirmation prompt (required on a non-interactive terminal)
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

* [cleura openstack user](cleura_openstack_user.md)	 - Manage OpenStack users

