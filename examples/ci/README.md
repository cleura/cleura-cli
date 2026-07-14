# Using cleura in CI/CD

Copy-paste starting points for authenticating the `cleura` CLI in pipelines:

- [`github-actions.yml`](github-actions.yml) — GitHub Actions workflow
- [`gitlab-ci.yml`](gitlab-ci.yml) — GitLab CI job
- [`any-ci.sh`](any-ci.sh) — plain shell for any other runner

## The login pattern

Set the password as a **masked/protected secret** in your CI system and export it
as `CLEURA_API_PASSWORD`. `cleura login` reads it from the environment — no prompt,
no pipes, no secret on any command line, nothing for a debug trace to expand:

```sh
cleura login -u "$CLEURA_USERNAME" --profile ci --cloud public --region sto1 --project-id "$CLEURA_PROJECT_ID"
cleura gardener shoot list          # the "ci" profile now carries token, cloud, region, project
...
cleura logout --profile ci          # always run this: it revokes the job's token
```

## Required variables

| Variable              | Secret? | Purpose                                          |
|-----------------------|---------|--------------------------------------------------|
| `CLEURA_USERNAME`     | no      | Account username — a CI variable you choose, passed via `-u` (the CLI does not read it directly) |
| `CLEURA_API_PASSWORD` | **yes** | Password, read by `cleura login` (never stored)   |
| `CLEURA_PROJECT_ID`   | no      | OpenStack project for project-scoped commands     |

## Things to know

- **The account must not have SMS two-factor authentication** — there is no phone in a
  runner. Use a dedicated service account with a strong password, scoped to what the
  pipeline needs.
- **Log in per job, log out after.** Cleura API tokens are short-lived, so caching a
  token between jobs does not work; minting one per job is also the security-friendly
  lifecycle. Run `cleura logout` in an always-executed cleanup step to revoke it early.
- **Alternative: pre-created token.** If you obtain a token some other way, skip the
  password entirely: set `CLEURA_API_USERNAME` + `CLEURA_API_TOKEN` as variables and run
  commands directly (no login step), or store it in a profile with
  `echo "$TOKEN" | cleura login -u "$CLEURA_USERNAME" --token-stdin` (validated first).
  Mind the expiry.
- **Shared/shell runners:** parallel jobs sharing a HOME also share
  `~/.config/cleura/config.yaml`. Isolate with `export CLEURA_CONFIG="$PWD/cleura-config.yaml"`
  (container-based runners do not need this).
- **Quiet logs:** add `-q` to suppress informational messages; success is exit code 0.
  Add `--debug` when troubleshooting — HTTP exchanges are logged with credentials redacted.
