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

## 2. Design decisions (DRAFT — grill targets)

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

## 3. Open questions for the grill (Q-L2-n)

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
