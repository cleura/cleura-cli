## cleura openstack role

View OpenStack roles and manage role assignments

### Synopsis

View the OpenStack (Keystone) roles that can be granted to users on projects, and manage role assignments (see 'cleura openstack role assignment').

```
cleura openstack role [flags]
```

### Options

```
  -h, --help   help for role
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

* [cleura openstack](cleura_openstack.md)	 - Manage OpenStack identity resources (domains, projects, users, roles)
* [cleura openstack role assignment](cleura_openstack_role_assignment.md)	 - Manage role assignments (a user's roles on a project)
* [cleura openstack role list](cleura_openstack_role_list.md)	 - List the assignable OpenStack roles

