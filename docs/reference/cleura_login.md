## cleura login

Log in to Cleura Cloud and store an API token

### Synopsis

Log in to Cleura Cloud with username and password and store the resulting
API token in the configuration file. The profile you log in to becomes the
current profile.

SMS is the only two-factor method the CLI supports; accounts with SMS 2FA are
prompted for the code and must log in from an interactive terminal. WebAuthn
accounts can create an API token in the Control Panel instead and store it
with --token-stdin.

For non-interactive use (CI), set CLEURA_API_PASSWORD in the environment — no
prompt, no secrets on the command line (single-factor accounts only). The
password can also be piped on stdin. Alternatively, store a pre-created API
token with --token-stdin (validated before storing).

```
cleura login [flags]
```

### Examples

```
  cleura login
  cleura login --profile compliant --cloud compliant
  CLEURA_API_PASSWORD=... cleura login -u johndoe     # CI: set as a masked variable
  echo "$TOKEN" | cleura login -u johndoe --token-stdin
```

### Options

```
  -h, --help              help for login
      --token-stdin       Read an existing API token from stdin and store it instead of logging in with a password
  -u, --username string   Username to log in with [$CLEURA_API_USERNAME]
```

### Options inherited from parent commands

```
      --api-url string      Cleura API base URL, required for private clouds; overrides --cloud [$CLEURA_API_URL]
      --cloud string        Named cloud with a predefined API URL: public or compliant [$CLEURA_CLOUD]
      --debug               Log HTTP requests and responses to stderr (credentials redacted)
  -o, --output string       Output format: table, json, yaml (default "table")
      --profile string      Configuration profile to use [$CLEURA_PROFILE] (default from config, or "default")
      --project-id string   OpenStack project ID [$CLEURA_PROJECT_ID]
  -q, --quiet               Suppress informational messages; errors and requested output are still shown
      --region string       OpenStack region (e.g. sto1) [$CLEURA_REGION]
```

### SEE ALSO

* [cleura](cleura.md)	 - Command-line interface for Cleura Cloud

