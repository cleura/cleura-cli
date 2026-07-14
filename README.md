# cleura-cli

`cleura` — the command-line interface for [Cleura Cloud](https://cleura.com/).

It gives you one scriptable way to authenticate to Cleura and manage your cloud —
from Gardener Kubernetes clusters to OpenStack identity — on your machine or in CI.

Built on the generated [cleura-client-go](https://github.com/cleura/cleura-client-go)
API client, commands are added incrementally as the API surface matures.

> [!WARNING]
> **Early (`0.x`) development.** The CLI has been tested against live Cleura
> environments and is functional, but command coverage is still limited and its
> commands and flags may change between `0.x` releases.

**Currently supported:**

- **Auth** — `cleura login` / `logout` / `whoami` (SMS 2FA; token or password for CI)
- **Account users** — `cleura user` (view users and their privileges)
- **Configuration** — `cleura config` (profiles; `get-credentials` for tooling like the Terraform provider)
- **Gardener Kubernetes** — `cleura gardener` (shoots, worker groups, cloud profiles, kubeconfig/SSH access, day-2 ops, CA rotation, monitoring)
- **OpenStack identity** — `cleura openstack` (domains, projects, users, role assignments)

## Install

**Install script** (Linux/macOS) — downloads the latest release binary, verifies
its checksum, and installs it:

```sh
curl -fsSL https://raw.githubusercontent.com/cleura/cleura-cli/main/install.sh | sh
```

Installs to `/usr/local/bin` (uses `sudo` if needed); override with
`BINDIR=$HOME/.local/bin`, or pin a version with `CLEURA_VERSION=v0.7.0`.

**Prebuilt binary** — or download an archive for your OS/architecture from the
[latest release](https://github.com/cleura/cleura-cli/releases/latest) and put
`cleura` on your `PATH`. Every release ships a `checksums.txt` signed with
[cosign](https://github.com/sigstore/cosign) — see [Verifying releases](#verifying-releases).

**With Go** (1.25+):

```sh
go install github.com/cleura/cleura-cli/cmd/cleura@latest
```

`cleura` is pure Go; if `go install` fails with a C-toolchain error (e.g. a missing
`stdlib.h`) on a minimal system, prefix the command with `CGO_ENABLED=0`.

**Homebrew** (once the tap is published):

```sh
brew install cleura/tap/cleura
```

From a checkout: `make install`.

## Verifying releases

Every release ships a `checksums.txt` (SHA-256 of each archive) signed with
[cosign](https://github.com/sigstore/cosign) using keyless
[Sigstore](https://www.sigstore.dev/) signing, alongside the signature
`checksums.txt.sig` and the signing certificate `checksums.txt.pem`. The install
script already checks the checksum of what it downloads; the steps below go one
further and prove that `checksums.txt` itself was produced by this repository's
release workflow (not tampered with or re-signed by someone else).

Requires [`cosign`](https://github.com/sigstore/cosign) (`brew install cosign`):

```sh
VERSION=v0.7.0   # the release you're verifying
base="https://github.com/cleura/cleura-cli/releases/download/$VERSION"

# 1. Fetch the checksum file, its signature, and the signing certificate.
curl -fsSLO "$base/checksums.txt"
curl -fsSLO "$base/checksums.txt.sig"
curl -fsSLO "$base/checksums.txt.pem"

# 2. Verify checksums.txt was signed by this repo's release workflow.
cosign verify-blob checksums.txt \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  --certificate-identity "https://github.com/cleura/cleura-cli/.github/workflows/release.yml@refs/tags/$VERSION" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"

# 3. checksums.txt is now trusted — verify your downloaded archive against it
#    (run from the directory where you downloaded the archive).
sha256sum --ignore-missing -c checksums.txt        # macOS: shasum -a 256 --ignore-missing -c checksums.txt
```

`cosign verify-blob` prints `Verified OK` on success. To verify any release without
editing the tag, swap `--certificate-identity` for
`--certificate-identity-regexp '^https://github\.com/cleura/cleura-cli/\.github/workflows/release\.yml@refs/tags/v'`.

## Getting started

```sh
cleura login              # prompts for username/password (SMS 2FA supported)
cleura whoami             # show the authenticated user
cleura whoami -o json     # machine-readable output

cleura user list                                             # account users with their privileges
cleura user get johndoe                                      # one user, full privilege breakdown
cleura gardener shoot list --region sto1 --project-id <id>   # your Kubernetes clusters
cleura openstack project list                                # OpenStack projects, users, roles
                                                             # (full tours below)

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
    - cleura login -u "$CLEURA_USERNAME" --profile ci --cloud public --region sto2 --project-id $PROJECT
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
cleura gardener shoot worker-group list prod                  # a shoot's node pools
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

## OpenStack identity (projects, users, roles)

`cleura openstack` manages OpenStack (Keystone) identity — domains, projects,
users, and their role assignments. OpenStack users are **distinct from Cleura
account users** (`cleura user`): they authenticate against OpenStack itself.

> [!NOTE]
> **`cleura openstack` is not a general-purpose OpenStack client, and is not meant to
> replace [`cleura-openstackclient`](https://github.com/cleura/cleura-openstackclient).**
> It deliberately covers only Keystone *identity* provisioning (projects, users,
> roles, and role assignments) — the setup that pairs with `cleura login` and the
> Cleura control plane. For compute, networking, storage, images, and the rest of the
> OpenStack API, use `cleura-openstackclient`. That hand-off is why the command
> surface here is intentionally small.

Almost every command is scoped to a **domain** and needs `--domain-id <domain-id>`.
Cleura gives an account a domain per region, so you normally have several — the CLI
auto-selects a domain only when the account has exactly one; otherwise `--domain-id` is
required. The exceptions are `domain list` and `project list`, which take no
`--domain-id` — use them to find the IDs:

```sh
cleura openstack domain list                       # domains + IDs and their region/area — find the one you want
cleura openstack project list                      # projects you can access, grouped by region
```

Everything else takes `--domain-id <domain-id>`. Lists within a domain:

```sh
cleura openstack user list --domain-id <domain-id>    # OpenStack users in the domain
cleura openstack role list --domain-id <domain-id>    # assignable roles (member, load-balancer_member, ...)
```

Projects (the API has no delete — `--disable` is the closest):

```sh
cleura openstack project create my-project --domain-id <domain-id> --description "team sandbox"   # prints the new project + ID
cleura openstack project edit <project-id> --domain-id <domain-id> --name new-name                # rename
cleura openstack project edit <project-id> --domain-id <domain-id> --disable                      # pseudo-delete (turn it off)
```

Users — the new user's password is read from a no-echo prompt or piped stdin,
never from a flag:

```sh
cleura openstack user create alice --domain-id <domain-id>                          # prompts for the password (no echo)
printf '%s' "$PASSWORD" | cleura openstack user create svc --domain-id <domain-id>  # non-interactive (CI)
cleura openstack user delete alice --domain-id <domain-id> --yes                    # by name or ID (--yes for CI)
```

Role assignments — grant a user roles on a project (a role assignment is the
user + project + role binding):

```sh
cleura openstack role assignment create --user alice --role member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment create --user alice --role member,load-balancer_member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment list   --user alice --domain-id <domain-id>        # projects alice can access + roles held
cleura openstack role assignment delete --user alice --role member --project-id <project-id> --domain-id <domain-id>
```

`--user` accepts a name or an ID and `--role` takes role names (both resolved for
you); `--project-id` and `--domain-id` are IDs (from `project list` and `domain list`).
`user delete` confirms and refuses on a non-interactive terminal unless `--yes`.
Every read command supports `-o json`/`-o yaml`.

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

Logging in always makes that profile the current one; `config profile use`
switches between already-logged-in profiles.

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
    region: sto2             # optional defaults for project-scoped commands
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
(and on `cleura login`, which stores them in the profile); `--domain-id` on most
`cleura openstack` commands (normally required — an account usually has several
domains, one per region; auto-selected only when there is exactly one).
Environment variables `CLEURA_REGION`/`CLEURA_PROJECT_ID` still apply everywhere
they are relevant.

Profile values can be changed without re-logging in:

```sh
cleura config profile set region kna1
cleura config profile set project_id a1b2c3
cleura config path                     # where the config file lives
```

## Tool integration

`cleura config get-credentials` is the stable interface for tools that
authenticate via the CLI (the Terraform provider's `cleura` credential
source, scripts, anything else). It prints the effective credentials — resolved
with the usual precedence — as one JSON object:

```sh
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
make docs       # regenerate docs/reference/ from the CLI's help text
```

## License

cleura-cli is licensed under the [Mozilla Public License 2.0](LICENSE).
