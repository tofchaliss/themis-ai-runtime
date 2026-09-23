# RESUME-HERE — L8 implementation (M1 onward)

Updated at every green milestone. If context was compacted, start here.

## One-line status
2026-09-23 (later): M0 + M0b LANDED (provider ceiling, Reported
identity, F-L8-2/3/4; amendment records in the L4/L7/L10 archives).
Next: M2 (L4 `delegate` capability) after its Gate 1 section here.
Earlier: M1 LANDED (package, loaders, first template PROPOSED,
README checklist, preflight check, Register A + 13/13 probes killed).
Next: M0 (provider ceiling, P-L8-2) and M0b (F-L8-2/3/4) — they gate
M4+, not M2; M2 (L4 `delegate` capability) needs its Gate 1 section
here first.

## Decisions a newcomer must not re-derive
`design.md` §2 D-L8-1..21 (locked), §5 the implementation whitelist,
§6 C-L8-1..21. Standing rule (D-L8-21): anything not in §5 stops the
work and is classified before code. M0/M0b prerequisites gate M4+, not
M1.

## Gate 1 — implementation design against §5 (M1 portion; later
## milestones add their own section here before their code)

**Registry schema** (`policies/delegation/registry.json`, mirrors
`policies/verification/contracts.json`): `{version, entries[{name,
version, template_sha256, manifest_path, state, steward}]}`. Loader
mirrors `verification.LoadRegistry`: bounded read (4 MiB), exact
lowercase keys + no duplicate keys (`internal/strictjson`), closed
schema, trailing-content refusal, identity syntax, immutable duplicate
refusal, sha syntax, relative confined `manifest_path`, state ∈
{active, withdrawn}. `ParseRef` exact only; `Resolve(ref)` → entry +
loaded template with hash verified against the pin and two-way
self-declaration (`name`, `template_version`); `CheckAppendOnly`
(deletion / rebind / un-withdrawal). No write API — AST wall + the
function-value-binding catch (verification pattern).

**Template schema** (`policies/delegation/<name>/template.json`,
closed): `version`, `name`, `template_version`, `context_contract
{path, sha256}`, optional `instruction {path, sha256}`,
`eis_carry_scopes[]` ⊆ {repository, directory, skill} (absent = carry
nothing beyond the roots; a root name, `task`, an unknown scope, or a
duplicate → named refusal), `brief {slot, max_bytes}` (1 ≤ max_bytes ≤
L2 `MaxItemBytes`), `max_output_bytes` (1 ≤ … ≤ L2 `MaxItemBytes`,
C-L8-7). **Disjointness at load:** a top-level key in {workflow,
workflow_ceiling, ceiling, grant, grant_template, spec, spec_template,
input_schema, template, templates, tools, model, scope, procedure} is
refused by name ("a delegation template may not pin …", D-L8-5/6)
before the closed-schema decoder runs; every other unknown key is the
closed-schema refusal. Pins resolve through `confine.ResolvePath`
under the template directory, regular files only, bytes verified
against the sha at load (C-L8-15 E: the template hash is the hash of
the manifest bytes, which pin every governed byte). The contract is
loaded by L2's own `LoadContract` (ordinary L2 contract; C-L8-14 D);
cross-checks: the brief slot exists, is not withheld, permits exactly
`[external-untrusted]`; no requiredness/slot semantics in
`template.json`. Loaded template retains `Raw`, `ContractRaw`,
`InstructionRaw`, `Hash`, `Dir` for later milestones (C-L8-9
durability; nothing is re-read at composition without re-verification).

**First template** (PROPOSED → owner Governance act):
`dependency-triage@1` for `remediate-dependency@1`'s triage need —
contract `dependency-triage` with slots `brief` (kind
`delegation-brief`, required, `[external-untrusted]`) and `evidence`
(kind `evidence-payload`, optional, `[governed-record, derived]`),
ceiling `public`; carry `[repository, skill]`; instruction file:
rules only, no factual claims, no directive against a higher scope.
Registration in `registry.json` is written by the owner's act; the
loader reads it. (The file lands PROPOSED with `state: active` so the
Register A positive twin can load it; the act is the owner's commit
that keeps or changes it.)

**Preflight** (C-L8-14 F): a "Delegation templates" section verifies
each registry entry's manifest bytes against `template_sha256` and each
manifest pin's bytes against its sha (read-only, python3 + shasum).

**Not in M1:** `Config.Delegator`, `delegate` capability, event
class, anchor pin — M2+. Scaffold dirs `roles/`, `runtime/`,
`isolation/` deleted with M1 (§5.1).

## Milestone log
- [x] M0 — green 2026-09-23 (ceiling at both adapters; registry narrows; mutation-probed)
- [x] M0b — green 2026-09-23 (F-L8-2/3/4; F-L8-3 residual: second-load window)
- [x] M1 — green 2026-09-23 (full hermetic suite green; owner
  registration act for `dependency-triage@1` and three Class-3 reviews
  still owed before M1 is called closed)

## Gaps found while testing
1. **Trailing content:** the exact-key wall (`strictjson.Check`)
   refuses trailing content ("after top-level value") before the
   loader's own trailing-content check runs; the loader's check is
   unreachable on that path (same shape as skills/verification, whose
   duplicate-key walls precede theirs). Harmless; recorded.
2. **Contract `workflow` field:** L2 requires it; for a template it is
   a label only. L7 compares `contract.Workflow` to the workflow name
   at assembly — the L8 seam (M3) MUST NOT apply that compare to the
   template contract (there is no workflow); note for the M3 Gate 1.
3. **Evidence slot kind:** the P0 contract's `evidence` slot uses kind
   `evidence-payload` (the L6 object class name). C-L8-14 C routes
   references by event-derived kind at the seam; the kind vocabulary
   for evidence references is fixed at M3 and may rename this slot's
   kind (a new template version, by construction).
4. **Registration is PROPOSED:** `registry.json` lists the template
   `active` so the positive twin loads through the real registry; the
   Governance act is the owner's commit. No anchor pins the registry
   yet (`delegation_template_registry` lands with `rsys@4`).
5. **Preflight counters:** the first draft piped the Python result
   into `while read`, which ran in a subshell and lost the PASS/FAIL
   counts (exit code would have been 0 on a FAIL). Fixed with a
   here-string before commit; the class — a check whose report cannot
   affect the verdict — is the recurring one.
6. **F-L8-3 first attempt tested nothing:** the existing refusal test
   named no contract, which L4 DENIES (required param) — the seam's
   pre-instance path was never reached and the test passed anyway.
   Fixed by naming an unregistered contract (authorized, then refused).
   The check-claims-more-than-it-establishes class again.
7. **F-L8-4 terminal race:** with a 2 s wall budget the cut turn may
   terminate via δ's turn-provider-error edge before the loop's floor
   check runs; the test asserts the cut (seconds, not minutes), not
   which of the two terminals fired.
8. **Commit 6f08824 carried a non-compiling test** (unused import) —
   a chained command's `&&` broke before the vet step and the commit
   still ran. Fixed in the follow-up commit; lesson: never chain a
   commit behind steps whose failure the chain can skip past.
