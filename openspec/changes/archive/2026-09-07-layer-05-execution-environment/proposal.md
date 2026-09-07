# Proposal: Layer 5 — Execution Environment

**Change ID:** layer-05-execution-environment · **Status:** GRILL CLOSED (Q-L5-1..12), awaiting Gate 0 · 2026-09-06
**Owner gate:** ships only after owner Gate-0 acceptance of the folded design (design.md §2 D-L5-1..10, §3 grill record).

## Why

Four layers of accumulated IOUs point here. L1 defers repository/directory instruction sources to "pinned-ref provenance machinery" (Q-L1-1 activation contract). L4 defers every mutating tool, `run_command`, and process isolation to "the L5 sandbox." The confinement TOCTOU is recorded as "the L5 threat model's problem." Today the harness reads a workspace the host process happens to see; L5 makes the workspace a *provisioned, isolated, provenance-pinned* place — the safe computer the layer doc demands — and thereby unlocks the write side of the tool vocabulary.

## What

An execution-environment layer at `src/harness/execution`:

- **Workspace provisioning:** git worktree at a **pinned, recorded ref** (the Q-L1-1 provenance machinery), provisioned per task, destroyed after; workspace identity {repo, ref, worktree path, hash} in the trace
- **Execution abstraction** (layer doc): an `Environment` seam with a local provider first (worktree + OS-level process isolation for tool executors); Docker as the second provider behind the same seam; no hardcoded provider (Daytona/E2B-shaped remotes possible later)
- **Sandboxed executors:** L4's dispatch gains an execution boundary — tool processes run inside the environment with resource limits (CPU/RAM/time), no network by default, and a filesystem view that IS the confinement root (closing the TOCTOU class structurally)
- **Mutating tools, behind the same L4 gate:** `write_file`, `apply_patch` scoped to the worktree; changes exist only in the isolated worktree until a governed hand-off
- **Credential discipline:** no secrets in model context ever (standing 3.5); the credential-broker *seam* defined (short-lived scoped credentials fetched by executors, never by the model); v1 likely needs zero credentials (local git + local fs)
- **Repository instruction activation:** with pinned refs available, `ScopeRepository` sources can activate per the archived L1 contract — registration + provenance + scope cap + pattern gate

## What this change does NOT do

- No `run_command` until OPEN-2 (command policy) is grilled — even inside the sandbox, shell is its own question
- No remote providers, no browser (layer doc: later); no dependency installation flows in v1
- No git push/commit to shared remotes from the environment — worktree changes are task artifacts pending governed review (the hand-off design is part of the grill)
- No L6 persistence, no L7 loop

## Success criteria

A task's entire filesystem effect is provably confined to its provisioned worktree; environment teardown leaves the host clean (asserted, not assumed); mutating tools cannot touch anything outside the worktree even via symlink/race games; workspace provenance {repo, ref} is recorded and the ref is what L1 repository sources (if activated) load from; the L4 live proof extends to a mutating call inside the sandbox; three-state verdicts per milestone.
