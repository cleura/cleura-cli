## cleura openstack user

Manage OpenStack users

### Synopsis

Manage OpenStack (Keystone) users, which live under a domain.

These are distinct from Cleura account users ('cleura user'): OpenStack users
authenticate against OpenStack itself and are granted access to projects via
roles ('cleura openstack role assignment create').

```
cleura openstack user [flags]
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

* [cleura openstack](cleura_openstack.md)	 - Manage OpenStack identity resources (domains, projects, users, roles)
* [cleura openstack user create](cleura_openstack_user_create.md)	 - Create an OpenStack user in a domain
* [cleura openstack user delete](cleura_openstack_user_delete.md)	 - Delete an OpenStack user
* [cleura openstack user list](cleura_openstack_user_list.md)	 - List OpenStack users in a domain

