# cleura-cli

`cleura` — the command-line interface for [Cleura Cloud](https://cleura.com/).

Built on the generated [cleura-client-go](https://github.com/cleura/cleura-client-go)
API client, commands are added incrementally as the API surface matures.

## Install

The repositories are private, so Go must fetch modules directly from GitHub
instead of the public module proxy. One-time setup, then install:

```sh
git config --global url."git@github.com:".insteadOf "https://github.com/"  # fetch GitHub over SSH
export GOPRIVATE='github.com/cleura/*'                                     # skip proxy + checksum DB

go install github.com/cleura/cleura-cli/cmd/cleura@latest
```

(In CI, use an access token instead of SSH — see [`examples/ci/`](examples/ci/).)

From a checkout: `make install`. Prebuilt binaries are planned once the
release pipeline lands.

## Getting started

```sh
cleura login              # prompts for username/password (SMS 2FA supported)
cleura whoami             # show the authenticated user
cleura whoami -o json     # machine-readable output
cleura logout             # revoke and remove the stored token

cleura gardener shoot list --region sto1 --project-id <id>   # list Kubernetes clusters
cleura gardener shoot kubeconfig prod > prod.kubeconfig      # time-limited admin kubeconfig
cleura gardener shoot kubeconfig prod --expiration 8h -f ~/.kube/prod.yaml
cleura gardener shoot hibernate prod                         # scale the cluster down
cleura gardener shoot wake prod                              # wake it up again
cleura gardener shoot reconcile prod                         # trigger a reconciliation
```

`--region` and `--project-id` can be stored in the profile (pass them to
`cleura login`) or set via `CLEURA_REGION` / `CLEURA_PROJECT_ID`.

Non-interactive login for CI (single-factor accounts only — SMS 2FA requires an
interactive terminal). Set `CLEURA_API_PASSWORD` as a masked CI variable and the
login needs no prompt, no pipe, and no secret on any command line:

```yaml
deploy:
  script:
    - cleura login -u "$CLEURA_USERNAME" --profile ci --cloud compliant --region sto1 --project-id $PROJECT
    - cleura gardener shoot list        # profile "ci" now carries token, cloud, region and project
```

The password can also be piped on stdin (`echo "$PW" | cleura login -u johndoe`),
and a pre-created token can be stored with
`echo "$TOKEN" | cleura login -u johndoe --with-token` (validated before storing).
Note that API tokens expire, so CI jobs should log in per run rather than reuse a
stored token.

Ready-to-copy pipeline examples (GitHub Actions, GitLab CI, plain shell) live in
[`examples/ci/`](examples/ci/).

## Configuration

Settings are resolved with the precedence **flags > environment > profile > defaults**.
Within each level an explicit API URL (`--api-url`, `CLEURA_API_URL`, `api_url`) wins
over a named cloud (`--cloud`, `CLEURA_CLOUD`, `cloud`). `cleura login` records the
endpoint it logged in to in the profile, together with the username and token.

Working with more than one cloud is a matter of profiles:

```sh
cleura login --profile compliant --cloud compliant   # separate profile for the compliant cloud
cleura --profile compliant whoami                    # one-off profile selection
cleura config use-profile compliant                  # switch the current profile
cleura config list-profiles
cleura config delete-profile compliant
```

When in doubt about what a command will actually use, `cleura config view` shows the
effective settings and the source of each value (flag, `$CLEURA_*` variable, profile,
or default), and warns when an environment variable shadows a stored profile value.

`CLEURA_PROFILE` selects a profile for scripted use without touching the config file.

The config file lives at `$CLEURA_CONFIG`, or `$XDG_CONFIG_HOME/cleura/config.yaml`,
or `~/.config/cleura/config.yaml`:

```yaml
current_profile: default
profiles:
  default:
    cloud: public            # public | compliant
    username: johndoe
    token: "..."             # stored by "cleura login"
    region: sto1             # optional defaults for project-scoped commands
    project_id: a1b2c3
  acme:                      # a private cloud sets both: the URL for the endpoint,
    cloud: acme              # the cloud name for API path parameters
    api_url: https://rest.cloud.acme.example
    username: johndoe
```

> **The config file format is internal, not an API.** It is shown here so you
> know what is stored where; its schema may change between releases. Programs
> and scripts must not parse it — use the CLI instead (`cleura config view -o json`
> for inspection). A dedicated machine-readable credential command for tool
> integrations (e.g. the Terraform provider) is planned.

Environment variables (shared with the
[Terraform provider](https://github.com/cleura/terraform-provider-cleura)):

| Variable              | Purpose                                        |
|-----------------------|------------------------------------------------|
| `CLEURA_API_URL`      | API base URL (required for private clouds)     |
| `CLEURA_API_USERNAME` | Username                                       |
| `CLEURA_API_TOKEN`    | API token (skips `cleura login`); the API requires it **together with** `CLEURA_API_USERNAME` |
| `CLEURA_API_PASSWORD` | Password for `cleura login` (CLI-only, never stored); the prompt-free CI login path |
| `CLEURA_CLOUD`        | Named cloud: `public` or `compliant`           |
| `CLEURA_REGION`       | OpenStack region (e.g. `sto1`)                 |
| `CLEURA_PROJECT_ID`   | OpenStack project ID                           |
| `CLEURA_PROFILE`      | Profile to use                                 |
| `CLEURA_CONFIG`       | Config file path override                      |

Global flags: `--profile`, `--cloud`, `--api-url`, `--region`, `--project-id`,
`--output/-o` (table, json, yaml), `--quiet/-q` (suppress informational messages;
stdout carries only data, so pipes stay clean), `--debug` (log HTTP exchanges to
stderr with credentials redacted).

Profile values can be changed without re-logging in:

```sh
cleura config set region kna1
cleura config set project_id a1b2c3
cleura config path                     # where the config file lives
```

## Shell completion

Completion scripts come built in (including profile names and flag values):

```sh
source <(cleura completion zsh)        # or bash, fish, powershell
```

Add that line to your shell profile to make it permanent.

## Development

```sh
make build      # build ./cleura
make test
make vet
```
