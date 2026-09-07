# Tasks: Layer 5 — Execution Environment

Grill closed 2026-09-06 (Q-L5-1..12, all CLOSED; design.md §3 is the record, §2 the folded decisions). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held; Q-L5-1..12 answered and recorded (12 closures committed)
- [x] Decisions folded into design.md §2 (D-L5-1..10)
- [x] Gate 0: design accepted by owner (2026-09-06)

## 1. L5-M1 — Envelope artifacts + environment seam + local provider (Class 3)

- [x] WorkspaceExecutionCeiling + ProvisionSpec governed artifacts (D-L5-2: floors/governed/intrinsic categories, spec ⊆ ceiling, pinned SHA, fail-closed loaders)
- [x] `Environment` seam with declared-property provider contract (D-L5-1: identity/network/host-services/limit strengths; spec-vs-declaration admission fails closed)
- [x] Local provider: mirror-clone provisioning as audited execution (D-L5-3: neutralization floors, empty-env allowlist, post-condition verification, binary attestation, non-elevation checks, process group)
- [x] Lifecycle state machine with SEALED (D-L5-9); typed transition trace events; reachability tests
- [x] Teardown with verified DESTROYED / TEARDOWN_ANOMALOUS split; host-cleanliness assertions
- [x] Security review (5 MED + 4 LOW found, all remediated in 3c82720 with regressions)

## 2. L5-M2 — Confinement modes + executor migration + limits (Class 3)

- [x] ResolveMode/CreateMode single implementation (D-L5-4); escape suite (symlink, `..`, absolute, `.git*`)
- [x] L4 executors re-rooted on the environment; decision-table rerun inside
- [x] Limit dimensions per the Q-L5-9 matrix (typed breach terminations for enforced; rusage accounting for observed); env-allowlist CI lint for in-process executors
- [x] Security review (M2+M3 combined: 1 HIGH case-fold .git* bypass + 2 MED + 2 LOW, all remediated in befcc33 with regressions)

## 3. L5-M3 — Mutating tools + egress + store (Class 3)

- [x] registry-v2: write_file, apply_patch (`mutating: true` grants, transactional patch, {path, old hash, new hash} records)
- [x] Artifact Egress Contract (D-L5-5: diff-against-pinned-base, structural gate, no partials, observed-breach gate)
- [x] ArtifactStore: write-once content-addressed local implementation; acknowledgment → teardown ordering; no-push structural proof
- [x] Security review (covered by the combined M2+M3 review; egress/store findings remediated in befcc33)

## 4. L5-M4 — Repository-instruction activation (Class 3)

- [x] Identity-bound registration artifact; four-control eligibility chain wired to L1 (D-L5-8, root AGENTS.md only)
- [x] Unregistered-file invisibility + symlink-refusal tests
- [x] Security review (1 MED sealed-files + 3 LOW, remediated in 79e0205; INFO residuals recorded)

## 5. L5-M5 — Operational proof + close (Class 2)

- [x] Live proof per the design.md §4 proof gate (PASSED vs qwen2.5:7b, 7.84s) (six stages, clean + failure paths)
- [x] Test review: 2 HIGH evidence gaps (rollback, persistence-failure) + MED/LOW remediated in ab1632c; coverage 88.6%% execution, 97.8%% instructions
- [x] Architecture review: 15 findings — 2 MED deviations remediated with mechanisms (f3858d4: OS-level seal, mem_bytes gate), amendments 1-7 recorded in design.md; everything else CONFORM

- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 6. Deferred (NOT L5 scope — recorded IOUs)

- run_command (OPEN-2; preconditions: stronger network/host-service isolation, identity-residual revisit)
- Docker/remote providers beyond the seam; browser; dependency installation
- Credential broker implementation (seam only); in-process secret capability design
- ArtifactStore retention/GC and durable lifecycle (L6); L7 provisioning owner
