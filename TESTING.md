# Testing

Status: current as of 2026-09-15, verified against the tree at that
date. L1–L7 and L9–L11 implemented and frozen, plus G1 and G2; **L8
(Subagents) is reserved and scaffolded, not implemented**, so no row
below covers it.

Two different things are tested here, and conflating them is a mistake:

| | What it protects | How |
|---|---|---|
| **The harness suite** | the governed chain's invariants — authority, refusal, determinism, recovery | `go test ./...`, hermetic, no model needed |
| **The benchmark gate** | the *quality of the models* the harness measures and routes to | `themis-bench gate`, needs a real model |

A green suite says the controls hold. It says nothing about model
quality. A green gate says the model did not regress. It says nothing
about the controls.

---

## Running the suite

From `src/harness/` (the Go module root — CI and the runbook both work
from there):

```bash
go build ./...              # must be clean
go vet ./...                # must be clean
gofmt -l .                  # must print nothing (CI fails on any output)
go test ./... -count=1      # the whole suite
go test -cover ./...        # with coverage
```

From the repository root, the same via the workspace:

```bash
go test ./src/harness/... -count=1
```

From `src/harness/benchmarks/`:

```bash
make check                  # fmt + vet + test, benchmark packages only
```

**Expect roughly ten minutes.** The suite is deliberately slow in
places: it kills real processes, sweeps fault points exhaustively, and
drives live models. Slowest packages on a laptop:

| Package | Wall | Why |
|---|---:|---|
| `orchestration` | ~220s | full governed walks, fault sweeps, real-kill recovery |
| `tools` | ~110s | timeout enforcement, provider loops |
| `execution` | ~95s | workspace provisioning, seal/egress, process-group kills |
| `verification/seam` | ~90s | end-to-end remediation walks |
| `ratchet` | ~45s | comparison, cold reconstruction, AST walls |
| `context`, `state` | ~34s each | pressure proofs; crash-window sweeps |

---

## Live tests: endpoint-gated, not environment-gated

This is the one thing that surprises people, so read it before running
the suite on a machine with Ollama running.

The live proofs **skip only when nothing answers at the model
endpoint.** They are not opt-in. If Ollama is up, they run.

| Variable | Default when unset | Used by |
|---|---|---|
| `THEMIS_LIVE_OLLAMA` | `http://localhost:11434` | all live proofs |
| `THEMIS_LIVE_MODEL` | `WhiteRabbitNeo/WHiteRabbitNeo-2.5-Qwen-2.5-Coder-7B:latest` | `context` — the L1+L2 conversation proofs |
| `THEMIS_LIVE_TOOL_MODEL` | `qwen2.5:7b` | `tools`, `execution`, `state`, `orchestration`, `ratchet`, `verification/seam` — the tool-calling proofs |

Consequences:

- **Endpoint down → clean skips.** This is how CI stays hermetic.
- **Endpoint up, default models not pulled → real failures.** Pull both
  defaults, or point the variables at models you do have:

  ```bash
  ollama pull qwen2.5:7b
  export THEMIS_LIVE_MODEL=qwen2.5-coder:7b      # if the default is absent
  ```

- **Endpoint up, models present, whole suite at once → the live proofs
  fight each other.** Do not do this. See below.

### Never run the full suite with live proofs enabled

There are nine live proofs, in nine packages, and `go test` runs
packages in **parallel** — so all nine drive one model server at once.
They queue behind each other and time out.

This is a concurrency property, not a host-capacity one. Measured on two
very different hosts, both failing:

| Host | Result with live proofs in the full sweep |
|---|---|
| 16 GB / 8-core / Metal | the two `context` proofs fail |
| 62 GB / 24-core / CPU-only | **six** live proofs fail — every one passing alone minutes earlier |

More memory and three times the cores made it *worse*, because
parallelism is the mechanism. An earlier version of this document
blamed "memory pressure"; that was inferred from the first host alone
and the second host disproved it. No host size fixes this — scheduling
does.

So: run the suite hermetic, then the live proofs separately.

```bash
THEMIS_LIVE_OLLAMA=http://127.0.0.1:9 go test ./... -count=1   # hermetic
```

Pointing the endpoint at a closed port makes every live proof skip
cleanly, giving a purely deterministic result where any failure is
unambiguously real.

**A full-sweep live failure is not evidence of a defect. A live proof
that fails in isolation is a real failure** — do not wave that one away.

### Running the live proofs deliberately

