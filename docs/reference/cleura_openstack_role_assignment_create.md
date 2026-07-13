## cleura openstack role assignment create

Grant a user roles on a project

### Synopsis

Grant an OpenStack user one or more roles on a project (an additive role
assignment). The user is given by name or ID and the roles by name (resolved
against the domain's roles — see 'cleura openstack role list'); the project is
an ID.

```
cleura openstack role assignment create [flags]
```

### Examples

```
  cleura openstack role assignment create --user alice --role member --project-id <id>
  cleura openstack role assignment create --user alice --role member,load-balancer_member --project-id <id>
```

### Options

```
      --domain string       OpenStack domain ID (required unless the account has a single domain)
  -h, --help                help for create
      --project-id string   Project ID to grant access on
      --role strings        Role name(s) to grant, comma-separated (see 'cleura openstack role list')
      --user string         User name or ID to grant access to
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

