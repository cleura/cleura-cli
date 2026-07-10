# cleura-cli

`cleura` — the command-line interface for [Cleura Cloud](https://cleura.com/).

Built on the generated [cleura-client-go](https://github.com/cleura/cleura-client-go)
API client, commands are added incrementally as the API surface matures.

## Install

This repository is private, so Go must fetch it directly from GitHub instead
of the public module proxy. One-time setup, then install:

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

cleura user list                                             # account users with their privileges
cleura user get johndoe                                      # one user, full privilege breakdown
cleura gardener shoot list --region sto1 --project-id <id>   # your Kubernetes clusters
                                                             # (full Gardener tour below)

cleura logout             # revoke and remove the stored token (do this last)
```

`--region` and `--project-id` can be stored in the profile (pass them to
`cleura login`) or set via `CLEURA_REGION` / `CLEURA_PROJECT_ID`.

**Two-factor authentication:** SMS is the only 2FA method the CLI supports
(the API offers no way to complete a WebAuthn ceremony outside the browser).
Accounts using WebAuthn or passkeys can log in to the Control Panel, create an
API token there, and store it with
`echo "$TOKEN" | cleura login -u <username> --token-stdin`.

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
`echo "$TOKEN" | cleura login -u johndoe --token-stdin` (validated before storing).
Note that API tokens expire, so CI jobs should log in per run rather than reuse a
stored token.

Ready-to-copy pipeline examples (GitHub Actions, GitLab CI, plain shell) live in
[`examples/ci/`](examples/ci/).

## Gardener Kubernetes clusters

`cleura gardener` manages Cleura's Gardener-based Kubernetes. Most commands need
a **region and project** — pass `--region`/`--project-id`, set
`CLEURA_REGION`/`CLEURA_PROJECT_ID`, or store them in the profile at login.
(Cloud profiles and `shoot check-name` are cloud-wide and need only a cloud.)

Discover what's available (cloud-wide — no region/project needed):

```sh
cleura gardener cloud-profile list                 # machine types, Kubernetes versions, regions
cleura gardener cloud-profile show cleuracloud      # one profile in detail (supported vs deprecated versions, with expiry)
cleura gardener shoot check-name my-cluster         # is a shoot name free? (scripts: --exit-code → 0 free / 2 taken)
cleura gardener project bootstrap                   # one-time: enable Gardener for the project
```

Inspect and access clusters:

```sh
cleura gardener shoot get prod                                # full detail: worker groups, maintenance, HA, networking, status
cleura gardener worker-group list prod                        # a shoot's node pools
cleura gardener shoot kubeconfig prod > prod.kubeconfig       # time-limited admin kubeconfig
cleura gardener shoot ssh-key prod -f ~/.ssh/prod-nodes.pem   # node SSH private key (written 0600)
```

Day-2 operations:

```sh
cleura gardener shoot hibernate prod    # scale down to save cost (reversible with wake)
cleura gardener shoot wake prod
cleura gardener shoot reconcile prod    # run the reconcile loop now
cleura gardener shoot maintain prod     # run the maintenance window now
cleura gardener shoot retry prod        # retry the last failed operation
cleura gardener shoot enable-ha prod --yes                    # highly-available control plane (irreversible; --yes for CI)
cleura gardener shoot ca rotate prod --stage prepare          # rotate the cluster CA (two-phase: prepare, then complete)
cleura gardener shoot ca status prod                          # rotation stage + the next action to take
```

Observability (needs a running cluster):

```sh
cleura gardener shoot monitoring credentials prod             # Prometheus/Plutono logins (passwords masked; --show-secrets to reveal)
cleura gardener shoot monitoring nodes prod wg-primary        # per-node CPU/memory/pods for a worker group
cleura gardener shoot monitoring worker-group prod wg-primary # worker-group aggregate metrics
```

Destructive operations (`enable-ha`, `ca rotate`, ...) ask for confirmation and
refuse on a non-interactive terminal — pass `--yes` in CI. Every read command
supports `-o json`/`-o yaml`.

## Configuration

Settings are resolved with the precedence **flags > environment > profile > defaults**.
Within each level an explicit API URL (`--api-url`, `CLEURA_API_URL`, `api_url`) wins
over a named cloud (`--cloud`, `CLEURA_CLOUD`, `cloud`). `cleura login` records the
endpoint it logged in to in the profile, together with the username and token.

Working with more than one cloud is a matter of profiles:

```sh
cleura login --profile compliant --cloud compliant   # log in to a separate profile — it becomes current
cleura --profile production whoami                   # one-off use of another profile
cleura config profile use production                 # switch the current profile without logging in
cleura config profile current                        # which profile am I on?
cleura config profile list
cleura config profile rename production prod         # rename (keeps the token; current_profile follows)
cleura config profile delete compliant
```

Logging in always makes that profile the current one (the `az login`/`gcloud`
convention); `config profile use` switches between already-logged-in profiles.

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
    cloud: public            # public, compliant, or a private cloud name (see acme)
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
> and scripts must not parse it — use `cleura config get-credentials` (see
> [Tool integration](#tool-integration)), the stable JSON contract. (`cleura
> config view` is a human diagnostic; its output shape is not a contract.)

Environment variables (shared with the
[Terraform provider](https://github.com/cleura/terraform-provider-cleura)):

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

Global flags (on every command): `--profile`, `--cloud`, `--api-url`,
`--quiet/-q` (suppress informational messages; stdout carries only data, so pipes
stay clean), `--debug` (log HTTP exchanges to stderr with credentials redacted).

Scoped flags, offered only where they apply: `--output/-o` (table, json, yaml) on
commands that render output; `--region` and `--project-id` on `gardener` commands
(and on `cleura login`, which stores them in the profile). Environment variables
`CLEURA_REGION`/`CLEURA_PROJECT_ID` still apply everywhere they are relevant.

Profile values can be changed without re-logging in:

```sh
cleura config profile set region kna1
cleura config profile set project_id a1b2c3
cleura config path                     # where the config file lives
```

## Tool integration

`cleura config get-credentials` is the stable interface for tools that
authenticate via the CLI (the Terraform provider's planned `cleura` credential
source, scripts, anything else). It prints the effective credentials — resolved
with the usual precedence — as one JSON object:

```sh
$ cleura config get-credentials
{
  "version": 1,
  "profile": "work",
  "cloud": "compliant",
  "endpoint": "https://rest.compliant.cleura.cloud",
  "username": "svc",
  "token": "...",
  "region": "sto1",
  "project_id": "p-1",
  "token_stored_at": "2026-07-08T08:00:00Z"
}

$ cleura config get-credentials | jq -r .token   # e.g. for curl scripts
```

Exit codes are part of the contract: `0` credentials printed; `2` no usable
credentials for the selected profile (a JSON `{"error": ...}` object is printed
on stdout, so credential chains can fall through to their next source); any
other exit is a malfunction. `--validate` verifies the token against the API
first. Compatibility: fields are only added — never renamed or removed — while
`version` is `1`; a breaking change bumps it, and consumers must check it.

The `region`, `project_id`, `cloud` and `token_stored_at` fields are omitted
when unset — in particular `cloud` is absent for a private cloud configured
with only an `api_url` and no cloud name.

`config get-credentials` is the only versioned, stable JSON contract. Every
other command's `-o json`/`-o yaml` output is for convenience: it mirrors the
unversioned API plus a few CLI-computed fields and may change between releases,
so don't build automation that depends on its exact shape.

## Command reference

Every command and flag is documented in [`docs/reference/`](docs/reference/cleura.md),
generated from the CLI's own help text (`make docs` regenerates it — the reference
cannot drift from the binary).

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
