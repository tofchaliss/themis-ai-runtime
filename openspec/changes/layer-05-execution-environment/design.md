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

## 3. Grill record (2026-09-06)

### Q-L5-1 — Execution-envelope ownership (CLOSED, locked)

Owner-locked five principles:

1. **Three ownership categories:** L5 hard floors · governed execution parameters (Themis ceiling → L7 narrowing) · capability-intrinsic constraints (L4 registry, Class-3).
2. **Hard-floor definition (mechanical):** a hard floor is a security invariant **outside the execution-envelope configuration vocabulary** — no contract/grant/task field can even express weakening it; weakening requires a Class-4 architecture change. v1 floors: network deny, path-resolution semantics (symlink refusal, VCS-metadata write deny-list), environment scrubbing, process identity, I/O capture mechanics. Principle: *capacity is never authority; configuration is never de-isolation.*
3. **Floor migration is an architecture-authority change, not a parameter edit** — it introduces a new expressible capability into the vocabulary and therefore **enters through the authorization boundary**: future network access arrives as a governed capability (Themis ceiling → L7 → L4 authorization → L5 enforcement, per Stage-3 §3.11), never as `L5: network=true`.
4. **Provider capability:** effective capability = requested execution tier ∩ provider isolation capability; incompatible tools are excluded from the dispatch vocabulary before requests exist (closed-vocabulary principle; no runtime provider judgment).
5. **L5 boundary:** L5 may reject an impossible/incompatible environment during provisioning (`environment-provision-failed` — not an authorization denial); after successful provisioning and L4 authorization, L5 only enforces declared constraints, never authorizes. *L5 can refuse an environment; it can never refuse a request.*

Ownership model: Themis ceiling → L7 narrowing → {L4 authorization ∥ L5 environment (floors + provider limits)} → execute. The model supplies the request; governed artifacts supply the binding; L5 supplies the floors; the model overrides none.

### Q-L5-2 — Execution substrate (CLOSED WITH AMENDMENT)

Answer: **C** — pluggable provider seam with a minimum isolation contract; the substrate is defined against the v1 capability vocabulary (Tier 0: in-harness Go executors; Tier 1: exactly one child process — git at provision), never against hypothetical Tier-2 tools. Option A (mandatory subprocess for Tier 0) rejected: our executors are TCB either way and a subprocess adds serialization/management attack surface for little containment — the guarantee is a property contract, not a process count.

**The seven-property contract (owner-amended):**

1. **Filesystem closure** — all reads/writes within provisioned scope via the canonical confinement predicate; for `apply_patch`, the ENTIRE path lifecycle (parse → normalize → resolve → VCS-deny-list → containment → write) obeys the same predicate, application is all-or-nothing (one refused hunk refuses the patch) — an explicit Tier-0 invariant, not "just another write_file".
2. **No ambient authority** — no unprovisioned credentials, environment, host services.
3. **Bounded consumption** — provider-enforced declared dimensions; unsupported-but-required dimensions ⇒ provision failure.
4. **Network deny (v1)** — not expressible in the v1 execution vocabulary.
5. **Captured, attributable I/O** — bounded, classified, never escaping to host output.
6. **Asserted teardown, scoped honestly** — provider-owned post-conditions verified, never assumed; **teardown evidence is scoped to observable provider-owned resources and is never interpreted as proof that no future external side effect can occur** ("container exited" is not a security conclusion — the VCS-persistence vector executes after teardown).
7. **Bounded termination (amendment — "unconditional kill" rejected as overclaim):** each provider declares its termination guarantee. v1 in-process Tier 0: cooperative cancellation required, all bounded loops carry cancellation points, compliance tested — forced termination NOT claimed (a non-cooperative goroutine cannot be killed without killing the process; "our code is cooperative" is an implementation invariant, not a security property). Isolated providers may declare forced termination with an independent kill boundary. **Termination strength is a provider-capability dimension: a spec requiring a strength the provider lacks fails provision closed.**

