## cleura openstack

Manage OpenStack identity resources (domains, projects)

### Synopsis

Manage OpenStack (Keystone) identity resources — domains and projects.

These are distinct from Cleura account users ('cleura user'): OpenStack projects
live under a domain and are addressed by an opaque domain ID. Most accounts have a
single domain, which the CLI selects automatically; pass --domain when there is
more than one (list them with 'cleura openstack domain list').

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

