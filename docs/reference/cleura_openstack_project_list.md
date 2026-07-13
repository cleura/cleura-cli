## cleura openstack project list

List your OpenStack projects

### Synopsis

List the OpenStack projects you can access, across regions.

This lists the caller's own projects — there is no account-wide project listing,
so a project created for another user will not appear here.

```
cleura openstack project list [flags]
```

### Examples

```
  cleura openstack project list
  cleura openstack project list -o json
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

* [cleura openstack project](cleura_openstack_project.md)	 - Manage OpenStack projects

