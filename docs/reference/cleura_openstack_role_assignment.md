## cleura openstack role assignment

Manage role assignments (a user's roles on a project)

### Synopsis

Manage OpenStack role assignments — the binding of a user, a project, and one
or more roles. 'create' grants access, 'delete' revokes it, and 'list' shows a
user's assignments across projects.

```
cleura openstack role assignment [flags]
```

### Options

```
  -h, --help   help for assignment
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

* [cleura openstack role](cleura_openstack_role.md)	 - View OpenStack roles and manage role assignments
* [cleura openstack role assignment create](cleura_openstack_role_assignment_create.md)	 - Grant a user roles on a project
* [cleura openstack role assignment delete](cleura_openstack_role_assignment_delete.md)	 - Revoke a user's role on a project
* [cleura openstack role assignment list](cleura_openstack_role_assignment_list.md)	 - List a user's role assignments across projects