Tier-2 implication preserved: entering the vocabulary requires registered computation + bounded proposition vocabulary + Q-L2-6 provenance + appropriate tier + satisfying provider. Container ≠ authorization ≠ derived authority ≠ computation registration. Both discovered vectors (provision-time git hooks, post-teardown VCS persistence) are invisible to runtime containers — the recorded reason "container" is never the boundary.

### Q-L5-3 — Provisioning vs execution (CLOSED WITH AMENDMENT)

Owner-locked:

> Provisioning is execution and is never trusted merely because it is infrastructure. Provisioning is authorized by a governed execution ceiling instantiated into a validated execution-specific specification. In v1, the test harness/caller may perform the future L7 orchestration role, but it cannot define or widen the governed ceiling. ProvisionSpec ⊄ WorkspaceExecutionCeiling fails closed before provisioning. **The governed ceiling artifact is a v1 dependency; L7 implementation is not.**

Locked structure: **there is no trusted-by-position execution** — everything that executes runs inside a declared envelope under a closed vocabulary with audit; the planes differ only in authorizer (L4 grant vs governed spec). Two v1 artifacts ship: `WorkspaceExecutionCeiling` {allowed repository universe, ref policy, resource maxima, process limit, deadline maximum, required provider capabilities; version/hash/owner/provenance, fail-closed loader} and `ProvisionSpec` {repository, pinned SHA (immutable commit identity MANDATORY — never a branch), workspace binding, limits, deadline, provider requirements}. Well-formed ≠ authorized: structural validation never substitutes for the subset check. L5 enforces spec ⊆ ceiling mechanically; it never judges whether the ceiling is good policy (jurisdiction ≠ enforcement).

Five provisioning controls: (1) Tier-1 subprocess in its own envelope (forced termination available and required); (2) neutralization profile as hard floor (hooks/config/helpers not expressible); (3) same audit plane as tool calls; (4) post-condition verification (worktree matches pinned SHA, hooks state empty, no unexpected executables); (5) **authorization ceiling before any external effect**. Network floor applies to provisioning: v1 provisions from local mirrors only; remote fetch is a future governed egress capability (§3.11 + the floor-migration rule).

**Cross-layer principle (owner-named, to be recorded architecture-wide):** *a caller/orchestrator may instantiate or narrow a governed ceiling, but cannot define the ceiling* — L2 ContextPlan ⊆ ContextContract · L3 behavior ⊆ ManagementPolicy · L4 ExecutionGrant ⊆ WorkflowCeiling · L5 ProvisionSpec ⊆ WorkspaceExecutionCeiling.

### Q-L5-4 — Filesystem confinement (CLOSED, locked)

The single-mode predicate applied to writes is demonstrably escapable (nonexistent-target lexical fallback + symlinked parent ⇒ out-of-workspace write) — the mode split is mandatory. Owner-locked:

1. **One canonical confinement implementation, two explicit modes** — ResolveMode (reads: shipped semantics) and CreateMode (mutations) — sharing lexical normalization, absolute/`..` rejection, root canonicalization, containment, VCS deny-list, and error construction; differing only where existence semantics genuinely differ. Never `safeReadPath()`/`safeWritePath()` twins.
2. **CreateMode refuses symlinks anywhere in the write path** — including intermediate components and even links resolving inside the workspace; parent chain must exist, fully resolved, non-symlink; final target non-symlink. Symlinks are read-legitimate, write-refused. Monorepo symlink needs trigger a provider-strength/design review, never a silent weakening.
3. **`.git*` deny-list stays deliberately broad** (any component named or prefixed `.git`) and applies to EVERY mutation mechanism — write, delete, rename-from, rename-to, patch. Classifying which `.git*` files are "safe" would itself become a security-maintenance surface. `.gitignore` edits are an accepted v1 usability cost.
4. **`apply_patch` is a transaction:** parse whole patch → validate ALL operations (renames: source resolve-mode + confined + deny-list; destination create-mode; rename-onto-symlink fails before mutation) → any failure applies NOTHING.
5. **TOCTOU is a threat-model statement, not a code comment:** v1 confinement guarantees hold against the defined non-concurrent-local-attacker model; kernel-enforced resolution (openat2/RESOLVE_BENEATH) and namespace/mount isolation are provider-strength mitigations, never an implicit v1 claim.
6. **No inode/hard-link restriction in v1** — hard-link aliasing does not violate the pathname-confinement invariant actually being protected; expanding the predicate beyond its invariant is rejected.

