## cleura openstack project edit

Update an OpenStack project

### Synopsis

Update an OpenStack project's name, description, or enabled state. Only the
flags you pass are changed.

The API has no project delete; --disable is the closest equivalent — the project
stays but is turned off.

```
cleura openstack project edit <project-id> [flags]
```

### Examples

```
  cleura openstack project edit <project-id> --name new-name
  cleura openstack project edit <project-id> --description "archived"
  cleura openstack project edit <project-id> --disable
```

### Options

```
      --description string   New project description (pass "" to clear)
      --disable              Disable the project (the closest thing to deletion)
      --domain string        OpenStack domain ID the project is in (default: the account's only domain)
      --enable               Enable the project
  -h, --help                 help for edit
      --name string          New project name
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

* [cleura openstack project](cleura_openstack_project.md)	 - Manage OpenStack projects

