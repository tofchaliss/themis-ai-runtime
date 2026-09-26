# Proposal: L11 ↔ Governance — the promotion boundary

Status: **OPEN 2026-09-26 — grill open (owner-led, one question per
turn).** Completion-matrix row 12. Not a re-grill of L11 (D-L11-1..20,
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
| Q-R-2 | What witness does a door act carry, and how does it cite L11 evidence? | each registration/activation entry gains `decision_ref` → a decision record in the governed tree citing L11 comparison packages by object id and root; the decider named as an authenticated principal where the door can observe one, else the reviewed commit's signer; L11 packages consumed by reference, never copied or interpreted | OPEN |
| Q-R-3 | Must a commissioned method be "promoted" (relied upon) for a commission to be valid? | no Themis check (locked D-C-2); the runtime refuses execution of an unregistered/withdrawn skill at admission and the anchor's `skills[]` bounds what can open, so an un-promoted method never yields a proposal-eligible execution; recorded as a corollary | OPEN |
| Q-R-4 | Does any automation path from L11 evidence to a door exist or get created? | none, by D-L9-16 and D-L11-14; the integration adds a way to CITE evidence in the human act, never a trigger; a test asserts no code path writes catalog or anchors | OPEN |

## Not in scope

No Themis registry of runtime methods · no auto-promotion · no proposal
artifact type in L11 · no change to D-L11-* or D-L9-16 · no
demo-specific field.
