## cleura user privilege set

Grant a user access to an area or a project

### Synopsis

Grant a user access, either across an area or on one OpenStack project.

  --area <area> --level full|read      access to everything in that area
  --project-id <project> --level ...   access to one project only

Setting a project grant leaves that user's other project grants alone, so
repeat the command to add more. Setting an area replaces whatever that area
had, including its per-project grants, and the command says so when it does.

--project-id takes a project ID, or a name the CLI resolves against the
projects you can see. Project names are not unique across regions, so a name
that matches more than one project is refused; qualify it as region/name or
pass the ID. For a project you cannot see yourself, pass its ID together with
--domain-id.

```
cleura user privilege set <user-id or username> [flags]
```

### Examples

```
  cleura user privilege set johndoe --area openstack --level full
  cleura user privilege set johndoe --project-id team-alpha --level read
  cleura user privilege set johndoe --project-id sto2/team-alpha --level full
  cleura user privilege set johndoe --project-id 8a22c50f… --domain-id eb76… --level read
```

### Options

```
      --area string         Privilege area to grant access to
      --domain-id string    Domain the project is in (needed only for a project that is not in your own project list)
  -h, --help                help for set
      --level string        Access level: full or read
  -o, --output string       Output format: table, json, yaml (default "table")
      --project-id string   Project to grant access on, by ID or name (mutually exclusive with --area)
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

* [cleura user privilege](cleura_user_privilege.md)	 - View and change a user's privileges

