## cleura openstack project

Manage OpenStack projects

### Synopsis

Manage OpenStack projects, which live under a domain.

```
cleura openstack project [flags]
```

### Options

```
  -h, --help   help for project
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
* [cleura openstack project create](cleura_openstack_project_create.md)	 - Create an OpenStack project in a domain
* [cleura openstack project edit](cleura_openstack_project_edit.md)	 - Update an OpenStack project
* [cleura openstack project list](cleura_openstack_project_list.md)	 - List your OpenStack projects

