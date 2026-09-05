# Design: Layer 2 — Context Delivery

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §5 (layer doc), the closed L1 design (`../layer-01-instructions/design.md` §6 — its L2 obligations bind here), `ARCHITECTURE.md`, the proven Model Interface (`runtime/model`), the L1 delivery seam (`instructions.EffectiveSet.Render/SystemMessage`).
**Decision IDs** `D-L2-n`; **open questions** `Q-L2-n` — the grill's targets.

## 0. Position in the flow

```
L1 Resolve → EIS ──Render(policy)──► system message + render hash
                                          │
Task payload facts ─┐                     ▼
Registered          ├─► ContextItems ─► Compose ─► model.ExecutionRequest messages
connectors ─────────┘   (typed, hashed,      │
                         trust-classed)      ▼
                                     Delivery trace {EISHash, PayloadHash,
                                      ContextHashes, Sources, TrustClasses}
```

L2 composes; it never calls the model (L7's job) and never selects/budgets (L3's job).

## 1. Hard invariants (inherited, not grillable)

- Instructions ≠ Context: no context byte can enter instruction text (L1 zero-interpolation holds; the EIS system message is delivered verbatim, hash-bound).
- Context is data, never instructions: every item is delivered inside structural data framing with its trust class; external content framed as untrusted (3.2).
- Recognition never discovery: only registered connectors/sources produce ContextItems; the agent encountering data never creates a source.
- No direct database access — explicit Themis APIs only (layer doc §5).
- The model payload is not the decision surface; L2 composes model input only.
- Epoch context boundary (L1 Q-L1-5): a new EIS epoch never silently inherits instruction-bearing conversation; carryover is explicitly classified data.
- Deterministic: same EIS + same items ⇒ same payload bytes ⇒ same hash.
- L2 failure is bounded: it can starve or garble model input, never bypass authorization, verification, or governance (four-transition test).

## 2. Design decisions (DRAFT — **amended by the grill record (§3), which governs where they conflict**; notably: D-L2-1's trust classes are superseded by the four-class authority taxonomy + transport metadata; D-L2-3's framing is governed by the dual-reader model; D-L2-2's sources sit under the Themis workflow contract ceiling)

### D-L2-1 — ContextItem is the only unit of context

```go
type TrustClass string // governed-record | tool-output | external-untrusted
type ContextItem struct {
    Kind    string     // e.g. "finding", "sbom", "source-file", "cve-description"
    Source  string     // registered connector/source ref
    Trust   TrustClass
    Content []byte     // byte-exact
    Hash    string     // SHA-256 of Content
}
```
Facts travel only as items; items never merge into each other or into instructions. Trust classes mirror the shipped `themis.tier-behavior` instruction so the model's stated weighting rule and the delivered labels agree.

### D-L2-2 — Registered context sources, mirroring L1's source discipline

