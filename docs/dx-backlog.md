# Developer Experience Backlog

Living tracker for CLI/SDK quality work. Regenerated 2026-07-08 (the original was
lost in a history rewrite); completed work is compressed to a summary, open items
are current. Tick items as they land.

> The 2026-07-09 full-CLI review (8 lenses, live-tested) has its own tracked file:
> [full-review-2026-07-09.md](full-review-2026-07-09.md) — 11 verified bugs + flag/
> placement + profile/login DX, organized into batches A–D.

## Done (compressed history)

- **DX audit, 2026-07-06** (7 dimensions, 52 adversarially verified findings):
  all bug and week-one-UX batches and the pure-code polish implemented — login
  prompt/terminal fixes, credential-pair validation, `config view` with per-value
  source provenance and env-shadow warnings, request timeouts, User-Agent,
  `--debug` (redacted), signal handling, misuse-error UX, early `-o` validation,
  stdout=data / stderr=info + `--quiet`, dynamic completions, help examples,
  `config set`/`path`, atomic 0600 config writes, Windows config path,
  `version` subcommand, `login --token-stdin` (renamed from --with-token
  2026-07-08; login now also always switches the current profile), `CLEURA_API_PASSWORD`.
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

- [ ] `config profile set` does not validate the *value* of `cloud` (any string
  is accepted). (The silent-phantom-create and no-"created"-signal parts are
  DONE in batches A/C: empty-value on a missing profile is a no-op, and creating
  a profile now prints "Created profile X".)
- [x] Nil profile entry (`profiles:\n  x:`): a present-but-nil key now counts as
  an existing empty profile everywhere (Resolve, config profile use/list). DONE
  batches A/C.
- [ ] `config view -o json` leaks table presentation strings: token value is
  `"(set)"`/`"(not set)"` while other unset values are `""` — add a boolean
  field, keep values machine-stable.
- [ ] GitLab example gaps: `GH_READ_TOKEN` missing from the variables list;
  protected-branch and masked-value constraints unexplained.
- [ ] `any-ci.sh` writes the token-bearing config into `$PWD`; a killed job
  leaves live credentials in the build directory.
- [ ] Token concurrency in CI unverified: do parallel logins on one service
  account invalidate each other's tokens?
- [x] SDK README: v0.x versioning policy stated; fetch instructions added
  (client-go is now a PUBLIC repo, so plain `go get` — no GOPRIVATE needed). DONE.
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

1. [x] Record `token_stored_at` in the profile at login (cheap now, impossible
   retroactively; enables staleness diagnosis without a network call).
   *(Done 2026-07-08 — also surfaced in `config view` and the 401/403 hint,
   which now says "the token was stored 23 hours ago".)*
2. [x] Add a `version: 1` field to the config schema before any other new fields.
   *(Done 2026-07-08 — stamped on Save; files from a newer schema are rejected
   with upgrade guidance instead of being silently rewritten.)*
3. [x] `cleura config get-credentials`: versioned envelope
   {version, profile, cloud, endpoint, username, token, region, project_id,
   token_stored_at} — the provider's consumption contract. JSON-only output,
   exit 0 = credentials / exit 2 + JSON error = fall through / other = broken;
   optional --validate. Documented in README "Tool integration".
   *(Done 2026-07-08 with envelope/exit-code regression tests.)*
4. [x] Provider-side work: credential source "cli" (precedence HCL > env > CLI),
   `profile`/`use_cli` attributes, and migration onto cleura-client-go — DONE on
   branch `feat/shared-client-and-cli-auth` (provider repo), adversarially
   reviewed. **Pending: push/PR/merge + a provider release.** Not yet on the
   provider's main.

Decision recorded: `internal/config` stays private; the subprocess is the boundary.

## Deferred cleanups

- [ ] shootAction closures → store the generated raw method values (identical
  signatures) + one `io.ReadAll` in `newShootActionCommand`.
- [x] Move the UPGRADE cloud-profile fetch out of the table-render closure and
  surface upgrade info in `-o json`. DONE Batch D (view model built before
  output.Render; `upgrade_available`/`status_summary` in json/yaml).
- [ ] cleura-client-go: `CreateShootAdminKubeconfig` helper owning the raw-body
  Content-Type workaround + a shared API-error type (CLI and provider both need it).
- [x] `login --api-url` only no longer persists `cloud: public` into a
  private-cloud profile. DONE Batch A (persistedEndpoint omits the cloud when it
  is only the built-in default and an explicit URL is set).
- [ ] Retries for idempotent GETs (provider's wait-loops prove transient failures).
- [ ] Inline env-shadow warnings on authenticated commands (`config view`
  covers the read side today).
- [ ] Exit-code differentiation (usage vs auth vs API failure) for scripting.
- [ ] Endpoint pairing corners: `config profile set` can store cloud/api_url
  pairs that resolveEndpoint ignores.

## ⚖ Open decisions (pre-v1)

- [x] Command grammar — RESOLVED 2026-07-09 (Batch D): restructured to
  `config profile list|current|use|set|rename|delete` (noun-verb, matching
  `gardener shoot` / `user`). Clean break, no aliases — the old flat paths
  (`config use-profile` etc.) were removed. Breaking vs v0.2.1 → next tag is v0.3.0.
- [ ] Token storage layout: current single 0600 file (aws/az practice) vs
  config/credentials split vs OS keyring with fallback.
- [ ] Commit the OpenAPI spec snapshot for reproducible generation vs today's
  live fetch — revisit at the first regeneration that affects SDK consumers.

**Decided 2026-07-08 — SMS stays the only CLI-supported 2FA.** WebAuthn cannot be
supported CLI-side: the REST API has no assertion endpoints (auth surface is
password, SMS, and an email-hash flow). If demand appears, the path is a
browser-assisted login flow (device code / localhost redirect — the gh/az/gcloud
pattern, which would also cover passkeys and future SSO), not native CTAP2
(CGO, per-OS authenticator APIs, no passkey access). Documented workaround:
Control Panel token + `login --token-stdin`.

## Integration review — 2026-07-08 (provider ↔ CLI seam, round 2)

Three-reviewer pass over the credential seam after the provider integration
landed; 23 findings, all confirmed items fixed the same day (CLI commit
4e9dbd4, provider commit 6059b74): env-stripped subprocess (tier 3 reflects
CLI state only), --validate no longer maps 5xx to "no credentials",
unresolvable endpoints exit 2 for chain callers, mixed-pair and
endpoint-override warnings on stderr, private-cloud logins no longer persist
the default cloud, live env URLs outrank stored pairings, signal-race exit
codes preserved, unknown-guards for profile/use_cli, v0.1.0 "unknown command"
detection, stderr surfacing, exit-2 reason threading, stringOrEnv empty-string
fallthrough. Accepted residual risks (documented, not coded): credentials are
snapshotted at Configure for the provider's lifetime (token expiring mid-apply
fails at the resource call), and a relative CLEURA_CONFIG resolves per-process
(README says use absolute paths).

## API-side asks — bundle into one ticket for the REST team

- [ ] admin-kubeconfig responds `Content-Type: text/html` for a YAML body (spec
  says `text/yaml`), so generated typed accessors can never work — consumers
  must read `resp.Body` raw until fixed.
- [ ] Long-lived, scoped service tokens for CI: tokens expire within ~a day,
  forcing password-per-job logins, which force 2FA-less service accounts. Also
  return token expiry in the API so tools can diagnose staleness locally.
