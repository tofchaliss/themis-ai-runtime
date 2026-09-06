# Tasks: Layer 5 — Execution Environment

Execution starts only after the grill closes Q-L5-1…9 and the owner accepts. Three-state verdicts per milestone: architecture-conformant · test-evidenced · operationally proven. An editing command or commit message is never evidence.

## 0. Gate

- [ ] Grill session held; Q-L5-1..9 answered and recorded
- [ ] Design accepted by owner
- [ ] Owner decisions surfaced by the grill: isolation strength (Q-L5-1, may interact with OPEN-3 hardware), network default (Q-L5-2)

## 1. L5-M1 — Environment seam + local provider (Class 3)

- [ ] `Environment` interface + workspace spec artifact (pinned SHA, limits, no network)
- [ ] Local provider: worktree provisioning, process isolation, teardown with cleanliness assertions
- [ ] Escape suite against the boundary; security review

## 2. L5-M2 — Executor migration + limits (Class 3)

- [ ] L4 executors re-rooted on the environment; decision-table rerun inside
- [ ] Limit enforcement + typed breach terminations; security review

## 3. L5-M3 — Mutating tools (Class 3)

- [ ] registry-v2: write_file, apply_patch (mutating-visible grants per Q-L5-5)
- [ ] old/new hash records; worktree-diff artifact (Q-L5-8); no-push structural proof
- [ ] Security review

## 4. L5-M4 — Proof + activation (Class 2)

- [ ] Live proof per Q-L5-9 (mutating call in a provisioned worktree, host untouched, teardown clean)
- [ ] Repository-instruction activation per Q-L5-4 decision (or recorded follow-up)
- [ ] Traceability, coverage-verified

## 5. Deferred (NOT L5 scope)

- run_command (OPEN-2 grill), Docker/remote providers beyond the seam, browser, dependency installation, credential broker implementation, L6 persistence, L7 provisioning owner

## 6. Close

- [ ] Reviews with three-state verdicts; code map + artifact updates; push/archive on owner approval
