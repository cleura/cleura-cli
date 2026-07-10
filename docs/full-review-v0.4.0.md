# Full CLI review — v0.4.0 (2026-07-10)

Expert review of the whole `cleura` CLI at v0.4.0 (login, user, config, gardener —
~40 commands). Method: **live testing** against the API with profile
`claudetestuser` (cluster `hampus-tf-test`, left awake as found), plus a
4-lens static review (symmetry, docs, CI/CD-scripting, bugs) with every finding
adversarially verified against the code. Companion to
[gardener-roadmap.md](gardener-roadmap.md) and [dx-backlog.md](dx-backlog.md).

**Verdict:** solid and shippable. Live-exercised across `-o table/json/yaml`,
`--quiet`, exit codes, error paths and a `reconcile` action — all correct.
Auth/precedence, secret masking, clean 404s (API message + available-name hints),
and the `get-credentials` envelope/`--validate` all work. No data-loss or security
defects. The gaps are in **machine-output/scripting contracts**, **documentation
completeness**, and **consistency polish** — no blockers.

Live-verified working: `whoami`, `user list/get` (by id + name), `config
view/path/get-credentials/profile *`, `gardener shoot list/get/check-name/
kubeconfig/ca status/monitoring credentials|nodes|node|worker-group/
worker-group list`, `cloud-profile list/show`, `reconcile`. `-o json` valid for
all renderers; `-o yaml` snake_case correct; `--quiet` keeps stdout data-only;
unknown-subcommand exits non-zero.

Out of scope (deferred, not defects): shoot/worker-group create/edit/delete,
the `--wait` poller (Batch G), and the API-side workarounds tracked in
`client-api-wishlist.md` (which needs **no new entries** — this review surfaced
no new API gaps; all findings are CLI-side).

---

## Confirmed findings (prioritized)

### Scripting / machine-output contracts

- **[MED] `get-credentials` reports `cloud: "public"` for a private-cloud
  (api_url-only) profile.** `get-credentials` is *the* stable contract (the
  terraform provider + scripts consume it). For a profile with `api_url` set and
  no `cloud`, the envelope reports `"cloud":"public"` (the built-in default)
  while `endpoint` is correctly the private URL. A consumer keying gardener's
  region tag off `cloud` would use the wrong value. **Verified live** (env
  cleared): `endpoint=https://rest.cloud.acme.example`, `cloud="public"`
  (source `default`).
  *Fix:* emit `cloud` only when its source is not the built-in default (mirror
  the `persistedEndpoint` pairing already used at login); omit it for a private
  cloud. `getcredentials.go` envelope build.

- **[MED] `shoot check-name --exit-code` returns exit 1 for "taken", colliding
  with the generic error code.** Every operational failure (expired token, 5xx,
  network, usage) also exits 1 (`main.go:40`), so a create-if-free predicate
  silently treats a transient failure as "name taken". This contradicts the
  CLI's own convention (`get-credentials`: 2 = expected-negative, 1 =
  malfunction) and the roadmap's principle "exit 1 reserved for real failures".
  *Fix:* return **exit 2** for "taken" (`gardener.go:490`), document `0=available
  / 2=taken / other=error` in help + README. **Behavior change to a flag shipped
  in v0.3.0** — no test pins the old value (verified), so no regression cost.