### Q-L5-5 — Environment and secret boundary (CLOSED WITH AMENDMENT)

Owner-locked, with the "credential contamination handling" reframe replacing "redaction":

1. **Empty environment by default.** An L5 execution starts with an empty environment; only explicitly declared, non-secret variables may be injected (v1 set: HOME/TMPDIR → env-owned tmp, LC_ALL=C, git config neutralizers). **No PATH** — executables are invoked by pinned absolute path. Declared injection values are validated non-secret at artifact load (the envelope is hashed and traced; a secret in it is a secret in the audit log forever).
2. **In-process executors receive no ambient environment access** — executors get `(entry, args, target)` only; `os.Getenv`/`os.Environ` forbidden in executor packages, mechanically enforced (CI lint, dispatch-completeness enforcement style). The boundary is an API surface, not a fake process sandbox.
3. **Secrets never enter static/versioned/hashed execution artifacts.** Future credentials enter ONLY through the broker path at execution time: ceiling declares permissible scopes → grant narrows → Class-3 credentialed-capability registration → broker issues short-lived scoped credential (every issuance audited). Broker is governance-plane, never model-addressable.
4. **Credential contamination handling, not redaction.** A broker-issued credential is a protected secret object whose exact byte representation must not cross the evidence boundary. The capture boundary may replace ONLY exact broker-issued secret byte sequences — never semantic, pattern, contextual, or heuristic judgment. Typed contamination event in trace; hashes computed post-handling; post-handling bytes are ordinary L2 evidence and thereafter byte-exact. Secret-pattern scanning stays defense-in-depth flagging, never transformation authority.
5. **Pre-contamination bytes must never exist in durable observability** — no debug-log/telemetry/crash-diagnostic branch before the contamination boundary; ordering is process output → capture → contamination boundary → {evidence, trace, hash}.
6. **Partial/transformed credential leakage is a documented residual** — exact-value handling prevents the exact issued value crossing; it does not detect transformations. Mitigated by broker discipline (short TTL, narrow scope, non-echoed credential types, memory-only where possible), never by turning L5 into a general secret detector. Stronger guarantees require a dedicated security review when the credential class arrives.
7. **Future in-process credentials must not use `os.Setenv`/ambient process state** — subprocess credential injection and in-process secret capability are distinct designs; the latter requires a memory-scoped interface (part of the future credential design, not v1).

**Locked principle:** *L5 may remove its own secret contamination before the evidence boundary; L5 may never sanitize external evidence.* Resolves the L2-5.1 tension: the byte-exact rule applies once something has become evidence; a Themis-issued secret in output is contamination, not evidence.

### Q-L5-6 — Process and privilege boundary (CLOSED WITH AMENDMENT)

Owner-locked:

