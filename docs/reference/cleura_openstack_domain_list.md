## cleura openstack domain list

List the OpenStack domains for the account

### Synopsis

List the OpenStack (Keystone) domains available to your account, with the domain ID used by 'cleura openstack project create --domain'.

```
cleura openstack domain list [flags]
```

### Examples

```
  cleura openstack domain list
  cleura openstack domain list -o json
```

### Options

```
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

* [cleura openstack domain](cleura_openstack_domain.md)	 - View OpenStack domains

