# Developer Experience Backlog

Living tracker for CLI/SDK quality work. Regenerated 2026-07-08 (the original was
lost in a history rewrite); completed work is compressed to a summary, open items
are current. Tick items as they land.

## Done (compressed history)

- **DX audit, 2026-07-06** (7 dimensions, 52 adversarially verified findings):
  all bug and week-one-UX batches and the pure-code polish implemented — login
  prompt/terminal fixes, credential-pair validation, `config view` with per-value
  source provenance and env-shadow warnings, request timeouts, User-Agent,
  `--debug` (redacted), signal handling, misuse-error UX, early `-o` validation,
  stdout=data / stderr=info + `--quiet`, dynamic completions, help examples,
  `config set`/`path`, atomic 0600 config writes, Windows config path,
  `version` subcommand, `login --with-token`, `CLEURA_API_PASSWORD`.
- **Pre-commit review, 2026-07-07** (7 angles over the full initial diff): 10
  confirmed defects fixed before the first commit — nil-profile panic,
  wrong-token logout under env override, private-cloud endpoint pairing,
  kubeconfig file permissions, YAML key mangling, piped-secret trimming,
  generate.sh pipefail, delete-profile revocation, README CI example, whoami
  formatting. Regression tests added.
- **v0.1.0 release review, 2026-07-08** (6 lenses): blockers fixed —
  `-o yaml` large-integer corruption (yaml intermediate parse instead of
  float64-producing json.Unmarshal), typo'd subcommands exiting 0 with help on
  stdout (parents now runnable + NoArgs), Go 1.26 Ctrl-C exit-130 contract,
  config-file "not an API" disclaimer, install docs (GOPRIVATE), replace
  directive dropped and cleura-client-go pinned at v0.1.0.

## Open — v0.1.x should-fix (from the release review)

- [ ] `config set` validates nothing: accepts any cloud value, and silently
  creates a new profile for a typo'd `--profile` name while `use-profile`
  refuses unknown names — make creation explicit or consistent.
- [ ] Nil profile entry (`profiles:\n  x:`): `use-profile` switches to it and
  `list-profiles` shows it, but authenticated commands say it does not exist.
  Fix in Resolve: key-present-but-nil counts as an existing empty profile.
- [ ] `config view -o json` leaks table presentation strings: token value is
  `"(set)"`/`"(not set)"` while other unset values are `""` — add a boolean
  field, keep values machine-stable.
- [ ] GitLab example gaps: `GH_READ_TOKEN` missing from the variables list;
  protected-branch and masked-value constraints unexplained.
- [ ] `any-ci.sh` writes the token-bearing config into `$PWD`; a killed job
  leaves live credentials in the build directory.
- [ ] Token concurrency in CI unverified: do parallel logins on one service
  account invalidate each other's tokens?
- [ ] SDK README: state the v0.x versioning policy for the regenerating `api`
  package; add private-module (GOPRIVATE) fetch instructions.
- [ ] generate.sh residue: `openapi_downgrade` unpinned; the `sed` transform is
  fragile line-oriented text surgery.
- [ ] Project-ID discovery (`cleura project list`): sharpest week-one hole —
  gardener commands require `--project-id` the CLI cannot look up. Needs an
  SDK include-tag decision (additive, semver-safe).
- [ ] `gardener shoot get` (detail view); document the split "CLI owns day-2,
  terraform owns lifecycle".
- [ ] Release engineering: goreleaser + CI + binaries (deliberately deferred
  from v0.1.0); private-repo binary-download recipe for the CI examples;
  release notes on tags.

## Provider-auth integration track (sequenced — design decided 2026-07-08)

The terraform provider will consume CLI credentials by **exec'ing the CLI**
(az-CLI subprocess pattern), never by parsing config.yaml (declared internal in
the README). In dependency order:

1. [ ] Record `token_stored_at` in the profile at login (cheap now, impossible
   retroactively; enables staleness diagnosis without a network call).
2. [ ] Add a `version: 1` field to the config schema before any other new fields.
3. [ ] `cleura config get-credentials -o json`: versioned envelope
   {version, profile, cloud, endpoint, username, token, region, project_id,
   token_stored_at} — the provider's consumption contract.
4. [ ] Provider-side work: credential source "cli" with precedence
   explicit > env > CLI, a `profile` attribute; bundle with migrating the
   provider off its duplicated generated client onto cleura-client-go.

Decision recorded: `internal/config` stays private; the subprocess is the boundary.

## Deferred cleanups

- [ ] shootAction closures → store the generated raw method values (identical
  signatures) + one `io.ReadAll` in `newShootActionCommand`.
- [ ] Move the UPGRADE cloud-profile fetch out of the table-render closure
  (RunE, optionally concurrent); consider upgrade info in `-o json`.
- [ ] cleura-client-go: `CreateShootAdminKubeconfig` helper owning the raw-body
  Content-Type workaround + a shared API-error type (CLI and provider both need it).
- [ ] `login --api-url` only (no cloud anywhere) persists `cloud: public` into a
  private-cloud profile; consider omitting cloud when its source is the default.
- [ ] Retries for idempotent GETs (provider's wait-loops prove transient failures).
- [ ] Inline env-shadow warnings on authenticated commands (`config view`
  covers the read side today).
- [ ] Exit-code differentiation (usage vs auth vs API failure) for scripting.
- [ ] Endpoint pairing corners: `config set` can store cloud/api_url pairs that
  resolveEndpoint ignores.

## ⚖ Open decisions (pre-v1)

- [ ] Command grammar: `config use-profile` (kubectl-style hyphenation) vs
  `config profile use` (noun-verb, matching `gardener shoot list`) — converge
  or add aliases before v1.
- [ ] Token storage layout: current single 0600 file (aws/az practice) vs
  config/credentials split vs OS keyring with fallback.
- [ ] Commit the OpenAPI spec snapshot for reproducible generation vs today's
  live fetch — revisit at the first regeneration that affects SDK consumers.

## API-side asks — bundle into one ticket for the REST team

- [ ] admin-kubeconfig responds `Content-Type: text/html` for a YAML body (spec
  says `text/yaml`), so generated typed accessors can never work — consumers
  must read `resp.Body` raw until fixed.
- [ ] Long-lived, scoped service tokens for CI: tokens expire within ~a day,
  forcing password-per-job logins, which force 2FA-less service accounts. Also
  return token expiry in the API so tools can diagnose staleness locally.
