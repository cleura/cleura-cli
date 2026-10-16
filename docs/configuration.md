# Configuration

Settings are applied in this order: **flags > environment > profile > defaults**.
Within each level, an explicit API URL (`--api-url`, `CLEURA_API_URL`, `api_url`) wins over a named cloud (`--cloud`, `CLEURA_CLOUD`, `cloud`).
The `cleura login` command records the endpoint it logged into in the profile, together with the username and a token.

## Profiles

Working with more than one cloud is a matter of profiles:

```sh
cleura login --profile compliant --cloud compliant   # logs into a separate profile, which then becomes the current profile
cleura --profile production whoami                   # one-off use of another profile
cleura config profile use production                 # switches the current profile without logging in
cleura config profile current                        # which profile am I on?
cleura config profile list
cleura config profile rename production prod         # rename (keeps the token; current_profile follows)
cleura config profile delete compliant
```

Logging in always makes that profile the current one; the `config profile use` subcommand switches between already-logged-in profiles.

When in doubt about what a command will actually use, the `cleura config view` subcommand shows the effective settings and the source of each value (flag, `$CLEURA_*` variable, profile, or default), and warns when an environment variable takes precedence over a stored profile value.

The `CLEURA_PROFILE` variable is used to select a profile for scripted use, without touching the config file.

## Config file

The config file lives at `$CLEURA_CONFIG`, or `$XDG_CONFIG_HOME/cleura/config.yaml`, or `~/.config/cleura/config.yaml`:

```yaml
current_profile: default
profiles:
  default:
    cloud: public            # public, compliant, or a private cloud name (see acme)
    username: johndoe
    token: "..."             # stored by "cleura login"
    region: sto2             # optional defaults for project-scoped commands
    project_id: a1b2c3
  acme:                      # a private cloud sets both: the URL for the endpoint,
    cloud: acme              # and the cloud name for the API path parameters
    api_url: https://rest.cloud.acme.example
    username: johndoe
```

> **The config file format is internal, and not exposed via an API.**
> It is shown here so you know what is stored where.
> Its schema may change between releases.
> Programs and scripts must not parse it — use `cleura config get-credentials` (see [Tool integration](#tool-integration)), the stable JSON contract.
> (The `cleura config view` command is used for manual diagnostics, and its output format may change.)

### Environment variables

Shared with the [Terraform provider](https://github.com/cleura/terraform-provider-cleura):

| Variable              | Purpose                                        |
|-----------------------|------------------------------------------------|
| `CLEURA_API_URL`      | API base URL (required for private clouds)     |
| `CLEURA_API_USERNAME` | Username                                       |
| `CLEURA_API_TOKEN`    | API token (skips `cleura login`); the API requires it **together with** `CLEURA_API_USERNAME` |
| `CLEURA_API_PASSWORD` | Password for `cleura login` (CLI-only, never stored); the prompt-free CI login path |
| `CLEURA_CLOUD`        | Named cloud: `public`, `compliant`, or a private cloud's name (set with `CLEURA_API_URL`) |
| `CLEURA_REGION`       | OpenStack region (e.g. `sto1`)                 |
| `CLEURA_PROJECT_ID`   | OpenStack project ID                           |
| `CLEURA_PROFILE`      | Profile to use                                 |
| `CLEURA_CONFIG`       | Config file path override (use an absolute path: relative paths resolve against each process's own working directory) |

### Global flags

On every command:
`--profile`, `--cloud`, `--api-url`, `--quiet/-q` (suppress informational messages; stdout carries only data, so pipes stay clean), `--debug` (log HTTP exchanges to stderr with credentials redacted).

### Scoped flags

Offered only where they apply:
`--output/-o` (table, json, yaml) on commands that render output;
`--region` and `--project-id` on `gardener` commands (and on `cleura login`, which stores them in the profile);
`--domain-id` on most `cleura openstack` commands (normally required — an account usually has several domains, one per region).

Environment variables `CLEURA_REGION`/`CLEURA_PROJECT_ID` still apply everywhere they are relevant.

## Profile values

Profile values can be changed without re-logging in:

```sh
cleura config profile set region kna1
cleura config profile set project_id a1b2c3
cleura config path                     # where the config file lives
```

## Tool integration

`cleura config get-credentials` is the stable interface for tools that authenticate via the CLI (the Terraform provider's `cleura` credential source, scripts, anything else).
It prints the effective credentials — resolved with the usual precedence — as one JSON object:

```console
$ cleura config get-credentials
{
  "version": 1,
  "profile": "work",
  "cloud": "public",
  "endpoint": "https://rest.cleura.cloud",
  "username": "svc",
  "token": "...",
  "region": "sto2",
  "project_id": "p-1",
  "token_stored_at": "2026-07-08T08:00:00Z"
}

$ cleura config get-credentials | jq -r .token   # e.g. for curl scripts
```

Exit codes are part of the contract:
`0` credentials printed; `2` no usable credentials for the selected profile (a JSON `{"error": ...}` object is printed on stdout, so credential chains can fall through to their next source);
any other exit is a malfunction.

The `--validate` switch verifies the token against the API first.

Compatibility: fields are only added — never renamed or removed — while `version` is `1`;
a breaking change bumps it, and users must check it.

The `region`, `project_id`, `cloud` and `token_stored_at` fields are omitted when unset.
In particular, `cloud` is absent for a private cloud configured with only an `api_url` and no cloud name.

`config get-credentials` is the only versioned, stable JSON contract.
Every other command's `-o json`/`-o yaml` output is for convenience:
it mirrors the unversioned API plus a few CLI-computed fields and may change between releases, so don't build automation that depends on its exact shape.

## Command reference

Every command and flag is documented in [`docs/reference/`](reference/cleura.md), generated from the CLI's own help text (`make docs` regenerates it — the reference cannot drift from the binary).
