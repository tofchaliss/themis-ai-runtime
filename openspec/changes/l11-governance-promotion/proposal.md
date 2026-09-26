# Proposal: L11 ↔ Governance — the promotion boundary

Status: **GRILL CLOSED 2026-09-26 — D-R-1..4 LOCKED (`design.md`); implementation folded into I-M1 (`tasks.md`).** Completion-matrix row 12. Not a re-grill of L11 (D-L11-1..20,
archived 2026-09-13) nor of L9 (D-L9-16). Starting material: D-L9-16's
two gates and D-L11-1's ownership split.

## What the locked record already settles (facts)

- **L11 owns no promotion authority and no security meaning**
  (D-L11-1). "Every promotion remains an existing Governance act —
  skill catalog, contract registry, instruction revision, or Themis
  knowledge ingestion — exercised through its existing governed
  mechanism." L11 may make a suggestion evaluable and establish
  comparative/regression evidence; it can never make the suggestion
  become behaviour. Automation may complete a governed request, never
  begin one (D-L11-14).
- **The propositions L11 can state are bounded** (D-L11-4, D-L11-18):
  a conditioned comparative fact, Δ over metric M under criterion K.
  "This should be promoted", "this is safer", "this is acceptable" are
  Governance/Themis propositions, never L11's. Selection among admitted
  alternatives is not promotion (D-L11-8).
- **Baseline authority sits at the owning door** (D-L11-5): for a
  skill revision, the L9 catalog's admitted registrations; L11 keeps no
  registry of its own; its instance artifacts live in the L6 record
  (D-L11-11). `themis-ratchet` mints comparison packages or refusal
  facts into the record, synchronously, on explicit invocation.
- **Two gates for a skill revision, never collapsed** (D-L9-16):
  registration review (safety/architectural conformance, precedes ANY
  execution) and ratchet evaluation (quality, ordinary governed tasks
  under the registered candidate). Registration establishes
  executability, never production endorsement; **production reliance,
  non-reliance, and withdrawal are governance/deployment decisions.**
  No auto-promotion; no reduced review; no proposal artifact type.
- **How the doors are exercised today (facts from the tree):** a
  catalog entry is `{name, version, composition_sha256, manifest_path,
  state, steward}` — steward is accountability metadata, never
  authority. Registration and withdrawal are reviewed commits to the
  governed checkout. Production RELIANCE is the deployment anchor:
  `skill_catalog` pin + `skills[]` list, registered in `anchors.json`
  as `{name, version, artifact_sha256, state, steward}` by a host act.
  The most recent registrations were ratified by the owner as a
  recorded decision ("the decision is the act", L8 archive §7). There
  is no authenticated principal on either door and nothing links a
  door act to the L11 evidence that informed it.
- **Themis never validates runtime registries** (D-C-2, locked): it
  records the commissioned method identity and equality-checks the
  execution against it. The real Themis models capability promotion
  for ITS OWN Intelligence gateway as human-gated with no model
  registry (EDR-INTELLIGENCE-01 D-Δ4a-4); it holds no registry of
  runtime methods.

## The relationship to establish (owner)

```
L11 evidence → promotion evidence → Governance decision → skill@N+1 becomes governed
```

The key question: is promotion merely a runtime ratchet state, or a
Themis Governance act? If it affects which skill revision is governed
or usable, explicit Governance authority is expected — the grill
establishes this rather than assuming it.

## Grill — question table

| # | Question | Recommendation (PROPOSED) | State |
|---|---|---|---|
| Q-R-1 | Where is the promotion act; does Themis hold it? | **LOCKED 2026-09-26 → D-R-1:** the two existing doors (registration = executable, reliance = relied upon) remain the acts; no Themis approved-methods registry; the gap is provenance of the human act. | LOCKED |
| Q-R-2 | Door witness and L11 citation | **LOCKED 2026-09-26 → D-R-2:** `policies/decisions/<id>.json` per act (registration/withdrawal/reliance), two-way bound via `decision_ref`, loaders refuse on mismatch, version 2 requires it everywhere; `actor` = `commit:<author>` (asserted) or `key:<KeyID>` (authenticated); L11 evidence by reference; empty evidence explicit. | LOCKED |
| Q-R-3 | Must a commissioned method be registered/relied upon? | **LOCKED 2026-09-26 → D-R-3:** no Themis check; runtime admission alone decides executability; a commission that cannot execute stays a valid authority record and yields no proposal-eligible execution. | LOCKED |
| Q-R-4 | Any automation path from L11 to a door? | **LOCKED 2026-09-26 → D-R-4:** none exists, none created; read-only decision loader; ratchet forbidden from `skills`/`deployment`; one-way evidence reference; no-side-effect test; no preparation helper. | LOCKED |

## Not in scope

No Themis registry of runtime methods · no auto-promotion · no proposal
artifact type in L11 · no change to D-L11-* or D-L9-16 · no
demo-specific field.