1. **Identity floor = non-elevation, not privilege isolation:** effective privilege(execution) ≤ effective privilege(harness). v1 checks: no elevation wrapper, reject setuid/setgid executable, assert real/effective UID relationship, reject world-writable executable or containing directory.
2. **v1 provider identity is honestly declared `inherited`** (`process_identity: inherited | dedicated-user | namespaced`), recorded in provider declaration and trace — the slogan is structurally impossible because the artifact says "inherited" in writing.
3. **Non-elevation is defined at the property level** ("the provider must not introduce privilege elevation relative to the harness execution context"); UID equality is an implementation *check*, never the complete architectural definition (setgid, supplementary groups, fs capabilities, privileged wrappers all exist). Implementation evidence states exactly which mechanisms it checks.
4. **Every out-of-process execution belongs to an execution-owned process group/session** L5 can terminate as a unit — property requirement (bounded termination depends on it), not platform mechanism.
5. **Executable identity is pinned + attested:** resolved path, observed cryptographic digest, mode/permission metadata recorded at provision. **Attestation ≠ behavioral trust** — the digest is evidence of what was executed, not proof it is trustworthy.
6. **Threat-model residual (explicit):** v1 subprocess isolation is behavioral confinement of a trusted pinned binary, not privilege confinement of an untrusted one; a hostile/compromised git binary operates with the developer's existing privileges. Bounded by the capability vocabulary: no run_command, no model-controlled executable, no arbitrary command strings, no dependency installation ⇒ no v1 ingress for attacker-directed subprocess code. Vocabulary expansion (OPEN-2) MUST revisit this assumption; future capabilities requiring stronger privilege isolation declare it as a provider/capability requirement, never silently inherit `inherited`.
7. **Supplementary-group handling stays mechanism-level** — never an independent L5 contract property (checkbox-security refusal).

**Locked principle:** *v1's actual security boundary is construction (argv, environment, binary, cwd, network), not identity — and the trace must present it as exactly that.*

### Q-L5-7 — Network boundary (CLOSED WITH AMENDMENT)

Owner-locked:

1. **Network isolation is a declared-strength property:** `denied-by-construction` (v1 local) vs `denied-by-enforcement` (stronger provider capability). v1 claims no kernel-level network isolation; its evidence is the deterministic absence/refusal of network-capable inputs and ambient network configuration — typed refusal of endpoint-naming input (URL schemes, scp-syntax) at argv construction, never a downstream git network error; invocation profile contains no remote-capable operation; sources are plain local paths under the governed mirror root; empty env kills proxy/SSH/agent channels; neutralized config kills `url.insteadOf`/credential helpers.
2. **The network property includes all address families and loopback.** "No Internet" is never an acceptable synonym for "network denied" (loopback is the classic hole — a netns-style provider leaving `localhost:11434` reachable would expose our own model endpoint under a "denied" declaration).
3. **Host-service isolation is a separate declared property from network isolation** — they fail independently (denied netns + bind-mounted `/var/run/docker.sock` = total escape under "network: denied"). v1 declares both `denied-by-construction`, sharing the Q-L5-6 trusted/attested-binary residual by reference.
4. **No channel blocklist is the security boundary.** Enumeration belongs to the property definition and its test corpus; the construction mechanism is fundamentally deny-by-not-providing (no input names an endpoint; no ambient state can discover one).
5. **Stronger network/host-service isolation is a precondition** to arbitrary-execution capabilities (run_command / OPEN-2) entering the vocabulary.

**Locked principle:** *A provider's declared isolation strength describes what it can guarantee; absence of an enforcement mechanism must never be represented as enforcement.*

### Q-L5-8 — Repository-instruction activation (CLOSED WITH AMENDMENT)

Owner-locked:

