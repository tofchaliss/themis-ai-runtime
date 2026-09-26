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

## D-L-2 — No delegation-specific proposal-admissibility rule (LOCKED 2026-09-26, owner)

> Delegation affects proposal evidence and human review, but creates
> no independent Themis admissibility condition while delegates remain
> tool-less and without Governance access. Themis admits on the
> existing commission, production, verification, and trust contracts;
> it does not evaluate runtime reasoning structure.

```
Commission {Finding, skill@version + composition, deployment anchor}
   → runtime governs delegation reachability (grant; anchor-pinned template registry)
   → execution record {production chain, verification}
   → proposal admissibility
```

Three decisive reasons:
1. **Delegation permission is already inside the commissioned
   identities** (method → grant → `delegate`; deployment → anchor →
   template registry). A Themis rule "verify template X was permitted"
   would duplicate runtime governance in Themis.
2. **Invalid delegation is already observable runtime evidence:** the
   seam refuses and `l8-delegation` records the outcome; visible in the
   D-L-1 rendering. Visible ≠ an automatic admissibility condition.
3. **Themis must not interpret runtime reasoning structure.** Themis
   establishes what was commissioned, what executed, whether the
   production chain exists, whether verification is reproducible, and
   the evidence class. It never establishes "the model reasoned
   correctly because delegate X saw evidence Y".

Explicitly NOT rules: `delegate failed → proposal refused`;
`delegate received Finding evidence → proposal refused`. Both would
turn an execution detail or a reasoning path into Governance policy
without an established decision. Instead: runtime evidence → D-L-1
rendering → human proposer/decider → human judgment (B-T-2:
admissibility mechanical, correctness human).

**Forward-compatibility invariant (owner, explicit):** D-L-2 is valid
only while the locked L8 tool-less / no-Governance-access boundary is
in force. If a future L8 architecture grants delegates Themis or other
governed capabilities, D-L-2 must be revisited, because commission
correspondence alone would no longer characterize the delegate's
authority surface. Not a reopening of L8 now.
