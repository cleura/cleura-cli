# Full CLI review — 2026-07-09

Full command/subcommand/flag/DX/bug review of the cleura CLI, run across 8 lenses
(login, profiles, gardener, user, flags-placement, cicd, bug-hunt, help-docs) with
live testing against the API (profile `claudetestuser`; the gardener cluster
`hampus-tf-test` was woken/reconciled/hibernated and restored to its original
hibernated state). Every bug-class finding was adversarially verified. Companion to
[dx-backlog.md](dx-backlog.md); tick items as they land.

**Verdict:** well-built — no blockers, no security defects, no data-loss in normal
use. Optional API pointers are nil-guarded, precedence resolution is sound, exit
codes and stdout/stderr discipline are correct, `-o json/yaml` preserves large
integers, and all live-exercised commands work. Not yet at the "flawless
login/profiles" bar: 11 verified bugs (3 in the login/profile path added in recent
sessions) plus a flag surface that over-promises.

Live-verified: `gardener shoot list/kubeconfig/wake/reconcile/hibernate` (correct
confirmations, 0600 kubeconfig, clean stdout, STATUS progress display
`Processing (Reconcile n%)` → `Succeeded`, clean 404 on a bad shoot name);
`whoami`, `user list/get` (by id and username), `config view/get-credentials`;
exit-code matrix and stdout-is-data-only across all render commands.

---

## Batch A — confirmed bugs (fix first)

_All fixed 2026-07-09, with regression tests; verified through the binary and (read-only) live API._

Login / profile path (the flagship):
- [x] **HIGH — `--token-stdin` without `-u` consumes the piped token as the username**,
  then dies `stdin closed before the Token prompt: EOF`. This is the WebAuthn-account
  onboarding path our own help points to. `login.go:70-77,273-280`. Fix: in token-stdin
  mode never prompt for username; fail fast if unresolved from `-u`/`$CLEURA_API_USERNAME`.
- [x] **MEDIUM — identity-replacement region/project reset is dead code.** On a confirmed
  identity swap, `login.go:126` clears region/project but `:130-135` re-apply the old
  profile's values (still in `settings`). New identity silently keeps the old project →
  possible gardener mutation against the wrong project. Fix: re-apply region/project only
  when sourced from a flag/env this invocation (`settings.Sources.Region == "--region" ||
  "$CLEURA_REGION"`, same for project).
- [x] **MEDIUM — overwrite-confirm reads piped stdin.** A piped password triggers a
  surprising "refusing to overwrite"; a password of `y`/`yes` would auto-confirm a
  destructive replace — opposite of the documented "non-interactive refuses".
  `login.go:95-101`, `prompt.go:81-90`. Fix: when stdin is not a TTY, `confirm()` refuses
  without reading.
- [x] **MEDIUM — `use-profile` ↔ `list-profiles` contradict on a nil profile entry**
  (`profiles:\n  x:`): use-profile switches + reports success while list-profiles warns
  it "does not exist". Found by two lenses. `configcmd.go:215-227` vs `:258-260`;
  `config.go:223-224`. Fix: a present key exists everywhere; reword warning to
  "has no stored settings".

Fix soon:
- [x] **MEDIUM — `get-credentials` prints a spurious "pair may not authenticate together"
  warning on 100% of pure-env invocations** — the tool-integration path it exists for
  (env username + env token always have different source labels). `getcredentials.go:93`.
  Fix: warn only on a genuine profile+env cross-mix, not env+env or profile+profile.
- [x] **LOW — `config set <key> "" ` on a nonexistent profile creates a phantom `name: {}`**
  while claiming to remove a value. `configcmd.go:167-176`, `config.go:137-146`. Fix:
  empty value + missing profile = no-op.
- [x] **LOW — `kubeconfig --expiration` under 1s truncates to 0s** and bypasses the
  positivity guard (validates the duration, sends `int(seconds)`). `gardener.go:198-211`.
  Fix: validate the integer seconds (`< 1` → error).
- [x] **LOW — `UPGRADE "?"` conflates "couldn't fetch profiles" with "shoot's cloud
  profile not matched"** (silent in the latter). `gardener.go:139-146,162-165`.
- [x] **NIT — `--token-stdin` doesn't reconcile the token's owner with `-u`** (a typo'd
  `-u` is stored). `login.go:286-293`. Use the login name from `IdentityGetCurrentUser`,
  or warn on mismatch.
- [x] **NIT — `"full (all areas)"` hardcodes the area count as 7** instead of deriving it.
  `users.go:310`.

## Batch B — flag scoping & `-o` placement (highest-leverage DX, low risk)

_Done 2026-07-09: --region/--project-id scoped to gardener+login, -o to render commands only, with a flag-placement regression test. The --version/-o divergence is resolved by construction (-o is no longer a root flag)._

- [x] **`--region` / `--project-id` are global but no-ops on `whoami`, `user`, `logout`**
  (accepted, exit 0, silently ignored). Scope them to `gardener` (persistent flags on the
  gardener command) while keeping `login` able to store them. ~2 flag registrations, zero
  resolver changes.
