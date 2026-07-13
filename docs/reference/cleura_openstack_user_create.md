## cleura openstack user create

Create an OpenStack user in a domain

### Synopsis

Create an OpenStack user. The password is read from a no-echo prompt, or from
stdin when piped — it is never passed on the command line. Grant the new user
access to projects afterwards with 'cleura openstack role assignment create'.

```
cleura openstack user create <name> [flags]
```

### Examples

```
  cleura openstack user create alice
  printf '%s' "$PASSWORD" | cleura openstack user create alice   # non-interactive (CI)
  cleura openstack user create svc --description "CI account" -o json
```

### Options

```
      --description string   Optional user description
      --domain string        OpenStack domain ID to create the user in (default: the account's only domain)
  -h, --help                 help for create
  -o, --output string        Output format: table, json, yaml (default "table")
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

