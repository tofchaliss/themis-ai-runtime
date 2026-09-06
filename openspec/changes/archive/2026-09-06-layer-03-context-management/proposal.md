# Proposal: Layer 3 — Context Management

**Change ID:** layer-03-context-management · **Status:** DRAFT for grilling · 2026-09-06
**Owner gate:** ships only after the grill closes Q-L3-1…9 (design.md) and the owner accepts.

## Why

L2 delivers exactly what the plan gathers — it refuses all judgment by design. Nothing yet decides *which* available evidence enters the model context when there is more than fits: no selection, ranking, dedup, compression, or token budgeting exists. L3 is that judgment layer — and precisely because it is the first layer whose job IS selection, the grill must establish where its judgment ends: selection shapes the model's world, and unprincipled dropping is withholding without a contract.

## What

A deterministic-first context-management layer at `src/harness/context` (extending the L2 package) plus reserved subdirs (`ranking/`, `compaction/`, `retrieval/`):

- **Selection policy as governed configuration** — filter/rank/dedup/budget rules are versioned artifacts (contract-adjacent posture), never ad-hoc code judgment
- **Token counting + budget enforcement** with typed outcomes: evidence dropped by budget is a recorded, typed state — never silent
- **Deterministic dedup** (byte/hash-identical; semantic dedup deferred)
- **Deterministic ranking** with stable tiebreaks; probabilistic rerankers/embeddings enter (if at all) as registered computations whose output is advisory ordering only
- **Compression as registered computation** — a summary creates propositions, so it is `derived`-class evidence with full computational provenance, never an in-place rewrite of the original (L2-5.1 verbatim invariant survives: originals stay reconstructable via trace)
- **Retrieval capabilities** (repo search/ripgrep, lexical, metadata filtering) as registered sources feeding L2's existing classification discipline

## What this change does NOT do

- No model-called selection at compose time (probabilistic components are registered producers, not deciders of availability)
- No override of contract semantics: required slots are never droppable; withholding stays contract-only
- No semantic dedup, rerankers, or vector stores in v1 (layer-doc "later"); no OpenSearch/Qdrant
- No change to L2's delivery invariants — L3 transforms the gathered set *before* composition through the same typed-state vocabulary

## Success criteria

Same gathered set + same policy ⇒ same selected set (byte-deterministic); every dropped/compressed item is trace-recorded with reason and reconstructable provenance; a required or governed slot provably cannot be dropped by budget; budgets fail closed; the L2 delivery proof still passes end-to-end through L3.
