# L10 Amendment: seam refusal hygiene + torn marker (integration-audit D11/D12, 2026-09-13)

Additive implementation corrections; no constitutional change.

- D11: refusal details surfaced to the model strip absolute
  filesystem paths (sanitizeDetail) — the reason class survives,
  host layout does not.
- D12: ReconstructTask returns a torn flag from the task verdict —
  a reconstruction over a TORN task now carries the marker
  (committed prefix remains valid; the tail's expulsion is visible
  to consumers).
