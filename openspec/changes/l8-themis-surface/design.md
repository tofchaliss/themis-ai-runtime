# Design record: L8 × Themis surface (D-L-*)

Locked decisions in the order the owner disposed them. Nothing here
changes D-L8-1..21; each decision is an integration/projection decision
over the already-witnessed delegation fact.

## D-L-1 — Delegations are rendered within the model-reasoning fact; proposal evidence references them by seq (LOCKED 2026-09-26, owner)

> Every delegation remains a runtime execution fact witnessed by
> `l8-delegation`; intake renders it within the model-reasoning
> evidence, and proposal evidence references its sequence numbers
> without copying or reinterpreting the delegation.

Delegation is part of the model-reasoning fact — NOT a fourth evidence
category. The three-way separation stands:

| Fact | What it establishes |
|---|---|
| Model turns + delegations | what reasoning/work the model path performed and what evidence entered that path |
| Artifact bytes | what was actually produced |
| L10 outcome | what verification established |

**Rendering, per delegation:** seq · parent call seq · template
name@version · model identity · each evidence reference with the
authority class it carried into the composition · output object id ·
outcome. The evidence refs answer what ordinary model-turn rendering
cannot: *what governed evidence entered the delegated reasoning path?*
— presenting an existing runtime fact (already checked against the
registry and context contract), creating no new trust relationship.

**Proposal evidence:** `harness-execution/v1` gains `delegations:
{count, seqs[]}`. Whole delegation records are NEVER copied into
Themis: Themis proposal evidence → delegation seqs → L6 record →
complete witness. No second authoritative representation.

**Wording constraint (owner):** the record establishes which delegate
execution was invoked, which model identity it used, which evidence
references entered its composition, what output it produced, and its
outcome. It must NOT imply human identity, independent authorization,
or semantic interpretation by the delegate (L8 has no Themis capability
and no Governance access). The view answers "which delegated execution
received which authorized evidence references?", never "who saw the
Finding".

**No runtime amendment:** the runtime already holds the authoritative
`l8-delegation` witness.
