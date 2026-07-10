## cleura config get-credentials

Print the effective credentials as JSON, for tool integration

### Synopsis

Print the effective credentials (resolved with the usual precedence:
flags > environment > profile) as a single JSON object on stdout. This is the
stable interface for tools that authenticate via the cleura CLI, such as the
Terraform provider — the config file itself is internal and must not be parsed.

Output is always JSON; --output does not apply. The token is printed in the
clear — that is this command's purpose.

Exit codes are part of the contract:
  0  credentials printed
  2  no usable credentials for the selected profile (a JSON {"error": ...}
     object is printed on stdout so callers can fall through to their next
     credential source)
  1  anything else went wrong (unreadable config, validation transport error)

Envelope fields are only added, never renamed or removed, while "version" is 1;
a breaking change bumps it. Consumers must check the version field.

```
cleura config get-credentials [flags]
```

### Examples

```
  cleura config get-credentials
  cleura config get-credentials --profile compliant --validate
  cleura config get-credentials | jq -r .token
```

### Options

```
  -h, --help       help for get-credentials
      --validate   Verify the token against the API before printing (one extra request)
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

* [cleura config](cleura_config.md)	 - Manage CLI configuration and profiles