1. **Core principle:** *Provisioning establishes availability and identity; registration establishes authority. L5 contributes provenance — which bytes — and never eligibility — whose voice.*
2. **Four-control chain, L5 supplying exactly one:** registration (governance decision, Themis-owned) + verified pinned checkout (L5's provenance contribution) + L1 scope cap (`ScopeRepository` weakest tier, never overrides governed) + L1 pattern gate (directive-pattern policy per load, fail closed).
3. **Unregistered `AGENTS.md` is ordinary workspace data** — readable via tools, L2-classified external-untrusted, invisible to the instruction plane, no error (closed vocabulary: absent = nonexistent).
4. **Registration binds repository identity, not filesystem path** — {repository identity, permitted instruction paths, maximum scope}. A mutable local path must never satisfy registration; identity must correspond to the provisioned workspace's repository identity. Eligibility: registered(repo_identity) ∧ provisioned(repo_identity, pinned_SHA) ∧ verified_checkout ∧ registered_path ∧ pattern gate — evaluated at L1 resolution time, never at provision.
5. **Registration does NOT pin instruction-content hash** — content hash belongs in provenance/audit (every load records {repo, SHA, path, content hash} in trace). Content needing immutable central governance belongs in the governed instruction tree; `ScopeRepository` must not become disguised governed instructions (scope cap + pattern gate are what make repo-mutable content safe).
6. **Instruction path must resolve to a regular file without symlink traversal**, reusing the L5 confinement implementation (ResolveMode + stricter no-symlink rule) — no instruction-specific filesystem predicate.
7. **v1 scope:** activation ships in this change as the final milestone, minimal — registration artifact + eligibility check + root `AGENTS.md` only, four-control chain wired end-to-end (closes the Q-L1-1 IOU).

### Q-L5-9 — Limit semantics (CLOSED WITH AMENDMENTS)

Owner-locked:

1. **"Supported" is removed from the contract vocabulary.** Per dimension a provider declares exactly one of: `enforced` (deterministic typed enforcement at breach, at declared granularity — always, not usually) | `observed` (measurement recorded in trace, available to a subsequent deterministic acceptance decision; no claim the execution was stopped) | absent (no claim exists). ProvisionSpec requirements default to `enforced`; a requirement for strength the provider cannot deliver fails closed at provision.
2. **No bare dimensions.** Explicit semantics and units: `wall_deadline_s`, `cpu_time_s`, `mem_bytes`, `disk_bytes`, `file_bytes`, `proc_count`. "CPU = 2 cores" (cpu-rate) is a different dimension from cpu-time and is absent in v1.
3. **Deadline is not a single guarantee:** effect enforcement (no result crosses the boundary post-deadline; typed timeout) vs consumption enforcement (execution stops consuming) — Tier-0 delivers effect-only (cooperative consumption, preserving the Q-L5-2 bounded-termination amendment); Tier-1 delivers both (group kill).
4. **v1 matrix:** wall_deadline_s: T0 enforced-effect/cooperative-consumption, T1 enforced (group kill) · cpu_time_s: T0 absent, T1 enforced (RLIMIT_CPU) · mem_bytes: T0 absent, T1 **observed** (post-hoc rusage max-RSS; RLIMIT_AS/DATA unreliable on macOS — the "we support memory limits" lie refused in writing) · disk_bytes: T0 absent, T1 observed · file_bytes: T0 enforced at tool boundary (pre-write byte check), T1 observed at hand-off scan · proc_count: **deliberately absent** (RLIMIT_NPROC is per-UID under `inherited` identity — enforcement mechanism could damage the developer's session/host; fork-bombs stay in the trusted-binary residual).
5. **Observation is a control input, not decorative telemetry:** an observed breach deterministically gates downstream artifact acceptance at hand-off; it is never retrospectively represented as execution enforcement.
6. **Model-visible breach typing:** outcome class + dimension only (`error(timeout)`, `error(resource-exceeded:{dimension})`); observed usage numbers never echo to the model; trace records {declared limit, observed usage, enforcement action}.

**Locked principle:** *A limit dimension may be claimed only at the strength actually delivered: enforced means deterministic typed enforcement at breach; observed means measurement recorded in the trace and available to a subsequent deterministic acceptance decision; absent means the provider makes no claim. Every dimension has explicit semantics and units. A provisioning requirement for a strength the provider cannot deliver fails closed at provisioning.*

### Q-L5-10 — Artifact boundary (CLOSED)

Owner-locked ownership: execution-completed (L5/provider) → eligible-to-egress (L5, deterministic structural/provenance/bounds gate) → is-evidence (L2) → accepted-into-Themis (governance/human). *L5 determines whether bytes are structurally eligible to leave; it never determines what those bytes mean.*

**Artifact Egress Contract v1:** the artifact is a **diff against the pinned base** — {base: {repository_identity, pinned_sha}, changes[]: {path, change_type, old_hash, new_hash, size, mode}, patch_content, tool_audit_refs[]}. Egress uses ResolveMode (no second confinement predicate); symlinks never followed (target string is data, typed entry); `.git*` excluded (mechanism state, noted in trace); **no file-type allowlist** (structural constraints only; binary/text is metadata, not an authorization category); per-file/total-size/file-count bounds from WorkspaceExecutionCeiling (narrow, never widen); cryptographic binding to {task, execution, repository identity, pinned SHA, provider declaration, limits record}; manifest hash length-framed canonical; **no partial artifacts** — timeout/breach/teardown anomaly ⇒ no artifact (observed resource breaches become deterministic egress refusal); artifact **copied and hashed before teardown**, never referenced or streamed (closes the inspected-vs-shipped gap). Exactly two workspace→L2 evidence paths: L4-authorized tool result during execution, and the L5 egress artifact via L2 — anything else is an architectural violation.

**Locked principle:** *Execution completion is a provider outcome; egress eligibility is a deterministic structural contract owned by L5; evidence authority is L2's; acceptance is governance's. The egress gate is content-neutral — structure, provenance, and bounds, never meaning — and produces either a complete, hashed, provenance-bound artifact from a clean terminal state, or nothing.*

### Q-L5-11 — Artifact lifecycle and durability (CLOSED)

Owner-locked. Three lifecycle transitions, only the first belonging to L5: ArtifactStore **acknowledgment** (execution-local bytes → Themis-owned durable record) → L2 delivery (record → evidence, classified external-untrusted) → governance decision (evidence → accepted change).

**Locked sequence:** egress passes → materialize directly into ArtifactStore staging (no L5-retained artifact state, no third custody location) → store commit → fsync → **ACKNOWLEDGMENT (authoritative L5 custody boundary)** → trace records artifact address → teardown permitted → L2 may consume by address.

**Key invariant:** teardown is permitted iff no artifact is expected or the ArtifactStore has acknowledged. Locked consequences: no L2 consumption pre-acknowledgment; persistence failure → typed `artifact-persistence-failed`, fails closed, **does not preserve the workspace** (teardown proceeds; work lost is an accepted v1 cost — provisioning is deterministic from the pinned SHA, so re-execution is an orchestration decision, never an L5 auto-retry); post-acknowledgment immutability is structural (content-addressed, write-once, `O_CREAT|O_EXCL`, no overwrite path in code) never procedural; artifact address bound into trace; v1 retention retain-all with retention/GC an explicit L6 IOU; **the store is outside the execution provider boundary** — teardown structurally cannot delete or mutate durable artifacts. v1 store contract: write-once + content-addressed + acknowledge-then-immutable + addressable retrieval.

**Locked custody invariant:** *L5's custody of an artifact ends at ArtifactStore acknowledgment. Before acknowledgment, no teardown and no evidence delivery are permitted. Persistence failure fails the execution closed and does not create a retained-workspace state. After acknowledgment, immutability is structural through content-addressed, write-once storage. L5 participates only in the write/acknowledgment seam; durable lifecycle belongs to L6.*

## 3x. Open questions for the grill (remaining)

1. **Q-L5-12 — Final boundary audit:** the complete monotonic L5 state machine; legal transitions; teardown verification claims; then the operational proof gate.

## 4. Test plan (three-state discipline)

Provision/teardown determinism + host-cleanliness assertions; escape suite (symlink, `..`, absolute, race where testable) against the environment boundary; limit-breach typed terminations; mutating-tool old/new hash records; no-push structural proof (no credential, no remote); pinned-ref recording; the L4 decision-table rerun inside the environment; live proof per Q-L5-9.