A `ContextSource` is registered per task by the orchestrator (trusted config): v1 kinds — inline task facts (the envelope's data half), local filesystem paths (workspace-confined), Themis API reads (via `integrations/themis`, later wiring). Unrecognized kind ⇒ hard error. Task-payload facts are `external-untrusted` unless they are verbatim governed records fetched by the harness itself.

### D-L2-3 — Payload shape: one system message + one structured user message

System message = `EffectiveSet.SystemMessage(policy)` verbatim. Context renders into a single user message: fixed furniture headings per trust class, each item wrapped in deterministic delimiters carrying kind/source/trust/hash. Provider-agnostic; no provider-specific roles (DEC-05).

### D-L2-4 — Delimiter integrity

Item framing uses a content-derived fence (delimiter embeds the item hash prefix), so content containing the literal fence text cannot close a frame it did not open. Composition refuses (fail closed) if content contains the exact computed fence — detect/refuse, never rewrite (L1's final-pass discipline).

### D-L2-5 — Deterministic composition + trace

`Compose(set *instructions.EffectiveSet, policy *instructions.Policy, items ...ContextItem) (*Payload, error)`; `Payload{Messages []model.Message, EISHash, RenderHash, PayloadHash string, Items []ItemRef}`. Sorted deterministic item order (trust class, kind, source, hash). PayloadHash = SHA-256 over canonical serialization of all delivered bytes. Trace shape for L6: {EISHash, RenderHash, PayloadHash, ContextHashes, Sources, TrustClasses}.

### D-L2-6 — Epoch boundary mechanics

`Compose` builds a fresh message list every time — there is no append-to-conversation API in L2. Multi-turn continuation within one epoch is L7's loop (tool results etc. via the Model Interface); crossing an epoch means a new Resolve + new Compose from classified ContextItems only.

### D-L2-7 — Caps

Per-item and total-context byte caps (fail closed) as v1 constants, revisited when L3 owns budgets — same posture as L1's Q-L1-6.

## 3. Grill record (2026-09-05) — owner's Round-1 numbering (Q-L2-1…10) supersedes the draft's

### Q-L2-1 — Who decides required context (CLOSED, locked)

Owner-locked principle:

> Themis defines the workflow context contract. L7 instantiates and sequences a plan strictly within that contract. L4 independently authorizes any capability operation. L2 executes the authorized plan without judgment, performing retrieval, validation, normalization, provenance tagging, and packaging. Model-identified context gaps are advisory and cannot expand the contract. Contract-declared mechanical closure is the only form of context expansion. Delivery and tool-execution traces together must reconstruct every model-visible byte.

The four grill amendments, now invariants:

1. **L2 has no retrieval-time judgment.** The only "expansion" is mechanical closure explicitly declared by the contract (e.g. "SBOM + transitively referenced components, depth ≤ 1"); L2 executes the rule, never interprets it. Boundedness is declared in the contract artifact, never judged at retrieval time.
2. **L7 does not authorize.** Three verbs, three owners: L7 *considers* (pursue? when?), L4 *authorizes* (capability + policy, independent of model desire), L2 *fulfills*. A model context request is a capability request — no second permission table in L7. **Uniform gate:** initial planned context flows through L4 exactly like expansions — no privileged "planned" bypass.
3. **The Themis contract is the ceiling.** `L7 Plan ⊆ Contract`, else fail closed; `Requested Context ⊆ Contract-Reachable Context` for expansions; the sensitivity ceiling rides in the contract. The contract is a real versioned artifact — {version, hash, permitted classes, required/optional, authority classification, sensitivity ceiling, closure rules, expansion ceiling} — its hash recorded in the delivery trace.
4. **Complete reconstructability.** Delivery trace + tool-execution trace jointly cover every model-visible byte: request, authorization decision, source, trust class, content hash, delivery/visibility info — expansions recorded with the same completeness as composed context (protects the Q-L2-4 epistemic invariant).

Causal chain (owner-sharpened): the model may identify an information gap; the gap is advisory only; L7 may choose to pursue it; L4 independently authorizes the resulting capability operation; L2 fulfills within the contract's declared closure and sensitivity ceiling. A model request never automatically triggers a fetch.

### Q-L2-2 — Model-requested context (CLOSED as corollary of Q-L2-1)

Yes, advisory only; L4 authorizes (never L2/L7); arbitrary data access is outside both the capability vocabulary and the contract ceiling; results recorded in the tool-execution trace with composed-context completeness.

### Q-L2-3 + Q-L2-4 — Withholding and typed absence (CLOSED)

Owner-locked closure:

> A governed record may be excluded from a model's analytical context only when the versioned Themis workflow contract explicitly permits the exclusion. The exclusion must remain visible as typed epistemic metadata (`withheld_by_contract`), without exposing the record's substantive content. L2 must never turn known-but-undelivered information into `not_applicable` or silent absence. The marker is metadata, not an instruction. Authoritative governance/decision surfaces cannot use analytical exclusion to hide governed truth.

Supporting rules (grill-proven):

- **ContextAvailability enum:** `delivered` · `unavailable` (expected, could not obtain) · `not_applicable` (class does not apply) · `withheld_by_contract` (exists, intentionally excluded). Four states never collapse into absence.
- **Epistemic honesty over artificial de-anchoring:** "I don't have this information" (honest) beats "this information does not exist" (potentially false). The marker's residual anchoring signal is accepted — the two objectives (model unaware of existence; model never falsely infers nonexistence) conflict, and a security workflow prefers honesty. The model recognizing "I cannot fully assess because a Position was withheld" is acceptable epistemic self-awareness.
- **Minimum disclosure:** the marker reveals existence + availability state only — no content, rationale, status, author, or history.
- **Marker is metadata, never guidance:** absence metadata describes epistemic state; it must not prescribe model behavior ("do not consider…" would be L2 injecting instructions — L1/L2 separation violation).
- **Exclusion permission is a contract-level property, not an L2 decision;** analytical surfaces may withhold, the governance/decision surface never can. **Independent analysis ≠ independent decision:** the advisory conclusion still lands at Themis governance with full authoritative state.
- Rejected: B (silent exclusion — manufactures a false world-model), C (always-deliver + instruction framing — post-exposure de-anchoring is weak, and it makes context contracts unable to control analytical surfaces).

### Q-L2-5 — Context containing instructions + framing mechanics (CLOSED, hardened)

Owner-locked invariants:

- **L2-5.1 Verbatim evidence:** delivered byte-exactly; L2 MUST NOT sanitize, paraphrase, redact, or rewrite evidence as an anti-injection mechanism (hash integrity + the analyst must see what the evidence says).
- **L2-5.2 Plane separation:** the L1 directive-pattern policy applies to the instruction plane and MUST NOT be a context-content rejection mechanism ("does evidence contain imperative language?" is not "is this instruction manipulating the instruction system?").
- **L2-5.3 Structural isolation:** serialization MUST structurally isolate metadata from evidence such that evidence bytes cannot alter frame structure.
- **L2-5.4 No semantic parsing:** frame parsing MUST NOT depend on semantic interpretation of evidence content (evidence that *looks like* metadata is just bytes).
- **L2-5.5 Collision handling:** a framing collision ⇒ deterministic refusal, never rewriting or encoding. Collision ≠ attack signal (short prefixes collide innocently; no security intent encoded into the event).
- **L2-5.6 No secret required:** framing MUST NOT rely on secrecy — it is a serialization boundary, not authentication.
- **L2-5.7 Metadata neutrality:** L2-authored frame metadata is descriptive only (kind/source/trust/hash), never behavioral ("treat the following skeptically" would make L2 an instruction author).
- **L2-5.8 Complete traceability:** refused, delivered, and expanded items all appear in the delivery/tool trace; model-visible evidence stays reconstructable.

Framing architecture (owner decision): *use unambiguous length/structure-based serialization where available; if delimiter-based serialization is required, use a deterministic content-bound fence with collision refusal; framing security never depends on secrecy; evidence is never rewritten to satisfy framing.* **Two-reader rationale (recorded):** the mechanical reader (trace/audit) gets length-framed canonical records — unambiguous parsing; the model reads through visible delimiters — best-effort cognitive hygiene that no scheme can guarantee. The real safety property is architectural non-authority: no path exists from context to the L1 resolver, and any model-originated effect remains subject to the deterministic authority, authorization, verification, and state-transition controls of the owning layers (broadened from "dies at L4" — L4 is not the whole boundary).

Refusal visibility: a framing refusal is **trace-visible** (`delivery_status = refused`, with source/hash/serializer detail), never model-visible as a reason; the model sees only the Q-L2-4 availability state, with **source status** (available/unavailable) distinguished from **delivery status** (delivered/refused/withheld) so refusal does not falsely imply source unavailability.

Owner-locked final principle (Q-L2-5 CLOSED):

> Canonical context records use unambiguous length/structure framing for deterministic reconstruction. Model-facing rendering uses explicit visible structural markers, with delimiter collision refusal where delimiter framing is used. These mechanisms provide serialization integrity and cognitive hygiene respectively; neither authenticates evidence nor guarantees model cognition. The security boundary is the architectural separation between data and authority.

Standing warning: a future engineer must not replace the model-facing visible markers with length framing merely because the canonical parser already parses unambiguously — length framing answers "can the system reconstruct the frame?", not "how does the model perceive where evidence begins and ends?", and neither answers "can the model be persuaded to treat evidence as an instruction?" (that property is architectural non-authority). **L2 protects the integrity and provenance of context delivery; it does not attempt to make the model itself trustworthy.**

### Q-L2-6 — Context vs security truth: derivation, reconciliation, jurisdiction (CLOSED, hardened)

Owner-locked final principle:

> L2 may transform representation, never security meaning. L2 may only deliver a derived security proposition when that proposition was produced by an explicitly owned, registered, versioned computation whose output vocabulary permits it and whose complete computational provenance is recorded. Registration grants bounded computational jurisdiction, not security authority. Reconciliation, ranking, applicability adjudication, Enterprise Position, and decision propositions remain with their respective owners regardless of deterministic computation. **Determinism qualifies a mechanism; it does not determine jurisdiction.**

Supporting rules (grill-proven):

- **Proposition, formalized:** a claim whose truth value could alter security interpretation, applicability, risk, governance state, or decision state. "1.2.3 < 1.2.5" is mathematics; "CVE affected = TRUE" is a security proposition — "I only calculated a Boolean" is not a defense.
- **Proposition taxonomy** (registration binds analyzers to their permitted kinds): evidence proposition (VEX.status) · derived technical proposition (hashes, version parses, dependency edges) · security applicability · governance position · decision proposition. The last three are never "derived facts"; a computation MUST NOT emit propositions owned by Security Governance or decision authority.
- **Adjudication-by-decomposition defense:** an analyzer whose logic is "IF VEX says X THEN applicability = X ELSE version-compare" is executing conflict-resolution policy, not deriving. The clean analyzer emits independently attributable propositions (VEX status, scanner status, version-comparison indication, conflict-detected = TRUE) and governance executes the precedence rule. Registration is judged by output vocabulary and jurisdiction, never by name or declared purpose.
- **Derived applicability ≠ Enterprise Position ≠ decision** — a registered engine may emit `applicability_assessment` (authority_class = derived) for the model and governance to consider; it is never authoritative.
- **Why even "trivial" derivation leaves L2:** ownership, not determinism, is the boundary — and the trivial case is wrong anyway (backports: installed < fixed while vendor VEX = NOT_AFFECTED; epochs; multi-branch fixes). "Completely deterministic" smuggles in "trivially correct."
- **Normalization invariant (corrected):** a lossless normalization must have a deterministic inverse/canonical reconstruction preserving the source datum's semantic content without introducing a new security claim (literal byte round-trip not required — `"1.2.3"` → {1,2,3} is legal).
- **Audit boundary:** every delivered proposition is attributable to a source item or an explicitly registered computation.
- **DerivedFact provenance (complete or it is a black box):** {proposition, producer_id, producer_version, algorithm/rule hash, **configuration hash** (all security-semantic config — e.g. distro, backport_policy — else "deterministic" is nominal), input item hashes, computation_id, output hash}. Historical reconstruction: same inputs + same versions + same config ⇒ same result, provable.
- **Possession of a rule ≠ ownership of its execution** (Q-L2-6.5): a precedence rule in the contract is a governance rule in transit to its owner; L2 delivers the evidence the rule needs, governance executes it.

### Q-L2-7 — Incomplete context (CLOSED)

Required-missing ⇒ **fail closed at composition — no model call, no degraded package** (a package missing required evidence is a different, unreviewed evidence basis; L1 atomicity transplanted). Optional-missing ⇒ typed `unavailable` state (Q-L2-4 vocabulary, source-status/delivery-status split). Required/optional classification lives in the versioned workflow contract. The model never decides necessity.

### Q-L2-8 — Raw database access (CLOSED as corollary)

No model↔DB relationship; L2 holds no database credentials and speaks only Themis-owned data/capability boundaries. Flow: model → capability request → L4 → Themis-owned API → L2 fulfillment → storage. Architectural bonus: DB schema ≠ context contract ≠ model vocabulary — a migration is never automatically a model-facing change.

### Q-L2-9 — Transform boundary (CLOSED, subsumed by Q-L2-6)

All governing content lives in the Q-L2-6 closure in stronger form: proposition test, lossless-normalization boundary, hash-attributability, registered-computation provenance, bounded proposition vocabulary, jurisdiction ≠ determinism.

### Q-L2-10 — Unit and taxonomy of context (CLOSED, hardened)

Owner-locked: typed evidence objects; **four mutually exclusive authority/provenance classes** determined from authorship + registered governance treatment; **tool-output is transport metadata, never an authority class**.

| Authority class | Proposition author | Qualification |
| --- | --- | --- |
| `governed-record` | Themis governed process | Themis-originated governed truth |
| `governed-external` | External party | Externally authored, accepted/stored/managed through a governed Themis process |
| `derived` | Registered computation executed by Themis | Bounded proposition vocabulary + complete computational provenance |
| `external-untrusted` | External party | No qualifying governance treatment |

Locked invariants:

- **Classification invariant:** an item's authority class is determined by who authored its propositions and the registered governance treatment of that content — not by where it is stored, what retrieved it, or how it was transported.
- **Fail-closed classification:** unproven provenance ⇒ `external-untrusted`; higher classes are earned through registered provenance, never asserted.
- **Immutable origin:** external-authored survives storage/retrieval/re-serialization/transport; only an explicit governed transition changes authority class, and even then external authorship remains recorded. Laundering path closed: storage ≠ producer ≠ authority class ≠ transport ≠ security truth.
- Class describes the producer category, never truth value or priority — class meaning for adjudication stays with governance and (advisorily) the model's tier instruction.
- Edge classifications: harness-executed scanner ⇒ `derived` (registered capability, config-hashed); tool-fetched raw external content ⇒ `external-untrusted` regardless of tool; Themis-ingested CVE text ⇒ `governed-external`.

Final envelope:

```
ContextItem { kind, provenance{origin, source, author}, authority_class,
              producer, sensitivity, version/as-of, evidence_bytes, hash }
ContextEnvelope { task_id, slot, source_status, delivery_status,
                  delivery{mechanism: planned-connector | capability-fetch} }
```

Delivery mechanism carries zero authority weight.

**L1 content dependency (recorded):** `instructions/themis/tier-behavior.md` must be revised to speak the four-class vocabulary the envelope exposes (with `governed-external` placed explicitly: storage attestation governed, content external prose). L1 instruction-content change under the established content-review gate, executed when L2 implementation begins.

### Grill board — ALL CLOSED 2026-09-05

Q-L2-1 amendments · Q-L2-2 corollary · Q-L2-3+4 amendments · Q-L2-5 closed · Q-L2-6 closed · Q-L2-7 closed · Q-L2-8 corollary · Q-L2-9 subsumed · Q-L2-10 closed. The L2 architectural grill is closed; remaining items are implementation planning (filesystem confinement root, caps, Themis connector real-vs-stub, operational-proof gate, tier-behavior vocabulary revision, tests/evidence) — tracked in tasks.md, no boundary reopening.

## 4. Draft open questions (SUPERSEDED — all resolved by the grill record above)

1. **Q-L2-1 — Trust-class taxonomy:** are three classes (governed-record / tool-output / external-untrusted) right, and who assigns them — the connector (fixed per source) or per-item? Proposal: fixed per registered source, never caller-supplied per item.
2. **Q-L2-2 — Payload shape:** one structured user message (D-L2-3) vs one message per item vs provider-specific structures. Does a single user message survive L3's future budgeting?
3. **Q-L2-3 — Delimiter-spoof defense:** is the hash-derived fence + refuse-on-collision (D-L2-4) sound, or is escaping/encoding (e.g. base64 for colliding content) better than refusal?
4. **Q-L2-4 — Task-fact trust:** the envelope's data half — always `external-untrusted`, or may Themis mark payload facts as governed records? (Caller-supplied trust vs "no caller-supplied field determines its own treatment".)
5. **Q-L2-5 — Filesystem connector confinement:** workspace-confined reads before L5 worktrees exist — what is the v1 confinement root and who sets it?
6. **Q-L2-6 — Themis API connector:** v1 real (wire `integrations/themis` read paths) or typed stub with the contract only?
7. **Q-L2-7 — Cap numbers:** per-item / total-context caps for v1?
8. **Q-L2-8 — Operational proof:** what is the required operationally-proven gate for L2 — mock-provider composition proof only, or a live local-model run of a full L1+L2 payload?

## 4. Interfaces to neighbors

- **← L1:** consumes `EffectiveSet` + `Policy` via `Render`/`SystemMessage`; records EISHash + RenderHash per composition.
- **→ L3 (future):** L3 will select/rank/budget ContextItems before Compose; Compose's signature stays.
- **→ L7 (future):** L7 calls Compose per epoch, then drives the Model Interface loop; mandatory-root registration and epoch succession live there.
- **→ L6 (future):** the delivery trace shape above.

## 5. Test plan (three-state discipline)

Every milestone review states three verdicts independently: architecture-conformant, test-evidenced (named tests exercising the claimed lines — verified via coverage), operationally proven (real execution path). Planned evidence: determinism/golden payload + hash; item-order independence; delimiter-collision refusal; instruction/context separation (no context byte in the system message); untrusted framing golden; epoch-boundary (no inheritance) proof; caps; connector registration failures; mock-provider delivery proof; live-model proof per Q-L2-8 decision.
