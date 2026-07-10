## cleura whoami

Show the currently authenticated user

### Synopsis

Show the account the current credentials authenticate as — its ID, username,
email and whether it is an account admin — to confirm which identity and
profile are in effect. For the full privilege breakdown, use 'cleura user get'.

```
cleura whoami [flags]
```

### Examples

```
  cleura whoami
  cleura whoami -o json
```

### Options

```
  -h, --help            help for whoami
  -o, --output string   Output format: table, json, yaml (default "table")
```

### Options inherited from parent commands

```
      --api-url string   Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string     Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug            Log HTTP requests and responses to stderr (credentials redacted)
      --profile string   Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
  -q, --quiet            Suppress informational messages; errors and requested output are still shown
```

### SEE ALSO

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud

