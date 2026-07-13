## cleura openstack domain

View OpenStack domains

### Synopsis

View the OpenStack (Keystone) domains available to your account. A domain ID
identifies where projects and users live; it is what 'cleura openstack project
create --domain' expects.

```
cleura openstack domain [flags]
```

### Options

```
  -h, --help   help for domain
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
* [cleura openstack domain list](cleura_openstack_domain_list.md)	 - List the OpenStack domains for the account

