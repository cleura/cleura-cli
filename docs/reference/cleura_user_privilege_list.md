## cleura user privilege list

List a user's privileges

### Synopsis

List one user's privileges, one row per area. Per-project grants are listed
in the SCOPE column as project:level, with project names resolved where the
project is visible to you.

Areas with no access are not listed.

```
cleura user privilege list <user-id or username> [flags]
```

### Examples

```
  cleura user privilege list johndoe
  cleura user privilege list 4763 -o json
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

* [cleura user privilege](cleura_user_privilege.md)	 - View and change a user's privileges

