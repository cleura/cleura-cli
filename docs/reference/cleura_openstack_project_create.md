## cleura openstack project create

Create an OpenStack project in a domain

### Synopsis

Create an OpenStack project. The project is created in the account's domain;
when the account has more than one domain, choose one with --domain (list them
with 'cleura openstack domain list').

The created project — including its ID — is printed; use -o json to capture the
ID for scripting (e.g. to grant access or launch resources in it).

```
cleura openstack project create <name> [flags]
```

### Examples

```
  cleura openstack project create my-project
  cleura openstack project create my-project --description "team sandbox"
  cleura openstack project create my-project --domain <domain-id> -o json
```

### Options

```
      --description string   Optional project description
      --domain string        OpenStack domain ID to create the project in (required unless the account has a single domain)
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

* [cleura openstack project](cleura_openstack_project.md)	 - Manage OpenStack projects

