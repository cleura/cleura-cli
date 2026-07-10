# Gardener CLI — full-coverage roadmap

Plan to grow `cleura gardener` from its current 5 commands to **complete coverage
of all 29 Gardener operations** in the Cleura REST API, with subcommands designed
for developer-experience parity with the rest of the CLI.

Produced 2026-07-09 from the live spec (`https://rest.cleura.cloud/apidoc.json`)
via a design workflow: 6 parallel per-family deep-dives + an SDK-readiness pass →
synthesis → adversarial critique (coverage + safety + DX) → a refine pass that
resolved every finding. Companion to [dx-backlog.md](dx-backlog.md); this
supersedes the backlog's `shoot get`, `project list`-adjacent discovery, and
`CreateShootAdminKubeconfig helper` items.

## Where we are

- **Done (20 of 29 operations):** the 5 original commands (`shoot
  list/kubeconfig/wake/hibernate/reconcile`) **plus roadmap batches A (discovery
  + bootstrap), B (reads), D (day-2) and F (observability), shipped 2026-07-10.**
- **Remaining (9 operations):** C (shoot create/edit/delete), E (worker-group
  CRUD + upgrade-conditions), and the cross-cutting `--wait` poller (G).
- **SDK:** the generated client (`cleura-client-go/api`) already exposes a
  `*WithResponse` method for **all 29** operations — nothing to regenerate. Two
  operations need a raw-body read helper (see [SDK work](#sdk--cleura-client-go-work));
  `shoot ssh-key` (batch F) currently inlines that workaround like `kubeconfig` does.

## Design principles (carried from the existing CLI)

1. **Strict noun-verb grammar.** `cleura gardener <noun> [<sub-noun>] <verb> [args]`.
2. **View-model before `output.Render`.** Every read builds a struct that embeds
   the raw `api.*` type plus computed/human fields, so `-o json/yaml` carries the
   same enrichment the table shows (the shipped `shootView` discipline).
3. **`projectScopedHelp`** on every project-scoped command; a distinct
   cloud-only context for the three ops the API scopes to the region tag alone.
4. **Secrets to 0600 files**, never echoed by default (with one documented
   exception: `kubeconfig`, for the `KUBECONFIG` piping idiom).
5. **Async ops return immediately** and print `watch progress with 'cleura
   gardener shoot list'` — matching today's UX. An opt-in `--wait` lands last, once
   (Batch G), never bolted on per-command.
6. **Destructive ops fail closed:** `prompt.confirm` (refuses on non-TTY) + an
   explicit `--yes`; `--force` is a *separate* flag that bypasses a safety
   precheck, never the prompt.
7. **House error style:** gerund-wrapped `"<doing thing>: %w"`, `apiAuthError`
   login hint on 401/403, exit 1 reserved for real failures.

## Proposed command tree

`[done]` = shipped · `[new]` = this roadmap. `→` names the generated SDK method.

```
cleura gardener                                        manage Gardener Kubernetes clusters (project-scoped)
├─ shoot
│  ├─ list                                      [done] NAME/REGION/K8S/UPGRADE/POOLS/STATUS            → GardenerListShoots
│  ├─ get <name>                                [new]  rich single-shoot detail (establishes shootDetailView) → GardenerGetShoot
│  ├─ create [<name>]                           [new]  create from -f JSON/YAML (+ scalar overrides); --example prints a spec → GardenerCreateShoot
│  ├─ edit <name>                               [new]  patch Changed-only mutable fields; REJECTS HA (use enable-ha) → GardenerEditShoot
│  ├─ delete <name>                             [new]  delete shoot + workloads (DESTRUCTIVE; --yes)   → GardenerDeleteShoot
│  ├─ check-name <name>                         [new]  is a name taken? cloud-only; --exit-code predicate → GardenerIsShootNameTaken
│  ├─ kubeconfig <name>                         [done] time-limited admin kubeconfig (raw body, 0600)  → GardenerCreateShootAdminKubeConfig
│  ├─ ssh-key <name>                            [new]  node SSH private key → -f 0600 | --stdout (SECRET) → GardenerGetShootSshPrivateKey
│  ├─ wake / hibernate / reconcile <name>       [done] shootAction lifecycle ops                       → GardenerWakeUpShoot / Hibernate / Reconcile
│  ├─ maintain <name>                           [new]  run the maintenance window now (shootAction)     → GardenerTriggerShootMaintenance
│  ├─ retry <name>                              [new]  retry the last FAILED operation (shootAction)    → GardenerRetryFailedShootOperation
│  ├─ enable-ha <name>                          [new]  SOLE HA-enable path (DESTRUCTIVE; --yes)         → GardenerEnableHighlyAvailableControlPlane
│  ├─ ca                                               Certificate-Authority rotation (sub-noun)
│  │  ├─ rotate <name> --stage prepare|complete [new]  two-phase CA rotation (BOTH stages gated)        → GardenerRotateShootCa
│  │  └─ status <name>                          [new]  current rotation phase + next-action guidance    → GardenerGetShootCaRotationStage
│  ├─ upgrade                                          Kubernetes upgrade-READINESS gate (sub-noun; does NOT bump the version)
│  │  ├─ check <name> [--to <ver>]              [new]  Ready/Pending/NotReady preflight; --detailed-exitcode → GardenerCheckShootUpgradeConditions
│  │  ├─ prepare <name> --to <ver>              [new]  green-light conditions (DESTRUCTIVE; --yes/--force) → GardenerPrepareShootUpgradeConditions
│  │  └─ revert <name> --to <ver>              [new]  back out prepared conditions (SAFE; no gate)     → GardenerRevertShootUpgradeConditions
│  ├─ monitoring                                       observability reads (sub-noun; pure GET)
│  │  ├─ credentials <name>                     [new]  Prometheus + Plutono logins (SECRET; masked)     → GardenerGetShootMonitoringCredentials
│  │  ├─ nodes <name> <worker-group> [--names-only] [new] per-node snapshot; --names-only → bare names  → GardenerGetShootWorkerGroupNodesOverview / (--names-only) GardenerGetShootNodeNames
│  │  ├─ node <name> <node-name>                [new]  one node's metrics (latest + series in json)     → GardenerGetShootNodeDetails
│  │  └─ worker-group <name> <group>            [new]  worker-group aggregate metrics                   → GardenerGetShootWorkerGroupDetails
│  └─ worker-group                                     worker-group (node pool) CRUD — sub-noun under shoot
│     ├─ list <shoot>                           [new]  pools of a shoot (synthesized from GardenerGetShoot — no API op)
│     ├─ create <shoot> --name … --machine-type … [new] add a pool (flags-primary + -f)                → GardenerCreateWorker
│     ├─ update <shoot> <pool>                  [new]  update a pool (read-modify-write)                → GardenerUpdateWorker
│     └─ delete <shoot> <pool>                  [new]  delete a pool (DESTRUCTIVE; refuse last pool)    → GardenerDeleteWorker
├─ cloud-profile                                       discovery/validation source (cloud-only)
│  ├─ list                                      [new]  machine types, k8s versions, images, regions    → GardenerListCloudProfiles
│  └─ show <profile-name>                       [new]  one profile in detail (same endpoint, filtered)  → GardenerListCloudProfiles
└─ project
   └─ bootstrap                                 [new]  one-time enable Gardener for the project (204)   → GardenerCommunicationBootstrap
```

Sub-noun nesting (`shoot ca|upgrade|monitoring <verb>`) is a **deliberate, newly
ratified convention** for cohesive families; single-op day-2 verbs stay flat
(`maintain`, `retry`, `enable-ha`). Worker-group (node pool) CRUD is the
**`shoot worker-group`** sub-noun (moved under `shoot` 2026-07-10) so it sits
with the other shoot sub-nouns; its help cross-references `shoot monitoring
worker-group` for pool metrics. `gardener` itself is a namespace: `shoot`,
`cloud-profile` and `project` are its sibling nouns.

## Implementation batches

Ordered by dependency (discovery → reads → writes → day-2 → advanced → observability
→ polling) and user value. Each is a small, independently verifiable increment in
the established A/B/C/D style.

### Batch A — Discovery & bootstrap (the pre-create spine) · effort M · ✅ SHIPPED 2026-07-10
**Commands:** `cloud-profile list`, `cloud-profile show <name>`, `project bootstrap`
Read-only + a trivial 204 write: the safest first increment, and it unblocks
everything. `cloud-profiles` is the single source that create/edit/worker/upgrade
reuse for **both** shell completion and client-side validation (k8s versions
[Supported only], machine types, images, volume types, regions/zones).
- **New plumbing:** `gardenerCloudContext(opts)` — auth + resolved `settings.Cloud`,
  **skips** `requireProjectContext` (verified: `ListCloudProfiles` takes only the
  region tag). Cloud-only leaves must **suppress** the inherited persistent
  `--region/--project-id` flags (`MarkHidden`), not just drop the help text.
  Export `cloudProfileView` + the completion/validator helpers for later batches.
- **`bootstrap` quirk:** APIResponse has **no** `JSON400` and returns 204 empty —
  success is `StatusCode()/100==2`; a 400 falls through to raw `Body`. A 404 likely
  means wrong region/project or Gardener not offered there — surface both.

### Batch B — Reads & view-models (read before write) · effort M · ✅ SHIPPED 2026-07-10
**Commands:** `shoot get <name>`, `shoot check-name <name>`, `shoot worker-group list <shoot>`
Establishes the reusable machinery the write batches consume: `shootDetailView`
(embed `GardenerShootShoot` + per-pool workers, maintenance window, hibernation
schedules, allowed CIDRs, HA flag, conditions/endpoints, UPGRADE hint) — reused
verbatim by create/edit's 202 render; `workerView`; and `shoot worker-group list`,
**synthesized from `GardenerGetShoot`** (there is no worker list/get API), which
supplies the `<pool-name>` that `shoot worker-group update/delete` need.
- `get`/`shoot worker-group list` use full `gardenerContext`; `check-name` uses A's cloud-only
  context + suppressed flags (its SDK signature has no region/project).
- `check-name` **defaults to exit 0** and renders `{name,is_taken,available}`;
  taken→nonzero is opt-in via `--exit-code` only (**not** `-q`, which is `--quiet`).
- Provides completions for C/D/E/F: shoot names ← `ListShoots`; pool/worker-group ←
  `GetShoot.provider.workers`.

### Batch C — Shoot lifecycle writes (create / edit / delete) · effort XL
**Commands:** `shoot create [<name>] [--example]`, `shoot edit <name>`, `shoot delete <name>`
The highest-value core capability and the batch that introduces file input and
destructive confirmation.
- **create:** body is irreducibly nested (`workers[]`, `infrastructure_config`
  with required `floating_pool_name`), so `-f/--file` (JSON/YAML, `-`=stdin) is
  **primary**; only top-level **scalar** overrides (`--kubernetes-version`,
  `--enable-ha-control-plane`), which override the file. `--example` (an embedded,
  copy-pasteable spec printed to stdout) is a **committed** deliverable — a nested
  body with no template is unusable. The positional `<name>` is the **sole** name
  override (the `--name` flag is dropped to kill the collision) and overrides the
  file's `.name`. Pre-flight the name with `check-name`; validate k8s
  version/machine type/image/volume against `cloud-profiles` before the round-trip.
- **edit:** first `PATCH` — send **only `cmd.Flags().Changed` fields** so an unset
  bool never silently flips state. `--kubernetes-version` → `body.kubernetes`
  (note: **not** create's `kubernetes_version`). Cannot change pools/name.
  **Rejects HA through both paths:** no `--enable-ha-control-plane` flag **and** an
  error if `enable_ha_control_plane` appears in the `-f` body (verified:
  `GardenerEditShootPatchShoot` *does* carry the field — which is exactly why edit
  must reject it, so the file path cannot bypass the gated `enable-ha`).
- **delete:** gardener's first destructive command.
- **New plumbing:** a `--file` loader (JSON-or-YAML, `-` stdin) + override
  precedence; a body-field-rejection check for edit's HA guard; **extend the
  `shootAction` struct** with `destructive bool` + `confirmPrompt`, registering
  `--yes/-y` and calling `prompt.confirm`. Both async ops ship **without** `--wait`.
- **Risk:** nested-body/precedence correctness; the PATCH field-omission footgun;
  first destructive command. Refuse downgrades; surface `JSON400` field messages
  verbatim.

### Batch D — Day-2 operations & CA rotation · effort L · ✅ SHIPPED 2026-07-10
**Commands:** `shoot maintain`, `shoot retry`, `shoot enable-ha`, `shoot ca rotate --stage prepare|complete`, `shoot ca status`
- `maintain`/`retry` are textbook `shootAction` fits (no body, 2xx=success).
- `enable-ha` is the **sole** HA-enablement path (edit rejects HA) and the first
  destructive `shootAction` (irreversible via CLI + cost increase — stated in
  prompt and help).
- `ca rotate` has a **flat** body (`{stage}`); accepts only `prepare|complete`
  (case-insensitive, `start` aliases prepare), rejects the 5 read-only enum states
  (→ `ca status`). **Both stages are gated:** `prepare` warns there is **no CLI
  path to abort** a started rotation (there is no revert op); `complete` warns to
  re-issue kubeconfigs. Do **not** trust the SDK enum `.Valid()` — it accepts all
  seven states; enforce in the CLI.
- `ca status` maps all 7 states to a computed next-action column (view-model).
- **Grammar note:** the `ca`/`upgrade` sub-nouns are ratified *before* this batch.

### Batch E — Worker groups (E1) & upgrade-conditions (E2) · effort XL
Delivered as **two verified sub-batches** to keep the micro-batch cadence.
**E1 `shoot worker-group create/update/delete`:** flags map a nested body (machine sub-object +
taints/labels/annotations) + `--file`; `update` **defaults to read-modify-write**
(fetch shoot, patch one pool, send the whole object — correct whether the server
merges or replaces) with explicit `--clear-*` flags and a clean "no such pool"
error before any 404; `delete` reuses C's confirmation and **refuses to remove the
last pool**.
**E2 `shoot upgrade check/prepare/revert`:** one shared `ShootUpgradeConditions`
renderer with outcome normalization (`Outcome`→Ready/Pending/NotReady). These
operate the **readiness gate** and do **not** bump the version — the actual change
is `shoot edit --kubernetes-version`; help says so both ways. `check --to` defaults
to the list-computed UPGRADE version but is **required** (clean error) when the
cluster is already newest — never send an empty `target_version` (all three ops
carry a **required** `TargetVersion` param). `prepare` runs `check` in-process,
refuses NotReady unless `--force`, confirms unless `--yes`, then PUTs. `revert` is a
**safe** recovery path — no gate. `--detailed-exitcode` (terraform-style): default
exit 0 on success; with the flag, 0=ready, 2=not-ready; exit 1 stays reserved for
real errors.

### Batch F — Observability & secrets · effort L · ✅ SHIPPED 2026-07-10
**Commands:** `shoot monitoring credentials|nodes|node|worker`, `shoot ssh-key`
Self-contained read family + the second raw-body secret.
- **Coverage fix:** `nodes` (default) invokes `getShootWorkerGroupNodesOverview`;
  `nodes --names-only` invokes `getShootNodeNames` **directly** — the clean 1:1
  binding, and its output backs node-name completion.
- Lead with CLI-computed **unit-independent ratios** (`cpu% = usage/allocatable`,
  `mem%`) since the API declares no units; table shows the latest snapshot
  (max-by-time; guard empty slices), `-o json/yaml` carries the full series.
- **Secrets:** `credentials` masks the password (reveal via `--show-secret` /
  `-o json|yaml` with a TTY-stderr warning / `-f` 0600); `ssh-key` requires exactly
  one of `-f` (0600) or `--stdout`, no default. Both raw-body reads gate on
  `StatusCode()/100==2` and read `resp.Body`; empty body = error.

### Batch G — Shared `--wait/--timeout` poller · effort L
Introduces the single opt-in polling mechanism and retrofits it across every async
op at once (including the shipped `wake/hibernate/reconcile`).
- **Pluggable `(fetch, isDone)` per op — the target, not just the predicate,
  varies:** most fetch `GardenerGetShoot` and test `LastOperation` 100%/Succeeded;
  `delete` treats 404/gone as done; **`ca rotate` polls
  `GardenerGetShootCaRotationStage`** (its phase is *not* on `GardenerShootShootStatus`,
  so a shoot-only poller would hang to timeout); `upgrade prepare` tests ready.
- **Decided semantics:** opt-in everywhere (never default-on, even on a TTY);
  default `--timeout 30m`; on timeout, exit **non-zero** with a "continues
  server-side" message — kept distinct from `--detailed-exitcode`'s 2 so `&&`
  chains and `if`-guards stay coherent.

## Coverage matrix (all 29)

| Operation | Command | Batch |
|---|---|---|
| listShoots | `shoot list` | done |
| createShootAdminKubeConfig | `shoot kubeconfig` | done |
| wakeUpShoot / hibernateShoot / reconcileShoot | `shoot wake/hibernate/reconcile` | done |
| listCloudProfiles | `cloud-profile list` / `show` | A |
| communicationBootstrap | `project bootstrap` | A |
| getShoot | `shoot get` (+ backs `shoot worker-group list`, the poller) | B |
| isShootNameTaken | `shoot check-name` | B |
| createShoot | `shoot create` | C |
| editShoot | `shoot edit` | C |
| deleteShoot | `shoot delete` | C |
| triggerShootMaintenance | `shoot maintain` | D |
| retryFailedShootOperation | `shoot retry` | D |
| enableHighlyAvailableControlPlane | `shoot enable-ha` | D |
| rotateShootCa | `shoot ca rotate` | D |
| getShootCaRotationStage | `shoot ca status` | D |
| createWorker | `shoot worker-group create` | E1 |
| updateWorker | `shoot worker-group update` | E1 |
| deleteWorker | `shoot worker-group delete` | E1 |
| checkShootUpgradeConditions | `shoot upgrade check` | E2 |
| prepareShootUpgradeConditions | `shoot upgrade prepare` | E2 |
| revertShootUpgradeConditions | `shoot upgrade revert` | E2 |
| getShootMonitoringCredentials | `shoot monitoring credentials` | F |
| getShootWorkerGroupNodesOverview | `shoot monitoring nodes` | F |
| getShootNodeNames | `shoot monitoring nodes --names-only` | F |
| getShootNodeDetails | `shoot monitoring node` | F |
| getShootWorkerGroupDetails | `shoot monitoring worker` | F |
| getShootSshPrivateKey | `shoot ssh-key` | F |

**29/29 mapped 1:1** (`listCloudProfiles` intentionally backs both `list` and
`show`; `getShoot` also backs the synthesized `shoot worker-group list` and the poller — read
reuse, not double-mapping). `--wait` (Batch G) adds no new op.

## SDK / cleura-client-go work

Nothing to regenerate — all 29 `*WithResponse` methods exist. Recommended
ergonomics work (not blockers):

- **`CreateShootAdminKubeConfig([]byte, error)` raw helper** — call the generated
  method, return `resp.Body` on 201, else map via the shared error helper.
  Replaces the workaround currently inlined at `cli/gardener.go:242-261` (the typed
  `YAML201 *string` field is never populated: a kubeconfig is a YAML mapping and
  the server sends a non-YAML content-type).
- **`GetShootSshPrivateKey([]byte, error)` raw helper** — same shape, but gate on
  `StatusCode()/100==2` (its APIResponse has **no** typed success field at all).
  This is the only new raw-body case with no CLI workaround yet.
- **Shared API-error helper** (backlog item) — promote `cli/root.go`'s
  `apiError`/`apiAuthError` (or add `func (FrameworkHttpErrorResponse) Error()` +
  `APIError(resp, body)`) into `cleura-client-go`. The generated parsers already
  populate `JSON4xx/5xx`; this helper owns the non-2xx path.

Informational: the POST/DELETE action endpoints (`delete*`, `rotateCa`, `enableHa`,
`maintain`, `retry`, `bootstrap`) have no typed success body **by design** — detect
success via `StatusCode()/100==2`, exactly as the shipped lifecycle ops do. The
three upgrade ops are the only Gardener ops taking a `*Params` struct (a required
`target_version` query param).

## Open questions

**Grammar / naming (maintainer's call — defaults chosen, easy to flip):**
- `monitoring node/nodes/worker` leaves are artifact-nouns, not verbs — kept as a
  documented read-family exception with explicit positional names. A rename
  (`node-metrics` vs `nodes`, or `--node/--group` flags) remains possible.
- `gardener project bootstrap` shares the `project` noun with a future top-level
  `project` (OpenStack projects). Paths never actually collide; `gardener enable`
  is the fallback if it confuses.

**Backend behavior to confirm (do not overclaim in help until answered):**
- **`updateWorker` PUT semantics** — merge (omitted = unchanged) or replace
  (omitted = cleared)? Read-modify-write is safe under both, but this decides
  whether `--clear-*` flags are meaningful and whether a partial `-f` PUT loses data.
- **upgrade-conditions vs edit ordering** — is a successful `upgrade prepare` a hard
  prerequisite before `edit --kubernetes-version` is accepted, or independent? Does
  `prepare` alone trigger anything server-side? Documented as gate-then-edit; exact
  ordering asserted nowhere until confirmed.
- **`bootstrap` sync vs async** — modeled as synchronous (204). If enablement is
  actually async, add a future `gardener project status` poll verb.

**Positioning — decide before Batch C:**
- This roadmap gives the CLI **full lifecycle CRUD** (`shoot create/edit/delete`,
  `shoot worker-group create/delete`), which **overlaps the terraform provider's role** and
  crosses the earlier "CLI owns day-2, terraform owns lifecycle" split. The user
  asked for full Gardener functionality, so the scope is intended — but confirm
  the CLI is deliberately a full lifecycle manager, and plan the README/help
  positioning update (when to reach for terraform vs the CLI) as part of Batch C.
  The backend open questions above are **must-confirm-before-coding**, not settled.

**Delivery:**
- Batch E ships as E1 (worker) + E2 (upgrade) to keep the verification cadence —
  confirm that's preferred over one atomic XL batch.
- Shipped decision (batch B): cloud-only `shoot check-name` inherits
  `--region/--project-id` from the gardener parent and **rejects** them if
  explicitly passed (rather than hiding them, which the shared persistent flag
  makes awkward). When batch A adds more cloud-only commands (`cloud-profile`),
  reconsider attaching the project-context flags per-command instead of
  persistently on the parent, so they can be genuinely hidden where unused.

---
*Provenance: designed by a fan-out/synthesis/critique/refine workflow over the live
OpenAPI spec and the generated client; all 29 operations verified present in
`cleura-client-go/api/client.gen.go`.*
