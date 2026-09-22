# RESUME-HERE — skill-admission implementation (SA-M1..M6)

Updated at every green milestone. If context was compacted, start here.

## One-line status
2026-09-22: SA-M1..M5 implemented and green in targeted tests
(`orchestration/skill_admission_test.go`, `tools/instantiate_test.go`,
`execution` TestSpecInstantiates); full suite running; M6 (live
anchored proof, reviews, archive) pending. Baseline had one
pre-existing failure: `TestLiveWalkProof` (ollama reachable, failed
after 120s — under investigation, see gaps).

## Decisions a newcomer must not re-derive
`design.md` §2 D-SA-1..10 (locked), §5 whitelist + twin suite, §6
A-SA-1..11. Do not reopen architecture during SA-M1..M6 absent a
category-4 gap (owner stop rule).

## Gate 1 implementation design (against design.md §5)
- **M1 envelope/L9/L7:** `Envelope.Skill` (`json:"skill,omitempty"`,
  exact `name@version` regex local to orchestration — L7 imports no
  skills semantics); `LoadEnvelope` coherence: `skill` ⇒ commitment
  required; any `skill`/`skill_*` origin key ⇒ `skill` field required
  and `origin["skill"] == skill` when present. L9 `Instantiate` emits
  `"skill"`. `SubmitTask` anchored: `skill == "" && skill_procedure_path
  != ""` ⇒ `ErrAssembly` (D-SA-3). `verifyAnchoredSkill` selects by
  `env.Skill`. Tests in `orchestration/skill_admission_test.go`.
- **M2 anchor:** `Anchor.Skills []string` (`json:"skills"`; absent =
  none admitted; entries exact refs, unique, ≤64). `SubmitTask`
  anchored: `env.Skill != "" ⇒ skill ∈ a.Skills` right after the
  registry pin check, before the bundle loop (ladder order). `rsys@4`
  is the owner's Governance act; test anchors via `anchorWorld` mutate.
- **M3 correspondence:** `verifyAnchoredSkill` compares the manifest's
  seven pins with the commitment's seven fixed fields, per member,
  distinct messages, keep the phrase "never its constituent hashes"
  (phasec C17 expects it); drop `Seal != entry.Composition`; return the
  resolved `*skills.Manifest` + catalog for M4/M5. AST wall: no `.Seal`
  reference in orchestrator.go.
- **M4 instantiation:** `tools.ParseGrantTemplate(raw)` (placeholders
  `@task_id`/`@workspace` allowed), `GrantEntry.TemplateScope`,
  `tools.Instantiates(effective, template)` per D-SA-4 table;
  `execution.SpecInstantiates(eff *Spec, templateRaw)`; L7 applies both
  after reading `rawGrant` / `spec` when anchored + skill; L9 mirrors
  `Instantiates` after `instantiateGrantTemplate`;
  `grantAuthorityDigest` adds `scopeDigest(e.TemplateScope)`;
  `Manifest.ResolvePin(name)` exported for L7's confined template read.
- **M5 Claim 2 bytes:** `Catalog.Raw`, `Manifest.Raw`; in `SubmitTask`
  anchored+skill: `materialized["skill_catalog"]`,
  `materialized["skill_manifest"]`; `governed["skill"] = env.Skill`.
- **M6 twins/live:** anchored world pinning the real catalog +
  investigate-cve@1 bundle; positive = L9-instantiated envelope
  ACCEPTed; negatives A-SA-1..10 each asserting its gate's message;
  mutation coupling; live run needs a local model (gap if absent).

## Milestone log
- [x] M1  - [x] M2  - [x] M3  - [x] M4  - [x] M5 — full suite green minus live registers (2026-09-22)
- [ ] M6: live anchored walk (running), three Class-3 reviews, mutation probes, amendment records, archive

## Gaps found while testing
1. **Pre-existing:** `TestLiveWalkProof` fails (not skips) with ollama
   reachable at localhost:11434 — 120s; reason being captured
   (scratchpad live-walk.log). Not caused by this change.
2. **Fixture debt:** four existing tests attached `origin.skill` with no
   load-bearing `skill` field; patched to carry the selector (D-SA-5),
   rule not loosened. `TestAnchoredSkillRequiresConfiguredCatalog`
   needed the skill in the anchor allowlist to reach the catalog gate
   (ladder order D-SA-9).
3. **Relation precision:** the spec relation must normalize an absent
   template `strength` to the loader default (enforced) — found by the
   positive twin, fixed in `execution.SpecInstantiates`.
4. **L9 mirror ordering:** `tools.Instantiates` must run after
   `checkGrantShape` or it preempts the substitution-site refusal
   message an existing adversarial test asserts.
5. **Type-checked wall:** a textual `.Seal` scan matched the L5
   environment `Seal` method; the D-SA-6 wall uses go/types.
6. **Object id format:** materialized object ids carry a `sha256:`
   prefix; test assertions must too.
7. **Live-walk cause (pre-existing):** `TestLiveWalkProof`'s negative
   half asserts the live model CALLS an ungranted `declare_done` so the
   denial lands in the record; qwen2.5:7b did not call it, so no
   `l4-audit` denial exists. The test couples a governed property to
   model behaviour — the very thing the live registers say not to do.
   Not touched by this change; recorded for the L7 test owner.
8. **Scripts:** `themis-status`/`themis-preflight` never read an anchor
   file (they print repo-derived pins), so "print skills[]" has no
   host today; deferred to the `rsys@4` act.
9. **Claim 2 reconstruction view:** the catalog/manifest bytes are now
   retained per task, but no L10-side view re-derives "ran the
   registered composition" from them yet — the twin asserts retention,
   not re-derivation.
10. **Evidence module is standalone:** `evidence/harness` is not in
   `go.work`; build/vet it with `GOWORK=off`. C17 patched and builds.
