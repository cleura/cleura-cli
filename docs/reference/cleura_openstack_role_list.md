## cleura openstack role list

List the assignable OpenStack roles

### Synopsis

List the OpenStack (Keystone) roles available in a domain — the names accepted by 'cleura openstack role assignment create --role'.

```
cleura openstack role list [flags]
```

### Examples

```
  cleura openstack role list
  cleura openstack role list -o json
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

* [cleura openstack role](cleura_openstack_role.md)	 - View OpenStack roles and manage role assignments

