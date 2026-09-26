# Tasks: L11 ↔ Governance provenance (harness-side; folded into I-M1)

Grill CLOSED 2026-09-26 (D-R-1..4). All changes are in the governed
tree and its loaders; the pin moves land with `rsys@6`.

- [ ] `policies/decisions/` schema (`id, kind, target{name@version, hash},
      evidence[]{criterion, package_object_id, record_root}, rationale,
      decided_at, actor{commit:<author> | key:<KeyID>}`), closed, strictjson
- [ ] Decision-record loader (read-only): existence, hash, `target` two-way
      against the referencing entry
- [ ] Skill catalog `version: 2`: `decision_ref` required on every entry;
      loader refuses missing/mismatched; decision records written for
      the four existing entries from the archived ratifications
- [ ] Anchors registry `version: 2`: same for every anchor entry
- [ ] Wall tests: zero `os` writers in `skills`, `deployment`, decision
      loader; `ratchet`/`themis-ratchet` forbidden importers now name
      `skills` and `deployment`; no-side-effect test around a comparison
      package; no helper that drafts a door entry from an L11 result
- [ ] `themis-status` / preflight print `decision_ref` per active entry
