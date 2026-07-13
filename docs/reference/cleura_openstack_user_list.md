## cleura openstack user list

List OpenStack users in a domain

### Synopsis

List the OpenStack (Keystone) users in a domain.

```
cleura openstack user list [flags]
```

### Examples

```
  cleura openstack user list
  cleura openstack user list -o json
```

### Options

```
      --domain string   OpenStack domain ID (default: the account's only domain)
  -h, --help            help for list
  -o, --output string   Output format: table, json, yaml (default "table")
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

