# Tasks: Layer 9 — Skills & Procedures

Grill closed 2026-09-08 (all questions CLOSED; design.md §3 is the record, §2 the folded decisions D-L9-0..17 + residuals). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally-proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held; boundary + all questions answered and recorded (incl. Q-L9-7′ instantiation surface taken early)
- [x] Decisions folded into design.md §2; fold-completeness cross-check: no orphans; two architecture-doc fields dispositioned (failure conditions = workflow edges; expected outputs flagged OPEN to owner)
- [x] Gate 0: design accepted by owner (2026-09-08, "GATE 0 — PASSED / IMPLEMENTATION AUTHORIZED"; expected-outputs v1 omission accepted as residual; M3 constraint: registration is the governed admission act, machinery must not self-register; M2 constraint: L7 attribution amendment non-semantic)

## 1. L9-M1 — Skill manifest + catalog (Class 3)

- [x] Skill manifest artifact + fail-closed loader (skills/manifest.go): closed schema, seven mandatory pins, hash-format validation, no skill-reference field; pins resolve through confine.ResolvePath so a symlinked pin cannot read outside the bundle
- [x] Catalog (skills/catalog.go): immutable name@version→composition bindings, two-way identity agreement, rebind refusal, active|withdrawn, atomic resolve-verify-instantiate, CheckAppendOnly detects deletion/rebinding/un-withdrawal
- [x] Input-schema validator (skills/schema.go): closed v1 vocabulary; rich features unrepresentable via DisallowUnknownFields; string bounds mandatory; integer inputs bounded
- [x] Catalog root disjointness at the instantiation call site (state.CheckDisjointRoots); decoded registry scan proves no catalog verb (TestNoCatalogVerbInRegistries)
- [x] Security review (2026-09-08: no CRITICAL; HIGH-1 arbitrary-write + MED-1 wall-2 vacuity remediated with regressions)

## 2. L9-M2 — Instantiation (Class 3)

- [x] Resolve exact name@version → verify ACTIVE → resolve all pins → byte-verify before anything is written (skills/instantiate.go)
- [x] Closed instantiation surface: Class-2 downward narrowing (quotas incl. the aggregate cap, wall deadline), Class-3 task-state inputs (schema-validated, external-untrusted), Class-4 deployment; unknown fields unrepresentable
- [x] Placeholder-only substitution (@task_id/@repo/@pinned_sha; @workspace stays for L7 to bind); never procedure text/workflow/contract/ceiling. NOTE: `@input.<field>` is NOT implemented — inputs travel via the payload only; recorded as a deliberate v1 narrowing for owner review
- [x] Effective grant + spec emission with template AND instantiated/effective identities in origin; spec re-validated through execution.LoadSpec; grant shape-checked (it cannot be loaded pre-workspace by design — L7 records grant_effective after binding)
- [x] L7 additive envelope amendment: optional non-semantic skill_procedure_path/sha256 + opaque origin, preserved verbatim into attribution under an origin: prefix; L7 interprets none of it
- [x] Security review (2026-09-08: HIGH-2 render trust-merge + MED-2 silent procedure drop remediated; MED-3 re-Lstat added)

## 3. L9-M3 — investigate-cve@1 (Class 2/3)

- [x] P0 skill composition authored (policies/skills/investigate-cve/): two-phase lattice (ANALYZE read-only, ASSESS adds write_file), ceiling, contract, grant template, spec template, closed input schema, advisory procedure artifact
- [ ] Governance registration in the catalog (OWNER ACT — written as catalog.proposed.json; the machinery has no write API and never self-registers)
- [x] Security review (2026-09-08: P0 bundle pinned to the v1 capability set; ANALYZE proven read-only against the registry's mutating flag)

## 4. L9-M4 — Proof registers + close (Class 2/3)

- [x] Register A structural (skills/skills_test.go) · Register B adversarial (skills/adversarial_test.go) · Register C equivalence + provenance-swap (orchestration/skill_seam_test.go) · Register E live proof PASSED vs qwen2.5:7b through the unmodified loop (orchestration/skill_e2e_test.go), plus the zero-trust-discount bypass proven against the real skill
- [x] Register D cold reconstruction (orchestration/skill_reconstruct_test.go): record + catalog + pinned bytes recover the exact reviewed composition; declared-vs-recorded artifact identity drift is deterministically detectable after the fact
- [x] Three Class-3 reviews run and remediated (security, test, architecture); every CRITICAL and HIGH closed with a mutation-verified regression
- [ ] Traceability; owner acceptance of the three-state verdicts
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 5. Deferred (recorded residuals — design.md §3; never silently promoted)

- Governed multi-Skill pipelines (L8-vs-L9 OPEN) · L8 delegation · runtime revocation/cancellation · approval channel · repository/external skill activation · model-property predicate · richer input schema · evaluation-purpose attribution (L11/L6) · proposal lifecycle (L11) · deterministic verification capabilities (OPEN-2/L10) · expected-outputs contract (Gate 0: accepted v1 omission; future owner unassigned)
