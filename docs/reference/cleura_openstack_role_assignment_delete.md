## cleura openstack role assignment delete

Revoke a user's role on a project

### Synopsis

Revoke one role from an OpenStack user on a project. The user is given by name
or ID and the role by name; the project is an ID. Reversible with 'create'.

```
cleura openstack role assignment delete [flags]
```

### Examples

```
  cleura openstack role assignment delete --user alice --role member --project-id <id>
```

### Options

```
      --domain-id string    OpenStack domain ID (required unless the account has a single domain)
  -h, --help                help for delete
      --project-id string   Project ID to revoke access on
      --role string         Role name to revoke (see 'cleura openstack role list')
      --user string         User name or ID to revoke access from
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

* [cleura openstack role assignment](cleura_openstack_role_assignment.md)	 - Manage role assignments (a user's roles on a project)

