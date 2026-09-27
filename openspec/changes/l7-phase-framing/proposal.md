# L7 phase framing — the model cannot know its phase (I-M5 finding, 2026-09-27)

## The fact that forced this

The first multi-phase live walks with the real Themis read door, under
`rsys@6` on the host (`demo-remediate-0001` qwen2.5:7b,
`demo-remediate-0002` gpt-oss:20b), ended as governed FAILED at the
ANALYZE → REMEDIATE boundary. Reproduced on the laptop and on the host
with `integration/livecapture_test.go` (provider request + raw response
per turn):

- D-L7-11 composes a FRESH conversation at every phase entry: the EIS
  render (identical each phase) plus one L2 slot, the task brief. The
  model is never told which phase it is in, which phases are done, or
  which capabilities this phase carries. Nothing read in ANALYZE (the
  Finding, governed-record) reaches REMEDIATE.
- `remediate-dependency@4`'s procedure is one text opening with "first
  call get_finding". A phase-fresh model restarts the method in
  REMEDIATE, where `get_finding` is not granted. qwen2.5:7b's attempt is
  dropped by the provider (empty content → `turn-no-action` ×4 →
  `@fail`); gpt-oss:20b reasons it out loud ("instructions mention
  get_finding but not provided as a tool") and falls back to erroring
  `read_file` calls (`tool-error` ×4 → `@fail`).
- Not the cause: context window (`truncated = 0`), tool-result wire
  shape, Ollama version, key or door (the Finding was read, twice).
- A phase-scoped procedure alone does not move qwen2.5:7b (it then
  declares done without editing); an explicit phase statement in the
  system or user text alone does not either. The fix is structural,
  then textual.

Everything Themis-facing worked: `rsys@6` opened, commission in
origin, read door over loopback with the read key, clean seal and
teardown. The two records are inspectable governed failures, not
errors.

## Scope of this grill (Q-P-1..6)

Q-P-1 where the phase fact lives · Q-P-2 what it says · Q-P-3 evidence
continuity across phases (B) · Q-P-4 `remediate-dependency@5` and the
anchor that follows · Q-P-5 the model allowlist for the demo · Q-P-6
the `search_code` denial seen at ANALYZE turn 4 (minor).

Out of this grill, tracked elsewhere: two-VM topology (D-I-1 amendment),
D-I-9 subject-bound scope, Themis product-scope write confinement (row 14).

## Rule while open

`rsys@6` stays ACTIVE (it produced the evidence); `rsys@5` is not
withdrawn; no host act until the amendment lands and `rsys@7` is
prepared the same way `rsys@6` was.
