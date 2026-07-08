# Plan: cleura CLI as an authentication source for the Terraform provider

Goal: `cleura login` once on a laptop, then `terraform plan` against the
cleura/cleura provider just works — the az-login/azurerm experience — without
changing anything for existing provider users or CI pipelines.

**Decided design** (see dx-backlog.md): the provider consumes credentials by
**exec'ing the CLI**, never by parsing `config.yaml` (declared internal).
Credential precedence in the provider: **explicit HCL > environment variables >
CLI**, the CLI strictly last.

## DX principles

1. **Zero-config happy path on laptops** — login once, every tool works.
2. **Determinism in CI** — env vars always win; a leftover CLI login on a
   runner can never hijack a pipeline.
3. **Errors name the fix** — every auth failure states the exact command or
   variable that resolves it, including token age when the CLI supplied it.
4. **One mental model** — both tools already share env vars and the profile
   concept; the integration must not introduce a second vocabulary.
5. **A stable machine contract** — one versioned JSON envelope; the config
   file stays free to evolve (keyring, split files) without breaking anyone.
6. **No breaking changes** — the provider auth chain only gains a new last
   tier; every existing configuration behaves identically.

## Phase 0 — CLI foundations (cleura-cli v0.2.0, ~half a day)

Ship these together; order matters inside the phase.

1. **Record `token_stored_at`** (RFC3339, UTC) in the profile at every login.
   First because it cannot be backfilled: tokens minted before this lands have
   unknown age forever. Enables "this token is 26h old" diagnostics with no
   network call.
2. **Add `version: 1`** to the config file schema — before any other new field,
   so future readers can tell generations apart. (Internal detail; the file
   remains not-an-API.)
3. **`cleura config get-credentials [--profile <name>]`** — the integration
   contract:
   - Output is always JSON (machine-facing, like `az account get-access-token`);
     `-o` does not apply.
   - Envelope: `{"version": 1, "profile", "cloud", "endpoint", "username",
     "token", "region", "project_id", "token_stored_at"}` — endpoint fully
     resolved, cloud included because it doubles as an API path parameter.
   - Exit codes are part of the contract: `0` credentials returned; `2` no
     credentials available (not logged in / profile has no token) with a JSON
     `{"error": ...}` on stdout so callers can distinguish "no CLI creds —
     continue the chain" from "CLI malfunction" (any other exit).
   - Optional `--validate` flag: round-trips `/auth/v2/tokens/validate` before
     answering. Not used by the provider by default (latency); for humans
     debugging.
   - README gains a **Tool integration** section declaring this command the
     stable interface and the envelope's compatibility rules (additive fields
     keep `version: 1`; breaking changes bump it and the provider supports
     both for one release cycle).
4. Unit tests (envelope shape, exit codes, profile selection) and a golden
   JSON fixture the provider repo can copy for its own tests.

## Phase 1 — Provider adopts cleura-client-go (~1 day, independent value)

Migrate the provider off its duplicated generated client onto
`cleura-client-go` v0.1.0+:

- Mechanical: swap imports, rename `...Response` wrapper types to
  `...APIResponse` (the suffix the shared client uses), delete the provider's
  `api/` package and the client-generation half of its `generate.sh`
  (tfplugingen steps remain).
- Why in this plan: auth behavior (headers, and later timeouts/User-Agent/
  retries) stays identical in both consumers, and API regeneration happens
  once instead of twice.
- Provider CI needs `GOPRIVATE=github.com/cleura/*` + a read token until the
  repos go public.

## Phase 2 — Provider: the `cli` credential tier (~1–2 days)

New auth chain in the provider's `Configure`, replacing the current
"HCL-or-env, else error":

```
explicit HCL attributes  →  CLEURA_* environment variables  →  cleura CLI  →  error
```

- **New attributes**: `profile` (optional; which CLI profile to read, also
  passed as `--profile`) and `use_cli` (optional bool, default `true`; the
  kill-switch for environments that must never touch ambient credentials).
- **Mechanics**: `exec.LookPath("cleura")`; missing binary silently skips the
  tier (it is a fallback), logged at debug level. Run
  `cleura config get-credentials` with a ~5s timeout; parse; require
  `version == 1`; consume endpoint/cloud/username/token.
- **Region/project_id**: available in the envelope but deliberately **not**
  used as resource defaults — infrastructure code should state region and
  project explicitly for reproducibility. Credentials are ambient; topology is
  not. (Revisit if users ask.)
- **Error UX** (the heart of the DX):
  - Nothing anywhere: one error listing all three tiers with copy-paste fixes
    — set `token` in the provider block, or `CLEURA_API_TOKEN` +
    `CLEURA_API_USERNAME`, or `cleura login`.
  - CLI credentials rejected by the API (401/403): "credentials from the
    cleura CLI (profile "x", stored 26h ago) were rejected — tokens are
    short-lived; run 'cleura login'." Uses `token_stored_at`.
  - Envelope version mismatch: "upgrade the cleura CLI (or the provider)".
- **Tests**: precedence unit tests (each tier shadows the next); hermetic
  acceptance tests using a fake `cleura` script on PATH that emits canned
  envelopes (success, exit-2, garbage, old version).
- Optional nicety: note the credential source in the provider's User-Agent
  (`auth=cli|env|config`) to ease API-side debugging.

## Phase 3 — Documentation and polish (~half a day)

- Provider registry docs: an **Authentication** page describing the three
  tiers in precedence order, with a copy-paste example per tier, and the
  token-lifetime caveat for the CLI tier.
- CLI README and provider README cross-link each other's auth stories.
- A "getting started in three commands" snippet everywhere it fits:
  `cleura login` → `terraform init` → `terraform plan`.
- Troubleshooting: what `get-credentials` exit codes mean, how to test the
  chain (`use_cli = false`, unset env, etc.).

## Phase 4 — Later, unlocked by API work (tracked as API asks)

- **Service tokens** (long-lived, scoped): makes the CLI tier CI-viable and
  removes the daily re-login on laptops.
- **Browser-assisted login** (device code): passkeys/WebAuthn/SSO — the
  envelope and provider code need zero changes; only `cleura login` grows.
- **Expiry in the API**: `get-credentials` gains `expires_at`; the provider's
  stale-token error becomes precise instead of heuristic.

## Compatibility verdict

The CLI's model is compatible with the provider by construction: the profile
stores exactly the tuple `Configure` needs (endpoint, cloud-as-path-parameter,
username+token, region/project defaults), the env tier is already shared
verbatim, and profiles map 1:1 onto provider aliases for multi-cloud setups.
The one structural difference from azurerm: `az` can mint fresh tokens on
demand, we cannot (no refresh without a password) — hence `token_stored_at`,
the crisp expiry error, and the service-token API ask as the durable fix.
