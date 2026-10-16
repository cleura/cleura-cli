# cleura-cli

`cleura` is the command-line interface for [Cleura Cloud](https://cleura.com/).
Together, `cleura-cli` and `cleura-openstackclient` provide CLI access to all functionality available via the [Cleura Cloud Management Panel](https://cleura.cloud).

It is built on the [cleura-client-go](https://github.com/cleura/cleura-client-go) API client, with commands added incrementally as the API surface matures.

> **Warning**
>
> **Early (`0.x`) development.**
> The CLI has been tested against live Cleura environments and is functional, but command coverage is still limited.
> Also, its commands and flags may change between `0.x` releases.

## Currently supported

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
Every release comes with a `checksums.txt` file, which is signed with [cosign](https://github.com/sigstore/cosign) (see [Verifying releases](release-verification.md)).

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

For detailed command documentation, see the linked guides:

- [Gardener Kubernetes clusters](gardener.md)
- [Cleura account users](users.md)
- [OpenStack identity](openstack.md)
- [Configuration](configuration.md)

## Two-factor authentication

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

Ready-to-copy pipeline examples (GitHub Actions, GitLab CI, plain shell) are in the [`examples/ci/`](https://github.com/cleura/cleura-cli/tree/main/examples/ci) directory.
