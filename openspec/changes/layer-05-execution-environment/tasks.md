# Tasks: Layer 5 — Execution Environment

Grill closed 2026-09-06 (Q-L5-1..12, all CLOSED; design.md §3 is the record, §2 the folded decisions). Execution starts only after owner Gate 0. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [x] Grill session held; Q-L5-1..12 answered and recorded (12 closures committed)
- [x] Decisions folded into design.md §2 (D-L5-1..10)
- [ ] Gate 0: design accepted by owner

## 1. L5-M1 — Envelope artifacts + environment seam + local provider (Class 3)

- [ ] WorkspaceExecutionCeiling + ProvisionSpec governed artifacts (D-L5-2: floors/governed/intrinsic categories, spec ⊆ ceiling, pinned SHA, fail-closed loaders)
- [ ] `Environment` seam with declared-property provider contract (D-L5-1: identity/network/host-services/limit strengths; spec-vs-declaration admission fails closed)
- [ ] Local provider: mirror-clone provisioning as audited execution (D-L5-3: neutralization floors, empty-env allowlist, post-condition verification, binary attestation, non-elevation checks, process group)
- [ ] Lifecycle state machine with SEALED (D-L5-9); typed transition trace events; reachability tests
- [ ] Teardown with verified DESTROYED / TEARDOWN_ANOMALOUS split; host-cleanliness assertions
- [ ] Security review

## 2. L5-M2 — Confinement modes + executor migration + limits (Class 3)

- [ ] ResolveMode/CreateMode single implementation (D-L5-4); escape suite (symlink, `..`, absolute, `.git*`)
- [ ] L4 executors re-rooted on the environment; decision-table rerun inside
- [ ] Limit dimensions per the Q-L5-9 matrix (typed breach terminations for enforced; rusage accounting for observed); env-allowlist CI lint for in-process executors
- [ ] Security review

## 3. L5-M3 — Mutating tools + egress + store (Class 3)

- [ ] registry-v2: write_file, apply_patch (`mutating: true` grants, transactional patch, {path, old hash, new hash} records)
- [ ] Artifact Egress Contract (D-L5-5: diff-against-pinned-base, structural gate, no partials, observed-breach gate)
- [ ] ArtifactStore: write-once content-addressed local implementation; acknowledgment → teardown ordering; no-push structural proof
- [ ] Security review

## 4. L5-M4 — Repository-instruction activation (Class 3)

- [ ] Identity-bound registration artifact; four-control eligibility chain wired to L1 (D-L5-8, root AGENTS.md only)
- [ ] Unregistered-file invisibility + symlink-refusal tests
- [ ] Security review

## 5. L5-M5 — Operational proof + close (Class 2)

- [ ] Live proof per the design.md §4 proof gate (six stages, clean + failure paths)
- [ ] Traceability, coverage-verified; reviews with three-state verdicts
- [ ] Code map + status doc + artifact updates; push/archive on owner approval

## 6. Deferred (NOT L5 scope — recorded IOUs)

- run_command (OPEN-2; preconditions: stronger network/host-service isolation, identity-residual revisit)
- Docker/remote providers beyond the seam; browser; dependency installation
- Credential broker implementation (seam only); in-process secret capability design
- ArtifactStore retention/GC and durable lifecycle (L6); L7 provisioning owner