```bash
export THEMIS_LIVE_OLLAMA=http://localhost:11434
export THEMIS_LIVE_TOOL_MODEL=qwen2.5:7b
cd src/harness
go test ./verification/seam/ -run TestLiveRemediateWalk   -count=1 -v
go test ./ratchet/           -run TestLiveModelAuthorsCandidate -count=1 -v
go test ./orchestration/     -run 'TestLiveWalkProof|TestLiveSkillWalk'  -count=1 -v
go test ./tools/             -run TestLiveToolProof      -count=1 -v
go test ./state/             -run TestLiveTaskReconstruction -count=1 -v
```

These are **machine-local evidence**: record the model name and digest
next to the result, or the result is not attributable.

---

## What the tests prove, by layer

Every package below is hermetic unless its row says otherwise:
`httptest` model backends, `t.TempDir()` fixture trees, no network, no
repository data directories mutated.

| Layer / package | What the tests establish |
|---|---|
| **L1** `instructions` | deterministic multi-source resolution (order independence, golden hashes, byte-change sensitivity); fail-closed loading (duplicate IDs, missing roots, unrecognized source kinds, atomic multi-source failure); the directive-pattern policy — boundary-tier attribution, exemptions that suppress a named pattern without authorizing anything, pattern-rejection recorded; namespace ownership and task-namespace violations; secret scanning |
| **L2/L3** `context` | contract-gated gathering and composition; confinement; evidence delivered **verbatim**; fence-collision scanning and sibling-fence forgery inert; metadata-injection refusal; amplification and total-byte caps; pressure management — deterministic drops, withheld markers, drop/dedup bijection, budget fail-closed, re-entry refused; golden payload hashes |
| **L4** `tools` | the authorization decision table; registry ∩ grant ∩ quota with availability decided before arguments; mutating-visibility requirements; transactional patch application; **no ambient environment** in executors (structural); timeout enforcement; closed vocabulary |
| **L5** `execution` `confine` | sealed read-only workspaces; egress gating — clean-seal requirement, observed-breach refusal, staged rename, count/total bounds, VCS exclusion, store-inside-environment refusal; artifact-store immutability; **no-push structural proof**; ceiling/spec fail-closed and spec⊆ceiling containment; lifecycle edge-product exhaustiveness; timeout group-kill; no `PATH` in the environment |
| **L6** `state` | append-only object and event planes; **no deletion path** (structural); forgery detection at the read boundary; torn tails, corrupt entries, junk inside frames; recovery authorities that refuse laundering and refuse a live writer; exhaustive crash-window fault-point sweep, kept in sync with the source; **real process-kill recovery** (a child appends until `kill -9`, then recovery runs); `ConstitutionHash()` identity; closed vocabularies; projection re-derivation |
| **L7** `orchestration` | deterministic governed walks and replay; control vocabulary two-way closed — **prose cannot move a workflow**, invented control args refused, JSON in prose is not an action; turn/quota/wall-clock exhaustion as governed edges; duplicate submit refused; envelope no-defaulting; grant ceiling at assembly; startup sweep hermetic; **real-kill → no continuation**; loop fault sweep synchronized with the source; exported-API closure |
| **L9** `skills` | catalog append-only with no write capability and no state verb in registries; unregistered copies and withdrawn entries refused; composition-hash mismatch, tampered artifacts, symlink escapes, repo traversal refused; quotas and wall deadlines **narrow only**; caller input never touches the procedure; writable-root disjointness |
| **L10** `verification` `verification/seam` | registered contracts, append-only registry, no write capability; hostile verifier output is **inert domain data**; evaluator failure mints no outcome; reason-class integrity; verification reconstruction (single-element swaps fail; machinery outcomes skip the result chain); seam refusals pre-instance, multi-slot, withdrawn-contract; remediation e2e and fail-closes-gate |
| **L11** `ratchet` `cmd/themis-ratchet` | comparison determinism, direction symmetry, refusals; per-metric vs overall relation; dominance and scalarization with retained Δ; non-regression boundary inclusivity; canonical-delta **golden bytes**; **G2 witness grounding** — facts must be witnessed, bench-witness predicate, admission observed at door bytes; cold reconstruction (CONFIRMED / discrepancy / missing inputs / never repairs); **AST walls** — no mutable package state, closed importers, API closure, forbidden component names; CLI acceptance contract |
| **G1** `deployment` | anchor admission (`AdmitAnchor`); parse refusals and loader bounds; `HashDir` determinism, **refusal of non-regular entries** (the symlink bypass), and **injective framing**; duplicate-artifact refusal; `VerifyAnchorRecord` read path; anchors-registry append-only |
| **G1 at L7** `orchestration` | `TestAnchoredOpen`, workflow-set and instruction-policy enforcement, model-registry pin, constitution pin, **indivisible bundles**, unanchored-requires-explicit-opt-in, registry append-only across Opens, submit refuses an unanchored bundle |
| **Chain** `integration` | `TestPhaseCEndToEndChain` — the whole L1→L11 chain as one anchored deployment instance, including the read-path deployment check |
| **Model layer** `runtime/model` `internal/llm` | Ollama and OpenAI clients against `httptest`: tool conversations, request shape (pinned `temperature 0, seed 42`), status/error/empty-choice handling, cancellation; registry resolution (defaults, aliases, API-key env, option overrides, malformed files) |
| **Router** `internal/service` | best-per-category selection, latest-run-wins, variant exclusion |
| **Benchmarks** `benchmarks/internal/*` | definition loading, prompt templating with partials and variants, the three validators (keyword, regex, JSON ground truth), report aggregation and rendering, gate pass/fail semantics, and the **end-to-end pipeline test** |

