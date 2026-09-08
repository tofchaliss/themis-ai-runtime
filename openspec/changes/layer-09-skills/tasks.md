# Tasks: Layer 9 — Skills & Procedures

Grill closed 2026-09-08 (all questions CLOSED; design.md §3 is the record, §2 the folded decisions D-L9-0..17 + residuals). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally-proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held; boundary + all questions answered and recorded (incl. Q-L9-7′ instantiation surface taken early)
- [x] Decisions folded into design.md §2; fold-completeness cross-check: no orphans; two architecture-doc fields dispositioned (failure conditions = workflow edges; expected outputs flagged OPEN to owner)
- [ ] Gate 0: design accepted by owner

## 1. L9-M1 — Skill manifest + catalog (Class 3)

- [ ] Skill manifest artifact + fail-closed loader: closed schema, mandatory pins (workflow, ceiling, contract, grant template, spec template, input schema, procedure ref), hash-format validation, no skill-reference field (D-L9-1/12)
- [ ] Catalog: append-only name@version→composition-hash bindings; manifest/catalog two-way version agreement; rebind refusal; active|withdrawn states; atomic resolve-verify-instantiate (D-L9-3/10)
- [ ] Input-schema validator: closed v1 vocabulary (type/required/max_length/enum), rich features refused (D-L9-4)
- [ ] Catalog root disjointness at the designed call site; no catalog verb in any registry (decoded scan) (D-L9-8)
- [ ] Security review

## 2. L9-M2 — Instantiation (Class 3)

- [ ] Resolve exact name@version → verify ACTIVE → resolve all pins → verify bytes against hashes BEFORE executable (D-L9-10/11)
- [ ] Closed instantiation surface: Class-2 downward narrowing (quotas, wall deadline), Class-3 task-state inputs (schema-validated, external-untrusted), unknown fields refused (D-L9-7)
- [ ] Closed placeholder vocabulary (@workspace/@task_id/@input.<field> into declared-substitutable fields only; never procedure text/workflow/contract/ceiling) (D-L9-4/7)
- [ ] Effective grant + spec emission (template hash AND effective hash recorded); ordinary governed envelope with mandatory opaque skill-provenance attribution (D-L9-7/13)
- [ ] L7 additive envelope amendment: non-semantic attribution field, preserved verbatim into task attribution; L7 remains skill-blind (Class 2/3, per D-L9-13)
- [ ] Security review

## 3. L9-M3 — investigate-cve@1 (Class 2/3)

- [ ] Author the P0 skill composition: workflow (analysis lattice on read_file/write_file/declare_done), procedure artifact, contract, grant template, spec template, input schema
- [ ] Governance registration in the catalog (owner act)
- [ ] Security review

## 4. L9-M4 — Proof registers + close (Class 2/3)

- [ ] Register A structural · Register B adversarial (incl. zero-trust-discount L9-bypass) · Register C equivalence + provenance-swap · Register D cold reconstruction · Register E live proof (unmodified production loop, qwen2.5:7b) with negative proofs
- [ ] Traceability, coverage-verified; reviews with three-state verdicts
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 5. Deferred (recorded residuals — design.md §3; never silently promoted)

- Governed multi-Skill pipelines (L8-vs-L9 OPEN) · L8 delegation · runtime revocation/cancellation · approval channel · repository/external skill activation · model-property predicate · richer input schema · evaluation-purpose attribution (L11/L6) · proposal lifecycle (L11) · deterministic verification capabilities (OPEN-2/L10) · expected-outputs contract (owner decision pending)
