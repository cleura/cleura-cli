## cleura openstack

Manage OpenStack identity resources (domains, projects, users, roles)

### Synopsis

Manage OpenStack (Keystone) identity resources — domains, projects, users, and
role assignments.

These are distinct from Cleura account users ('cleura user'): OpenStack users
authenticate against OpenStack itself. Almost every command is scoped to a domain
and needs --domain <id>; an account usually has several domains (one per region),
so it is normally required — the CLI auto-selects only when there is exactly one.
'domain list' and 'project list' take no --domain; use them to find the IDs.

```
cleura openstack [flags]
```

### Options

```
  -h, --help   help for openstack
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
* [cleura openstack domain](cleura_openstack_domain.md)	 - View OpenStack domains
* [cleura openstack project](cleura_openstack_project.md)	 - Manage OpenStack projects
* [cleura openstack role](cleura_openstack_role.md)	 - View OpenStack roles and manage role assignments
* [cleura openstack user](cleura_openstack_user.md)	 - Manage OpenStack users

