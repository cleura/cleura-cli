# cleura-cli

`cleura` is the command-line interface for [Cleura Cloud](https://cleura.com/).
Together, `cleura-cli` and `cleura-openstackclient` provide CLI access to all functionality available via the [Cleura Cloud Management Panel](https://cleura.cloud).

It is built on the [cleura-client-go](https://github.com/cleura/cleura-client-go) API client, with commands added incrementally as the API surface matures.

> **Warning**
>
> **Early (`0.x`) development.**
> The CLI has been tested against live Cleura environments and is functional, but command coverage is still limited.
> Also, its commands and flags may change between `0.x` releases.

**Currently supported:**

- **Auth:** `cleura login` / `logout` / `whoami` (SMS 2FA; token or password for CI)
- **Account users:** `cleura user` (view users, manage their privileges and passwords)
- **Configuration:** `cleura config` (`get-credentials`, for tooling like the Terraform provider)
- **Gardener Kubernetes:** `cleura gardener` (shoots, worker groups, cloud profiles, kubeconfig/SSH access, day-2 ops, CA rotation, monitoring)
- **OpenStack identity:** `cleura openstack` (domains, projects, users, role assignments)

## Installation

### Install script (Linux/macOS)

It downloads the latest release binary, verifies it, and installs it.

```sh
curl -fsSL https://raw.githubusercontent.com/cleura/cleura-cli/main/install.sh | sh
```

The script uses `sudo`, if necessary, and works with `/usr/local/bin`.
Override with `BINDIR=$HOME/.local/bin`, or pin a version with `CLEURA_VERSION=v0.7.0`.

### Prebuilt binary

Download an archive suitable for your OS/architecture from the [project's releases page](https://github.com/cleura/cleura-cli/releases/latest).
Do not forget to put `cleura` on your `PATH`.
Every release comes with a `checksums.txt` file, which is signed with [cosign](https://github.com/sigstore/cosign) (see [Verifying releases](#verifying-releases)).

### Using Go (1.25+)

```sh
go install github.com/cleura/cleura-cli/cmd/cleura@latest
```

If `go install` fails with a C-toolchain error (e.g., a missing `stdlib.h`), prefix the command with `CGO_ENABLED=0`.

### Via Homebrew

```sh
brew install cleura/tap/cleura
```

### From a checkout

Use `make install`.

## Verifying releases

Every release ships with a `checksums.txt` file, which has the SHA-256 sum of each available archive.
This file is signed with [cosign](https://github.com/sigstore/cosign) using keyless [Sigstore](https://www.sigstore.dev/) signing.
The file is accompanied by the signature, `checksums.txt.sig`, and the signing certificate, `checksums.txt.pem`.

The install script already checks the checksum of what it downloads.
The steps below go one step further and prove that `checksums.txt` itself was produced by this repository's release workflow.
In other words, it proves it wasn't tampered with or re-signed by someone else.

To follow along, you need to have [`cosign`](https://github.com/sigstore/cosign) installed.

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
# (run from the directory where you downloaded the archive).
sha256sum --ignore-missing -c checksums.txt # macOS: shasum -a 256 --ignore-missing -c checksums.txt
```

On success, `cosign verify-blob` returns `Verified OK`.
To verify any release without editing the tag, swap `--certificate-identity` for `--certificate-identity-regexp '^https://github\.com/cleura/cleura-cli/\.github/workflows/release\.yml@refs/tags/v'`.

## Getting started

```sh
cleura login              # prompts for username/password (SMS 2FA supported)
cleura whoami             # shows the authenticated user
cleura whoami -o json     # returns machine-readable output

cleura user list                                             # lists account users with their privileges
cleura user get johndoe                                      # returns full privilege breakdown of specified user
cleura user create johndoe --email john.doe@example.org      # creates a user (reads password from a no-echo prompt)
cleura gardener shoot list --region sto1 --project-id <id>   # lists Kubernetes clusters
cleura openstack project list                                # lists OpenStack projects, users, and roles

cleura logout             # revokes and removes the stored token
```

The values for `--region` and `--project-id` can be stored in the profile (pass them to `cleura login`), or set via `CLEURA_REGION` and `CLEURA_PROJECT_ID`.

### Two-factor authentication

SMS is the only 2FA method the CLI supports (the API offers no way to go through WebAuthn outside the browser).
Accounts using WebAuthn or passkeys can log in to the Control Panel, create an API token there, and store it with `echo "$TOKEN" | cleura login -u <username> --token-stdin`.

Also, non-interactive logins for CI are supported (single-factor accounts only; SMS 2FA requires an interactive terminal).
Set `CLEURA_API_PASSWORD` as a masked CI variable; the login does not need a prompt, a pipe, or a secret:

```yaml
deploy:
  script:
    - cleura login -u "$CLEURA_USERNAME" --profile ci --cloud public --region sto2 --project-id $PROJECT
    - cleura gardener shoot list  # profile "ci" now carries token, cloud, region and project
```

The password can also be piped on stdin (`echo "$PW" | cleura login -u johndoe`), and a pre-created token can be stored with `echo "$TOKEN" | cleura login -u johndoe --token-stdin` (validated before storing).
Note that API tokens expire, so CI jobs should log in per run rather than reuse a stored token.

Ready-to-copy pipeline examples (GitHub Actions, GitLab CI, plain shell) are in the [`examples/ci/`](examples/ci/) directory.

## Gardener Kubernetes clusters

The `cleura gardener` command manages Cleura's Gardener-based Kubernetes.
Most commands need a **region** and **project**.
Pass `--region`/`--project-id`, set `CLEURA_REGION`/`CLEURA_PROJECT_ID`, or store them in the profile at login.
(Cloud profiles and `shoot check-name` are cloud-wide and require only a cloud ID.)

Discover what's available cloud-wide — no region/project is required:

```sh
cleura gardener cloud-profile list                  # lists machine types, Kubernetes versions, and regions
cleura gardener cloud-profile show cleuracloud      # inspects one profile in detail (supported vs deprecated versions, with expiry)
cleura gardener shoot check-name my-cluster         # check whether a shoot name is free (scripts: --exit-code → 0 free / 2 taken)
cleura gardener project bootstrap                   # a one-time operation; it enables Gardener for the project
```

Inspect and access clusters:

```sh
cleura gardener shoot get prod                                # returns full details: worker groups, maintenance, HA, networking, status
cleura gardener shoot worker-group list prod                  # lists node pools of a shoot
cleura gardener shoot kubeconfig prod > prod.kubeconfig       # creates a time-limited admin kubeconfig
cleura gardener shoot ssh-key prod -f ~/.ssh/prod-nodes.pem   # creates a node SSH private key
```

Day-2 operations:

```sh
cleura gardener shoot hibernate prod    # scales down to save cost (reversible with wake)
cleura gardener shoot wake prod         # brings the shoot back online
cleura gardener shoot reconcile prod    # runs the reconcile loop now
cleura gardener shoot maintain prod     # runs the maintenance window now
cleura gardener shoot retry prod        # retries the last failed operation
cleura gardener shoot enable-ha prod --yes                    # creates a highly-available control plane (irreversible; --yes for CI)
cleura gardener shoot ca rotate prod --stage prepare          # rotates the cluster CA (two-phase: prepare, then complete)
cleura gardener shoot ca status prod                          # returns the rotation stage and the next action in the queue
```

Observability (needs a running cluster):

```sh
cleura gardener shoot monitoring credentials prod             # returns Prometheus/Plutono logins (passwords are masked, use --show-secrets to reveal)
cleura gardener shoot monitoring nodes prod wg-primary        # returns per-node CPU, memory, and pods of a worker group
cleura gardener shoot monitoring worker-group prod wg-primary # returns worker-group aggregate metrics
```

Non-reversible operations, like `enable-ha` and `ca rotate`, ask for confirmation and do not work non-interactively in the terminal (in a CI setting, pass the `--yes` parameter).

Every read command outputs in JSON (`-o json`) or YAML (`-o yaml`).

## Cleura account users

`cleura user` manages the users of your **Cleura account**, that is, the identities that log in to the Control Panel and this CLI.
These are distinct from the OpenStack (Keystone) users that `cleura openstack user` manages.
The command requires `users` privileges or account-admin rights; to see your own account, use `cleura whoami`.

```sh
cleura user list                          # lists every user, with a privilege summary and 2FA status
cleura user get johndoe                   # returns full breakdown of a single user (identified by username or numeric ID)
```

### Create a user

The required password is never specified via a flag.
Instead, it is read from a no-echo prompt or from stdin, when piped:

```sh
cleura user create johndoe --email john.doe@example.org --first-name John --last-name Doe
printf '%s' "$PASSWORD" | cleura user create ci-bot --email ci@example.org    # non-interactive (CI)
```

The `edit` command sends only the flags you pass, so an unset field is never overwritten:

```sh
cleura user edit johndoe --email new.address@example.org   # may need email verification
cleura user edit johndoe --first-name Jonathan
cleura user edit johndoe --set-password                    # prompts for a new password
cleura user edit ci-bot --ip-restriction 203.0.113.0/24    # replaces the whole set
cleura user edit ci-bot --clear-ip-restrictions
```

Deleting a user is irreversible, so the `delete` command asks for confirmation.
On a non-interactive terminal, the `delete` command needs the `--yes` parameter.
Note that the account you are currently logged in cannot be deleted here:

```sh
cleura user delete johndoe
cleura user delete 4763 --yes
```

### Privileges

A new user starts with no access.
Privileges may be about account-wide access to an *area*, or about access to specific OpenStack *projects*.

```console
$ cleura user privilege list johndoe
AREA        LEVEL     SCOPE
invoice     read      account-wide
openstack   project   team-alpha:read, team-beta:full
```

Grant account-wide access to an area (`account`, `ai-gateway`, `application`, `invoice`, `monitoring`, `openstack`, `users`) with the `--area` parameter.
Give access to a single project with the `--project-id` parameter.
In any case, the access level is either `full` or `read`:

```sh
cleura user privilege set johndoe --area invoice --level read
cleura user privilege set johndoe --project-id team-alpha --level read
cleura user privilege set johndoe --project-id team-beta --level full
```

Each project grant leaves the others alone, so repeat the command to add more.
The `--area` parameter replaces whatever that area had, including its per-project grants.

Remove access the same way:

```sh
cleura user privilege unset johndoe --project-id team-alpha   # one project
cleura user privilege unset johndoe --area invoice            # the whole area
```

The `--project-id` parameter takes a project ID or a name.
**Project names are not unique across regions**, so a name matching more than one project is not accepted:

```plain
Error: "team-alpha" matches 2 projects; qualify it as region/name or pass the ID:
  kna1/team-alpha    bbbb2222…
  sto2/team-alpha    aaaa1111…
```

Use something like `sto2/team-alpha` or, better yet, the corresponding ID.
For a project that is not in your own project list (the API offers no account-wide project listing), pass its ID together with  `--domain-id`.

#### Important notes

The API has no privilege sub-resource: a privilege change is a read, merge, and write of the whole user.
Two people changing privileges at the same moment can overwrite each other, and the last write prevails.

Account-admin rights cannot be changed through the API.
The `ADMIN` column is set to read-only for every client.
The create and edit request bodies have no such field.
Change the `ADMIN` column in the Control Panel.

## OpenStack identity (projects, users, roles)

The `cleura openstack` command manages OpenStack (Keystone) identity — domains, projects, users, and their role assignments.
OpenStack users are **distinct from Cleura account users** (`cleura user`): they authenticate against OpenStack itself.

> **Note**
>
> **The `cleura openstack` command does not double as a general-purpose OpenStack client, and is not meant to replace [`cleura-openstackclient`](https://github.com/cleura/cleura-openstackclient).**
> It deliberately covers only Keystone *identity* provisioning (projects, users, roles, and role assignments), i.e., what is covered by `cleura login` and the Cleura control plane.
> For compute, networking, storage, images, and the rest of the OpenStack API, use `cleura-openstackclient`.
> That hand-off is why the `cleura openstack` command surface is intentionally small.

Almost every command is scoped to a **domain** and needs the `--domain-id` parameter.
Cleura gives an account a domain per region, so you normally have several domains.
`cleura-cli` (trivially) auto-selects a domain only when the account has exactly one; otherwise, the `--domain-id` parameter is required.

The `domain list` and `project list` subcommands are an exception, in that they do not require the `--domain-id` parameter.
Use those subcommands to find the IDs you need:

```sh
cleura openstack domain list                       # returns domains, IDs, and their region/area
cleura openstack project list                      # returns projects you can access, grouped by region
```

List subcommands within a domain:

```sh
cleura openstack user list --domain-id <domain-id>    # returns OpenStack users in the domain
cleura openstack role list --domain-id <domain-id>    # returns assignable roles (member, load-balancer_member, etc.)
```

Project subcommands (the API has no delete functionality; use `--disable` when you wish to delete):

```sh
cleura openstack project create my-project --domain-id <domain-id> --description "team sandbox"   # prints the new project and ID
cleura openstack project edit <project-id> --domain-id <domain-id> --name new-name                # renames a project
cleura openstack project edit <project-id> --domain-id <domain-id> --disable                      # pseudo-deletes (turns off)
```

When using the `openstack user` subcommand, the new user's password is read from a no-echo prompt or piped stdin — never from a parameter:

```sh
cleura openstack user create alice --domain-id <domain-id>                          # prompts for the password (no echo)
printf '%s' "$PASSWORD" | cleura openstack user create svc --domain-id <domain-id>  # non-interactive (CI)
cleura openstack user delete alice --domain-id <domain-id> --yes                    # deletes an OpenStack user by name or ID (--yes for CI)
```

The `role assignment` subcommand grants a user roles on a project (a role assignment is the user + project + role binding):

```sh
cleura openstack role assignment create --user alice --role member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment create --user alice --role member,load-balancer_member --project-id <project-id> --domain-id <domain-id>
cleura openstack role assignment list   --user alice --domain-id <domain-id>    # lists projects alice can access
cleura openstack role assignment delete --user alice --role member --project-id <project-id> --domain-id <domain-id>
```

The `--user` parameter accepts a name or an ID, while the `--role` parameter takes role names.
The `--project-id` and `--domain-id` parameters accept IDs, the ones we get from the `project list` and `domain list` subcommands.

The `user delete` subcommand asks for confirmation.
It refuses to work on a non-interactive terminal — unless you add the `--yes` switch.

Every read command supports output in JSON (`-o json`) or YAML (`-o yaml`).

## Configuration

Settings are applied in this order: **flags > environment > profile > defaults**.
Within each level, an explicit API URL (`--api-url`, `CLEURA_API_URL`, `api_url`) wins over a named cloud (`--cloud`, `CLEURA_CLOUD`, `cloud`).
The `cleura login` command records the endpoint it logged into in the profile, together with the username and a token.

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

Environment variables (shared with the [Terraform provider](https://github.com/cleura/terraform-provider-cleura)):

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

Global flags (on every command):
`--profile`, `--cloud`, `--api-url`, `--quiet/-q` (suppress informational messages; stdout carries only data, so pipes stay clean), `--debug` (log HTTP exchanges to stderr with credentials redacted).

Scoped flags, offered only where they apply:
`--output/-o` (table, json, yaml) on commands that render output;
`--region` and `--project-id` on `gardener` commands (and on `cleura login`, which stores them in the profile);
`--domain-id` on most `cleura openstack` commands (normally required — an account usually has several domains, one per region).
Environment variables `CLEURA_REGION`/`CLEURA_PROJECT_ID` still apply everywhere they are relevant.

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

Every command and flag is documented in [`docs/reference/`](docs/reference/cleura.md), generated from the CLI's own help text (`make docs` regenerates it — the reference cannot drift from the binary).

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