- [x] **`-o/--output` is offered on ~7 side-effect commands** (`login`, `logout`,
  `config set/use-profile/delete-profile/path`) where it's a silent no-op (`-o json`
  ignored, `-o bogus` rejected). Attach `-o` only to commands that call `output.Render`.
- [x] **`--version` (plaintext) vs `version` subcommand (honors `-o`)** diverge — low
  scriptability wart; make the flag path render through the same code.

## Batch C — profile & login ergonomics (flagship polish)

_Done 2026-07-09, with regression tests. Adds `config current` and `config rename-profile`; the user 403 now points non-admins at `whoami`; config set announces creation and warns on username/token desync; delete-current hints at remaining profiles; list-profiles marks the stored current with an override note; config view flags an unknown profile; login prints a context line and no longer echoes a stray secret._

- [x] **HIGH — a `user` 403 for missing privilege dead-ends** without telling a non-admin
  that `whoami` shows their own account. Add the hint at the user-command call site (not
  in shared `apiAuthError`, which would make whoami's own 403 circular).
- [x] **MEDIUM — no `rename-profile`** (standard operation; today it's hand-edit or
  destructive delete+re-login). Add `config rename-profile <old> <new>` (rekey + update
  current_profile, refuse if new exists).
- [x] **MEDIUM — `config set` silently creates a profile** — a typo'd `--profile` mints a
  phantom with no "created" signal. Print "Created profile X" on first write.
- [x] **LOW — `config view --profile ghost`** shows a defaults table with no "does not
  exist" hint (it has `ProfileExists` but ignores it). The diagnostic command masks a bad
  `--profile`.
- [x] **LOW — `config set username`** on a logged-in profile silently desyncs it from the
  stored token, unwarned. Warn / suggest re-login.
- [x] **LOW — `list-profiles` `*` marks the effective (flag/env) profile, not the stored
  `current_profile`** — can contradict its own warning. Mark `cfg.CurrentProfile`.
- [x] **LOW — no quick "which profile am I on"** — add `config current` (prints the
  resolved profile name, no network).
- [x] **LOW — deleting the current profile** clears `current_profile` and falls back to a
  nonexistent `default` even when other profiles remain. Hint / auto-select if one remains.
- [x] **LOW — same-identity re-login shows a bare `Password:`** with no identity/profile/URL
  context. Print a context line first.
- [x] **LOW — the no-args guard echoes the (likely secret) argument** into stderr/logs.
  Report a count instead of the value.

## Batch D — naming restructure & docs

_Done 2026-07-09. config profile restructure (no aliases, clean break); gardener list WORKERS→POOLS, Succeeded→"ready", computed columns now in -o json via an embedded view model; gardener commands check auth before region/project; kubeconfig validity wording; docs sweep (roles→privileges, reconcile/user-get help, shoot region/project prerequisite, getting-started logout order, -o json stability note)._

- [x] **Naming inconsistency:** `config list-profiles`/`use-profile`/`delete-profile`
  (verb-noun) vs `user list` / `gardener shoot list` (noun-verb). Restructure to a nested
  `config profile list|use|delete` group, keeping the current names as hidden aliases.
  Pre-1.0 is the cheap moment.
- [x] Gardener list semantics: **`WORKERS` shows worker-pool count, not node count** —
  rename to `POOLS` or render a node range; **STATUS** shows raw `Succeeded` (map the
  steady case to "ready"); **kubeconfig "(valid X)"** asserts a lifetime the server may
  cap ("requested validity X"); **computed `UPGRADE`/normalized STATUS are table-only** —
  absent from `-o json/yaml` (add a view model or an `upgrade_available` field).
- [x] **Gardener context validated before auth** — a not-logged-in user is told "no
  region" before "log in". Check auth first in `gardenerContext`.
- [x] Docs: README still says "roles" (→ privileges); region/project prerequisite only on
  `shoot list` (add to wake/hibernate/reconcile/kubeconfig or the parent); `reconcile` and
  `user get` help too thin; getting-started block runs `logout` mid-sequence; state that
  only `get-credentials` (and `config view -o json`) are stable script contracts.
- [] ** Commands / Subcommnds that require project and region to be set should expose it in the help text *
    cleura gardener --help folr exampel do not expose any text that indicates that project-id and region has to be set in profile or via flags

## Refuted (for the record)
- "All-digit usernames are unreachable / resolve to the wrong user" — accurate mechanism
  but no realistic harm; Cleura usernames are not all-digits and the numeric path only
  ever resolves by ID.

## Deferred from earlier reviews (still open, see dx-backlog.md)
- shootAction closure simplification; move UPGRADE fetch out of the render closure;
  cleura-client-go `CreateShootAdminKubeconfig` helper + shared API-error type; retries for
  idempotent GETs; the ⚖ pre-1.0 decisions (token storage layout; commit the spec snapshot).