### Structural tests are load-bearing

Several tests assert properties of the *source*, not of a run:
`TestNoDeletionPathStructural`, `TestNoAmbientEnvironmentInExecutors`,
`TestNoPushStructuralProof`, `TestASTWall`, `TestNoMutablePackageState`,
`TestExportedAPIClosure`, `TestNoCatalogWriteCapability`,
`TestNoRegistryWriteCapability`, and the fault-point
synchronization tests (`TestFaultPointsSyncWithSource`,
`TestLoopFaultPointsSyncWithSource`).

These fail when someone *adds a capability that must not exist* — a
delete path, a registry writer, ambient environment, a package-level
variable. Do not "fix" one by relaxing the assertion; that is the
control, not the test's opinion.

---

## Mutation testing: the class the suite cannot measure

A green suite says every test passed. It does not say a **control** was
exercised — a guard can be correct, cited in a traceability record, and
never once executed by any test. The archive sweep finds dead citations;
a deployment finds controls that fire. Neither finds a guard that is
correct, referenced, and silent.

`evidence/harness/mutate` measures exactly that class. The operator is
deliberately narrow: for every `if <cond> { … return <error> … }`,
replace the condition with `false` and run that package's tests. If
nothing fails, that control is not covered by its own package's tests.

```bash
git worktree add /tmp/themis-mutate HEAD        # isolation is ENFORCED
cd evidence/harness
GOWORK=off go run ./mutate -worktree /tmp/themis-mutate -timeout 90s
```

It **refuses to run anywhere but a linked git worktree** — it rewrites
source in place, and the AGENTS.md probe-isolation invariant is not
advice there.

Read the results carefully:

- **A survivor is a finding, not a failure.** Some are equivalent
  mutants (unreachable, or redundant with an adjacent guard). Each needs
  a judgement recorded; none should be dismissed unexamined.
- **"Survived" is narrower than "untested"** — several survivors are
  proven by the Phase C matrix, which lives outside the module.
- **An equivalent mutant is not a closed question.** Ask "unreachable
  because of *what*, and is that tested?" Four of five equivalents in
  the 2026-09-14 pass rested on a premise nobody had checked, and two of
  those premises were false.

The dominant defect it finds is **mutual cover**: two adjacent guards
and one test that trips both, green, named for the control, evidencing
neither half. When a control is untestable in place (a privilege floor,
a TOCTOU window), the remedy is to extract the predicate to a pure
function — never to add a seam that makes the system more permissive in
order to observe it.

Results and dispositions:
[`docs/development/mutation-pass-2026-09-14.md`](docs/development/mutation-pass-2026-09-14.md).

### The same standard applies to the tests

A test that fails for the wrong reason is the same defect as a guard
that refuses for the wrong reason. Two rules follow, both learned by
violating them:

1. **Assert which refusal, not merely that one happened.** When adjacent
   guards share a message, assert the distinguishing detail — the error
   class, the side effect, the wording only one of them produces.
2. **A mutant killed by panic is not evidence.** If suppressing a guard
   crashes the test through an under-constructed fixture, the test
   failed for its own reasons. Give it a real fixture so it fails by its
   assertion — unless the panic IS the behaviour the guard prevents,
   which is itself worth stating.

---

## The end-to-end pipeline test (benchmarks)

`benchmarks/internal/benchmark/e2e_test.go` drives run → evaluate →
validate → report in a temporary suite root against an `httptest` mock
Ollama server. It asserts that the run stage writes runtime-agnostic
envelopes plus a manifest recording options and rendered-prompt hashes,
that evaluate skips the manifest and normalizes metrics, that validate
scores against the expected spec, and that the report carries the right
aggregates. It catches the cross-stage contract breaks (file formats,
directory layout, manifest exclusion) unit tests miss.

## The Phase C register (the chain)

`integration/phasec_test.go` is the harness equivalent: it runs the
governed chain end to end as a concrete **anchored** deployment
instance. When it breaks, a cross-layer contract broke — not one
package's internals.

---

## Continuous integration

