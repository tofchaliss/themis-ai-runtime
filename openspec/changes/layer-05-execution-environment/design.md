# Design: Layer 5 — Execution Environment

**Inputs:** `docs/architecture/harness/00-p0-architecture-v2.md` §8 (layer doc), the archived L1–L4 designs (their deferrals land here: Q-L1-1 pinned-ref provenance, L4's sandbox/mutating-tool/TOCTOU deferrals), `ARCHITECTURE.md`, Stage-3 §3.5/3.6 (secret + execution isolation).
**Decision IDs** `D-L5-n`; **open questions** `Q-L5-n`. DEC-05 applies throughout.

## 0. Position in the flow

```
Task ──► L7 (future) ──provision──► ┌──────────────────────────┐
                                    │ L5 Environment           │
                                    │  worktree @ pinned ref   │
                                    │  resource limits         │
                                    │  no network (default)    │
                                    │  isolated process exec   │
                                    └───────────┬──────────────┘
                                                │ is the confinement root for
                                                ▼
                          L2 filesystem/search sources · L4 executors
                                                │
                                                ▼
                          worktree changes = task artifacts
                          (governed hand-off, never direct push)
```

## 1. Hard invariants (inherited, not grillable)

- **Execution isolation (3.6):** tool execution happens inside the provisioned environment; the model never touches the host; harness host state is not reachable from executors.
- **Secret isolation (3.5):** no credential ever enters model context; executors obtain scoped short-lived credentials via the broker seam, never from env vars visible to tool output.
- **Provenance pin:** every workspace is {repo, ref, hash}-recorded; repository-scope instruction activation (if any) loads from exactly that pin (archived Q-L1-1 contract).
- **Same L4 gate:** mutating tools pass the identical registry/grant/target authorization — the sandbox adds containment, never a second permission system.
- **Worktree = blast radius:** all writes confined to the provisioned worktree; teardown restores the host; nothing pushes to shared remotes from inside.
- **Fail closed:** provisioning failure ⇒ no execution; limit breach ⇒ typed termination, recorded; unknown provider ⇒ refuse.
- **L5 failure is bounded:** it can deny the agent a computer or kill its processes — it cannot grant authority, and its absence must never silently widen confinement.

## 2. Draft decisions (grill targets)

### D-L5-1 — Environment seam, local provider first
`Environment` interface: {Provision(spec) → Workspace, Exec(req) → result, Teardown}. v1 provider: local git worktree + OS process isolation (separate process, rlimits, cwd-jailed, cleared env). Docker provider second, same seam. Provider choice is configuration, never code branching (DEC-05 posture applied to execution).

### D-L5-2 — Workspace spec is a governed artifact
{repo, ref (commit SHA — not a branch name), allowed byte/time/proc limits, network: none} — versioned, hashed into the trace. Branch names resolve to SHAs at provision time and the SHA is what's recorded.

### D-L5-3 — Executor migration
L4's filesystem executors re-root onto the provisioned workspace; the confinement root becomes the environment's boundary rather than a caller-supplied path. TOCTOU closes structurally where the provider supports it (process cwd jail / container mount), with `ConfinePath` retained as defense in depth.

### D-L5-4 — Mutating tools (write_file, apply_patch)
Registered in a v2 registry under the same schema/target/grant discipline; grants for mutating tools carry explicit `mutating: true` visibility for review; results record {path, old hash, new hash}. Worktree diff is the task's artifact.

### D-L5-5 — Hand-off, not push
Task completion emits the worktree diff + provenance as an artifact for governed review (Themis/human). The environment has no push credential and no remote write path. (Where the diff goes is L6/L7-era; the *impossibility of direct push* is L5's.)

### D-L5-6 — Credential broker seam only
Interface defined ({scope, ttl} → credential), no implementation; v1 provisions credential-less environments. Any future credentialed executor is a Class-3 registration.

### D-L5-7 — run_command stays out
Even sandboxed, arbitrary shell is OPEN-2's own grill (command policy, allowlists, output classification). The sandbox is necessary but not sufficient.

## 3. Open questions for the grill (Q-L5-n)

1. **Q-L5-1 — Isolation strength for v1:** is OS-level process isolation (separate process, rlimits, cleared env, cwd jail) an acceptable first provider, or is Docker mandatory before any mutating tool activates? (Hardware/OPEN-3 constraints apply.)
2. **Q-L5-2 — Network default:** none-by-default is proposed; is *any* v1 executor allowed egress (e.g. future advisory fetch), and is network a per-tool grant dimension or an environment-level switch?
3. **Q-L5-3 — Who provisions:** L7 will own provisioning eventually; v1 has no L7 — does the test harness/caller provision directly, and is that seam shaped now?
4. **Q-L5-4 — Repository instruction activation:** does activating `ScopeRepository` sources (per the archived Q-L1-1 contract) belong in this change, or its own follow-up once pins exist?
5. **Q-L5-5 — Mutating-tool review surface:** what makes a mutating grant reviewable enough — `mutating: true` flags, separate grant artifact, or a distinct registry section?
6. **Q-L5-6 — Limit semantics:** what do CPU/RAM/time limits mean per provider, what's typed to the model on breach (`error(timeout)` precedent), and what's trace-only?
7. **Q-L5-7 — Teardown guarantees:** what is *asserted* clean (worktree removed, temp dirs, processes reaped) vs best-effort, and what happens on teardown failure (fail closed how)?
8. **Q-L5-8 — Diff artifact shape:** what exactly is the governed hand-off object {diff, provenance, tool audit trail}?
9. **Q-L5-9 — Operational proof:** proposed — live model drives a mutating call (`write_file`) inside a provisioned worktree; assertions: change exists in worktree, host untouched, escape attempts refused, teardown clean.

## 4. Test plan (three-state discipline)

Provision/teardown determinism + host-cleanliness assertions; escape suite (symlink, `..`, absolute, race where testable) against the environment boundary; limit-breach typed terminations; mutating-tool old/new hash records; no-push structural proof (no credential, no remote); pinned-ref recording; the L4 decision-table rerun inside the environment; live proof per Q-L5-9.
