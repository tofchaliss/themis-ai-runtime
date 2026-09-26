# Tasks: L8 × Themis surface (folded into I-M3)

Grill CLOSED 2026-09-26 (D-L-1..3). Themis-side only; no runtime change.

- [ ] `themis-intake` evidence view: delegations rendered within the
      model-reasoning fact (seq, parent call seq, template name@version,
      model identity, evidence refs + authority class, output object id,
      outcome); wording per D-L-1 (never "who saw")
- [ ] `harness-execution/v1.delegations {count, seqs[]}`; by reference only
- [ ] No admissibility rule touches delegations (D-L-2); test: a record
      with a refused delegation and a valid production/verification chain
      is admitted; the refusal is visible in the rendering
- [ ] Witnessing-constitution table entries carry `premises`; seeded
      `l8-delegates-tool-less`; test asserts every entry names it (D-L-3)