`.github/workflows/ci.yml` runs on every push to `main` and every pull
request, from `src/harness`: gofmt (fails on any unformatted file),
`go vet ./...`, `go test ./...`, and release builds of `themis-bench`,
`themis-ratchet`, and `themis-run` (the production invocation surface).
CI runs on `ubuntu-latest`, and the harness is developed on darwin —
so every push is a dual-platform check, which is how the 2026-09-14
host-timing defect in the L5 group-kill test was caught.

CI has no model endpoint, so every live proof skips there. **The live
proofs are therefore never evidence CI produced** — the
operationally-proven verdict always cites a named local execution with
a recorded model name and digest.

---

## Testing a deployment (not the code)

The suite tests the implementation. Proving a *deployment instance* —
real anchor, real ceiling, real host — is a separate procedure:

- [`docs/development/deployment-test-plan.md`](docs/development/deployment-test-plan.md)
  — Phases A–F plus a concrete real-VM scenario (VM-0…VM-9), including
  the 19-row anchored negative-space table (C1–C19: every attempt to
  govern execution with something the anchor did not admit).
- [`docs/operations/deployment-runbook.md`](docs/operations/deployment-runbook.md)
  — how the deployment is built in the first place.

Most of the negative space is already automated here
(`orchestration/verification_seam_test.go`, `deployment/anchor_test.go`);
on a real host those rows are re-run against the **real** anchor rather
than a test-minted one.

**After any binary change, re-ask the old records.** Phase D proves a
record is re-establishable at the moment it is written; it cannot prove
the record still means something once the binary is different. That is
the G1 claim applied over time, and `evidence/harness/recheck` asks it:

```bash
GOWORK=off go build -o <bin> ./recheck/
<bin> -deploy "$DEPLOY" -anchors-registry <anchors.json>
```

Read-only and authority-free — it creates no task, writes no object,
registers nothing. For every recorded task it re-runs D4 (the identity
the record carries), D5 (the anchor bytes, recovered by their own hash)
and D6 (registry re-establishment), plus the L6 verdict. Tasks governed
by a **withdrawn** anchor must still re-establish: withdrawal stops new
opens without rewriting history. Optional `-anchor-sha256` asserts which
deployment you expect; a task under another one reports `OTHER`, not a
failure, and exits 3 rather than 1 — belonging to a different deployment
is not a durability fault.

---

## The regression gate (testing *models and prompts*, not code)

After any model, prompt, or quantization change:

```bash
cd src/harness/benchmarks
make bench MODEL=<model>                       # produce today's scores
./bin/themis-bench gate <model> --baseline <last-good-date> --max-drop 5
```

Exit code is non-zero when a benchmark that existed at the baseline is
missing, or the average score dropped more than `--max-drop` points.
Prompt changes are gated the same way through variants: run `make bench
MODEL=<model> VARIANT=<name>` and gate/compare `MODEL@<name>` against
the base series.

Scoring is fully deterministic — keyword, regex, and JSON-ground-truth
validators. No LLM-as-judge anywhere in the pipeline.

---

## Triage rule for anything a test surfaces

The owner's standing classification applies to test failures as much as
to deployment refusals:

1. **deployment/configuration** → fix the configuration (wrong model
   pulled, endpoint down, stale `models.json`)
2. **implementation defect** → fix it under the frozen constitutions
3. **existing documented residual** → verify it is recorded; do not
   widen it
4. **genuine architectural gap** → only this reopens a locked decision

A refusal that fires with the **wrong reason** is a finding, not a
pass. Never make a test green by weakening a loader, inferring a
default, or reinterpreting an artifact.

---

## Writing new tests

- Table-driven `t.Run`; helpers call `t.Helper()`.
- Anything that talks HTTP gets an `httptest` server — never a live
  endpoint in a non-live test.
- Anything that touches the filesystem builds its fixture tree in
  `t.TempDir()` — never the repository's own data directories.
- A new live proof must skip cleanly when the endpoint does not answer,
  and must name its model.
- A new refusal needs a test asserting the **message**, not just the
  error — a refusal for the wrong reason is a defect.
- A new benchmark needs no Go test: add `definitions/BXXX.json`,
  `prompts/BXXX.md`, `expected/BXXX.json`; the loader validates it at
  run time and the e2e test covers the machinery.
- A new validator gets: happy path, scoring edge cases, malformed-spec
  error, a case in `validator.go`'s dispatch, and a line in the READMEs.

**Review probes never run against the live working tree.** Mutation
testing, backup/restore, and experimental edits happen in an isolated
copy or a `git worktree`, with artifacts removed before reporting. This
is a repository invariant recorded in
[`AGENTS.md`](AGENTS.md) after a reviewer silently reverted enforcement
code while the build stayed green.
