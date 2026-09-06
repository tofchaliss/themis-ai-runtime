# Proposal: Layer 2 — Context Delivery

**Change ID:** layer-02-context-delivery · **Status:** GRILLED 2026-09-05 — all ten questions closed (design.md §3); pending owner acceptance of the folded change (tasks.md gate 0). **Implementation not started.**
**Owner gate:** implementation starts when the owner accepts the post-grill proposal/design/tasks and decides the four gate-0 planning items.

## Why

L1 produces the EIS; nothing composes it with task evidence into a model payload. Every run today would hand-assemble messages. L2 is the reference flow's terminal ("Context delivery") and the next layer in the owner-locked sequence. The L1 grill left L2 explicit obligations: deliver the EIS verbatim with per-call hash recording, keep context (facts) structurally separate from instructions (rules), label untrusted content as data, and establish epoch context boundaries.

## What

A deterministic context-delivery system at `src/harness/context`:

- **ContextItem**: typed evidence with provenance — kind, registered source, byte content, hash, trust class (governed-record / tool-output / external-untrusted)
- **Registered connectors only** (recognition never discovery, mirroring L1 sources): local filesystem, inline task data, Themis API (via `integrations/themis`); no direct DB access ever
- **Payload composition**: EIS system message (verbatim, hash-bound) + context as structurally delimited data sections in non-system messages; provider-agnostic (DEC-05)
- **Untrusted-content framing**: every context item delivered inside deterministic data delimiters with its trust class; delimiter-spoof defense
- **Delivery trace**: {EISHash, PayloadHash, ContextHashes, Sources, TrustClasses} per composition — the L6 shape
- **Epoch context boundary**: composing for a new EIS epoch never inherits prior instruction-bearing conversation; cross-epoch carryover only as explicitly classified ContextItems

## What this change does NOT do

- No selection, ranking, dedup, compression, or token budgeting (L3)
- No tool authorization or execution (L4); no model calls (L7 orchestrates)
- No retrieval policy over external web/browser (deferred per layer doc)
- No human decision surface — the model payload is not the decision surface (grill invariant)

## Success criteria

Composition is byte-deterministic (same EIS + same items ⇒ same payload + hash); the EIS arrives verbatim (delivery-proof test); no context byte can enter instruction text (structural, tested); every item in the payload is attributable to a registered source with hash + trust class; epoch-boundary test proves no conversational inheritance.
