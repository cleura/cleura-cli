## cleura openstack role assignment list

List a user's role assignments across projects

### Synopsis

List the projects an OpenStack user can access and the roles they hold on each.
The user is given by name or ID with --user.

```
cleura openstack role assignment list [flags]
```

### Examples

```
  cleura openstack role assignment list --user alice
  cleura openstack role assignment list --user alice -o json
```

### Options

```
      --domain string   OpenStack domain ID (required unless the account has a single domain)
  -h, --help            help for list
  -o, --output string   Output format: table, json, yaml (default "table")
      --user string     User name or ID whose assignments to list
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

* [cleura openstack role assignment](cleura_openstack_role_assignment.md)	 - Manage role assignments (a user's roles on a project)