- **[MED] `config view -o json` emits the table string `"(set)"`/`"(not set)"`
  as the token value.** Presentation leaks into machine output (`configcmd.go`
  computes the word into the struct before `Render`; the `""→"(not set)"`
  substitution for *other* fields is table-only, so the output is internally
  inconsistent: token→`"(set)"`, region→`""`). **Root cause is positioning:**
  `get-credentials` is already the machine contract, yet the README/help steer
  scripts to `config view -o json` *and* separately call it unstable.
  *Fix (positioning, not re-architecture):* stop recommending `config view -o
  json` for scripting (README/help) — keep it a human diagnostic; and clean the
  token value opportunistically (store `""` when unset, move the wording into the
  table closure, or add a boolean like `config profile list`'s `logged_in`).

### Documentation completeness

- **[MED] README is incomplete for v0.4.0.** Not *wrong* — *stale*: it mentions
  **none** of ~11 commands shipped since the first release (`shoot
  get/check-name/ssh-key/maintain/retry/enable-ha`, `ca`, `monitoring`,
  `worker-group`, `cloud-profile`, `project bootstrap`). Getting-started omits
  entire feature areas (monitoring, discovery, day-2). Also: the "Tool
  integration" convenience-`-o json` command list is stale; and the README
  recommends `config view -o json` for scripting while declaring it unstable
  (same root as above). *Fix:* refresh getting-started + add short sections for
  the gardener detail/day-2/monitoring/discovery commands; correct the
  tool-integration guidance to point at `get-credentials`.

### Consistency & polish (low)

- **[LOW] `whoami`'s Long promises "privileges" the default table doesn't show.**
  The Long (added in the docs pass) says it shows "the user's ID, name and
  privileges"; the table shows ID/Username/Name/Email/Admin/Currency/Profile —
  no privileges. *Fix:* narrow the Long (point to `user get` for the breakdown),
  or render privilege areas like `user get` does.

- **[LOW] `version` is the only rendering command without an `Example`.** *Fix:*
  add `cleura version` / `cleura version -o json`.

- **[LOW] `user list -o json` omits its computed columns** (privileges summary,
  2FA yes/no) — `view-model-before-render` not applied (renders the raw API
  slice), unlike `shoot list`. The actionable gap is `two_factor_active`: a
  script filtering "users without active 2FA" must re-derive it. *Fix:* a
  `userView` embedding the API user + `two_factor_active` (and optionally
  `privileges_summary`). `cloud-profile list`'s count columns are a lower-impact
  instance of the same.

- **[LOW] `monitoring node` rounds metrics (`0.08 cores`, `3.9%`) while
  `monitoring worker-group` prints raw sample precision (`0.1041 cores`,
  `6.2074%`).** Two sibling detail commands disagree. *Fix:* round the
  worker-group sample values to match (parse + `%.2f`/`%.1f%%`).

- **[LOW] Positional resource args have no shell completion and fall back to
  file completion.** `<shoot-name>`, `<worker-group>`, `<node-name>`,
  `<user>`, `<profile-name>` offer filenames on `<TAB>`. *Fix:* register
  `ValidArgsFunction` returning `ShellCompDirectiveNoFileComp` on every
  resource-name positional; ideally dynamic completion for `<shoot-name>` (from
  `ListShoots`) and `<worker-group>` (from the shoot's workers).

- **[TRIVIAL] `maintain`/`retry` `Long` say "the shoot's"** (their `Short`s were
  already fixed to "a shoot's"); **`check-name` hand-rolls its cloud-only help**
  instead of reusing the `cloudOnlyHelp` constant added in Batch A.

## Known limitation (enhancement, not a bug)

- **Exit codes: all failures collapse to 1** (except the `get-credentials`
  contract's 2 and, after the fix above, `check-name`'s 2). A script can't
  distinguish usage vs not-found vs auth-expired vs API error. Defensible as a
  baseline (non-zero = failure); differentiation is tracked in dx-backlog. The
  `check-name` fix addresses the sharpest instance.

## Refuted (checked and cleared)

Verified against code and dismissed: `get`-vs-`show` verb split (both idiomatic;
`show` = from a catalog); `kubeconfig` echoing to stdout (documented intentional
`KUBECONFIG`-piping exception); `latestSample` last-element-newest (documented
assumption; API lists ascending); CI `CLEURA_USERNAME` vs `CLEURA_API_USERNAME`
(different roles — a CI-defined value passed to `-u`, vs the env var the CLI
reads); `login` identity-replace lacking `--yes` (CI uses the password path).

## Recommended fix batches

- **R-A — scripting contracts** (highest CI/DX value): `get-credentials` cloud
  fix, `check-name` exit-2, `config view` positioning + token cleanup.
- **R-B — README refresh**: cover the v0.4.0 surface; fix the tool-integration
  guidance and the config-view contradiction.
- **R-C — consistency polish**: `whoami` Long, `version` Example, `user list`
  view-model, monitoring precision, no-file completion, the trivial Long/const.
