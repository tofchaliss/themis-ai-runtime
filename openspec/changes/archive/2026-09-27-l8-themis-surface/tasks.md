# Tasks: L8 × Themis surface (folded into I-M3)

Grill CLOSED 2026-09-26 (D-L-1..3). Themis-side only; no runtime change.

- [x] `themis-intake` evidence view: delegations rendered within the
      model-reasoning fact (seq, parent call seq, template name@version,
      model identity, evidence refs + authority class, output object id,
      outcome); wording per D-L-1 (never "who saw")
- [x] `harness-execution/v1.delegations {count, seqs[]}`; by reference only
- [x] No admissibility rule touches delegations (D-L-2); test
      `TestRefusedDelegationIsRenderedNotJudged` (Themis, 2026-09-27): a forged
      record with a `provider-error` delegation and a valid chain is admitted,
      the refusal renders by identity, the evidence cites it by seq. Found and
      fixed on the way: `delegationOf` decoded `template.name`/`model_identity.model`,
      which the runtime never emits — now `template.ref` and
      `model_identity.governed.name` (the fixture records carry no delegation,
      so nothing had exercised it). Original wording: a record
      with a refused delegation and a valid production/verification chain
      is admitted; the refusal is visible in the rendering
- [x] Witnessing-constitution table entries carry `premises`; seeded
      `l8-delegates-tool-less`; test asserts every entry names it (D-L-3)
