# Systematic mutation pass — refusal suppression (2026-09-14)

Closes the one class every other method used that day structurally could
not reach: **a control that is untested AND correctly cited.** The
archive sweep finds dead citations. The deployment finds controls that
fire. Neither can find a guard that is correct, referenced, and never
exercised.

Tool: `evidence/harness/mutate`. Operator: for every
`if <cond> { … return <error> … }`, replace the condition with `false`
and run that package's tests. If nothing fails, that control is not
covered by its package's tests.

Run on the deployment host, 20m35s, in an isolated worktree.

## Result

| | 2026-09-14 | 2026-09-15 re-run |
|---|---|---|
| Refusal guards found | 971 | 971 |
| Killed (a test failed) | 433 | **445** |
| Did not compile | 130 — cannot ship, not a gap | 126 |
| **Survived** | **408** | **380** |
| Not mutated (package has no test files) | — | 20 |

**Read the 408 as historical.** It is the figure this document was
written around and the one every Tier-1 finding below was ranked from,
but it is not the current state. The 2026-09-15 confirmation re-run is
the live number, with two qualifications below.

### Reconciling the two runs

The 2026-09-14 columns sum to 971 with no "not mutated" row because that
run **predates `32b3b58`**, the `hasTests` fix. The ~20 guards in
packages with no test files of their own were still being counted then:
four as compile errors (130 → 126) and sixteen as false survivors.

Killed is the only bucket directly comparable across the two runs — a
guard in a package with no tests can never be killed — and it moved
**433 → 445, exactly +12**. The survivor list independently confirms all
twelve Tier-1 lines are absent, so those twelve *are* the +12. Nothing
is unexplained, and no previously-killed guard became a survivor (that
would have shown 444). Guard count identical at 971 corroborates it: the
day's edits added and removed no guards.

### Two qualifications on the 380

1. **It was measured mid-sweep.** The re-run ran at the commit where
   Tier 1 was complete and nothing else was. The nineteen guards closed
   afterwards — `themis-ratchet derive` (1), the G1 anchor sweep (8),
   L7 static boundedness (5), the L9 substitution boundary (5) — are
   still counted as survivors in it. Expected current figure is
   therefore **~361, not re-measured.** Do not quote 361 as evidence;
   quote 380 with this caveat, or re-run.
2. **"Survived" still does not mean "untested."** The distinction the
   original run made holds: some survivors are covered externally by
   the Phase C matrix, and some are equivalent or redundant guards. Of
   the twelve Tier-1 items worked through, five turned out to be
   equivalent mutants.

The mutation pass having been RUN closes the "not yet run" qualification
that stood before 2026-09-14. It does not establish that every surviving
mutant needs a test, nor that mutation coverage is complete in any
absolute sense — the operator is deliberately narrow (see below).

### Survivors by category

| Category | Count | Reading |
|---|---|---|
| I/O & parse error propagation (`if err != nil`) | 191 | error plumbing; reachable only with filesystem/serialisation fault injection. `state/` has fault points for this; most packages do not. A coverage observation, not a security finding. |
| Semantic controls | 157 | the real output |
| Schema / format validation | 60 | genuinely untested; mostly low severity because malformed input usually fails elsewhere too |

Of the 157 semantic controls, 115 are in security-relevant packages.

## Two corrections to that number

**`confine` (10 guards) is a tool false positive.** The package has no
`_test.go` of its own — its tests live in `context/confine_test.go` — so
`go test ./confine/` reports "no test files" and exits 0, making every
mutant survive trivially. Fixed in the tool (commit `32b3b58`); those
controls are tested. Same for `benchmarks/cmd/themis-bench`.

**"Survived" means "not covered by its own package's unit tests",**
which is narrower than "untested". Several survivors are proven by the
Phase C matrix, which lives outside the module in `evidence/harness`:
`orchestrator.go:168` (C19), `:474` (C16). The converse also holds — a
survivor that Phase C does not cover either is genuinely unguarded.

## Tier 1 — where a disabled guard permits something promised against

| Control | Exposure |
|---|---|
| ~~`orchestration/orchestrator.go:763`~~ — grant digest changed between attribution and execution | TOCTOU on authority itself. **Resolved 2026-09-15**: equivalent mutant, but its premise was not — the digest omitted `ThemisScope`. See below. |
| ~~`orchestration/orchestrator.go:547`~~ — artifact changed between loading and durable capture | TOCTOU on the governed record. **Closed 2026-09-15**: the positive half was proven e2e, the refusal never exercised. See below. |
| ~~`orchestration/orchestrator.go:215`~~ — supplied ceiling ≠ anchored ceiling **at Open** | G1. The C matrix covers only the SubmitTask side (C13). **Closed 2026-09-15**. See below. |
| ~~`orchestration/orchestrator.go:372`~~ — **L7** constitution pin | C9 exercised the **L6** pin at `:369`; this is a separate line. **Closed 2026-09-15**, both pins. See below. |
| ~~`state/task.go:211`~~ — a record may reference only already-durable objects | record-before-effect. **Resolved 2026-09-15**: equivalent mutant; the sink enforces it and IS covered. The test naming it was overstated. See below. |
| ~~`execution/local.go:243`~~ — HEAD ≠ pinned SHA post-condition | workspace could sit at the wrong commit. **Closed 2026-09-15**. See below. |
| ~~`skills/catalog.go:193`~~ — manifest self-declaration ≠ registration | L9 two-way identity. **Closed 2026-09-15**. See below. |
| ~~`skills/catalog.go:85`~~ — manifest_path traversal (`..`, absolute) | path containment. **Closed 2026-09-15**. See below. |
| ~~`orchestration/loop.go:474`~~ — verification event undeclared yet produced | L7 invariant. **Resolved 2026-09-15**: equivalent mutant; two of the three gates it rests on were untested and now are. See below. |
| ~~`orchestration/loop.go:382`~~ — verifier-eligible call with no evaluator wired | fail-closed. **Resolved 2026-09-15**: equivalent mutant; its assembly gate was already tested. |
| ~~`ratchet/package.go:129`~~ — criterion bytes ≠ constituent conditioning tuple | L11 binding. **Closed 2026-09-15**. See below. |
| ~~`verification/contract.go:249/257`~~ — provenance completeness | "not contract-relaxable". **Closed 2026-09-15**. See below. |

### One that deserves separate attention

**`execution/local.go:153`** — endpoint refusal (`://`, `::`).
`TestEndpointRefusal` exists and is cited in the L5 traceability, yet
this guard survives. The likely cause is that `refuseEndpoints` carries
several conditions and every test case is caught by a *different* one,
leaving this branch unexercised.

**CONFIRMED and CLOSED 2026-09-14.** `refuseEndpoints` carries two
checks. The second — first colon with no `/` before it — catches
everything the first catches *unless* the text before the colon contains
a slash. Every case in `TestEndpointRefusal` (schemes, scp with and
without user, `ext::`, `fd::`) has no slash before its colon, so the
second check always fired and the first was never the control under
test.

The gap the first check alone covers is real, not cosmetic:

    /abs/path::evil   second check: prefix "/abs/path" contains "/", does not fire
                      first  check: contains "::", fires

git reads `<transport>::<address>`, so that names a remote helper —
exactly what the control exists to refuse.

Closed by adding the two cases only the first check can catch,
`/local/mirror/repo::evil` and `./dir/x://y`. Mutation-verified: with
the guard replaced by `if false`, both new cases fail and the eight
original cases still pass — which is the point. The test was green
before and green after; only the new cases distinguish them.

This is the third instance that day of a passing test standing in for a
control that never fired, after the never-executed L10 tamper test and
Phase C row C15.

### `orchestrator.go:763` — the guard is equivalent; its premise was not

**RESOLVED 2026-09-15, with a real defect found underneath.**

The mutant survives and will keep surviving: it is an equivalent mutant.
Between `governed["grant_authority"] = grantAuthorityDigest(grant)` at
`:723` and the comparison at `:763`, the only intervening statement is
`CreateTask`, which never sees `grant`. Nothing in-process can move
either operand, so no test can reach the branch. It is a fail-closed
assertion against a *future* edit inserting grant-mutating code between
recording and use — which is a legitimate thing to keep, and not a thing
a test can pre-empt. Adding a production seam solely to make it
reachable would buy nothing and widen the surface.

What the survivor did surface is the guard's premise, and that turned
out to be false on two counts.

**1. The digest was blind to `ThemisScope` (fixed).** The digest hashes
"exactly what a grant PERMITS", but omitted the per-entry themis-id
scope — the prefix list `tools/authorize.go:158-167` consults to decide
which Themis identifiers a `themis-id` tool may reach. A grant scoped to
`FIND-1:` and one scoped to the whole `FIND-` family are materially
different authorities that recorded an **identical** `grant_authority`.
That defeats exactly the L9 test-review HIGH the digest was added for:
"same authority?" was unanswerable in the one dimension a reviewer is
least likely to eyeball. The comparison at `:763` compounded it — it
would compare two values that already agree on a field neither covers.

Fixed by folding the scope into the digest as a *set* (order and
repetition do not change what a scope permits, so they must not move the
digest) with length-prefixed encoding, since prefixes are author-supplied
text and `["a;b"]` would otherwise encode identically to `["a","b"]`.

**2. `CreateTask` aliases the caller's attribution map (pinned).** The
manifest retains `opts.GovernedHashes` by reference rather than copying
it. Today nothing writes through that alias, so the comparison is sound
— but a future normalization or annotation written in place would
rewrite L7's attribution *after* L7 recorded it, and `:763` would
compare against a value it did not produce. `TestCreateTaskDoesNotMutate
CallerAttribution` now pins it; probe-verified by inserting one in-place
write into `CreateTask`, which the test catches.

Evidence: `TestGrantAuthorityDigestCoversEveryAuthorityField`
(`orchestration/review_test.go`) asserts the digest moves for every
field authorization consults and stands still for task identity and
scope reordering. Verified against the pre-fix digest in a worktree: the
three scope assertions and the encoding-ambiguity assertion fail, the
six pre-existing fields pass — so the test distinguishes the fix rather
than restating it.

The pattern from `local.go:153` repeats with a twist: there, a passing
test stood in for a control that never fired. Here, an *unreachable*
control stood in for a premise nobody had checked. Both were found by a
mechanical check that refused to agree, not by reading.

### `orchestrator.go:547` — half a control, proven

**CONFIRMED and CLOSED 2026-09-15.** `TestDurableBytesMustMatchWhatWas
Loaded` proves the positive: for every materialized artifact, the
durably stored bytes hash to the identity L7 recorded from its own load.
It cannot prove the negative, because producing the divergence requires
a write landing between two reads inside a single `SubmitTask` call.
So the control's *refusal* had never once executed — the test that
carried its name only ever walked the agreeing path.

The refusal matters on its own terms: an artifact rewritten after its
loader hashed it and before capture yields a record claiming durability
over bytes that never ran. That is identity over A with durability over
B, the exact shape R-L9-2 forbids.

The package's `faultAt` seam was the obvious lever and is the wrong one:
every fault point is swept by Register C, which asserts a governed
FAILED with an invariant event, and this code runs before the task
exists. Adding a production seam that admits a mid-assembly write would
also be a worse trade than testing the control directly.

Closed by lifting the read-and-prove step into `captureVerified`, which
`TestCaptureVerifiedRefusesArtifactChangedAfterLoad` exercises directly:
content replaced, one byte flipped (length-preserving — no size or mtime
check would catch it), and truncated to zero all refuse as
`ErrInvariant`, while a vanished artifact refuses as `ErrAssembly`,
since nothing has been misrepresented there. Mutation-verified: with the
guard replaced by `if false`, all three change cases fail.

No behavior changed — the same bytes, the same two errors, the same
order. The control simply became reachable by a test.

### `orchestrator.go:215` — the Open side of the G1 ceiling pin

**CONFIRMED and CLOSED 2026-09-15.** The mutation record predicted this
one exactly: the Phase C matrix exercises the ceiling pin on the
SubmitTask side (C13) and nothing exercised it at Open. So a deployment
could be opened under a ceiling its anchor does not pin, and every walk
in it would run under bounds no Governance act ever admitted — the
ceiling is frozen at Open and carried forward, so a single unchecked
Open contaminates the whole deployment rather than one task.

Closed by a subtest under `TestAnchoredOpen` that supplies a
substitute ceiling differing only in what it PERMITS — the same shape
and the same mirror root, with `max_wall_deadline_sec` and
`max_proc_count` raised. That is the substitution the pin exists to
refuse.

The subtest asserts its own premise first: `execution.LoadCeiling` must
accept the substitute. A malformed ceiling would be refused two lines
later by the instantiation check, the pin would stay untested, and the
subtest would pass anyway — the Phase C row C15 failure mode exactly.

Mutation-verified: with the pin replaced by `if false`, the deployment
opens and the subtest fails.

### `orchestrator.go:372` — the L7 constitution pin

**CONFIRMED and CLOSED 2026-09-15.** Two adjacent guards refuse a binary
whose compiled control vocabulary differs from the one its anchor
pinned. Phase C row C9 exercises the L6 pin at `:369`; the L7 pin one
line down was exercised by nothing, so a rebuilt binary with a changed
L7 constitution could have opened under an anchor that pinned the old
one — which is exactly owner finding 4's question ("can changing this
artifact change the behavior or authority of an anchored deployment?")
answered wrongly.

Both pins now have a subtest, and each moves ONLY its own hash. A single
subtest doctoring both would pass against either guard alone and prove
neither — the same trap as `local.go:153`, where one guard stood in for
another.

Mutation-verified independently: suppressing the L7 pin fails only the
L7 subtest, suppressing the L6 pin fails only the L6 subtest.

### `state/task.go:211` — a duplicate rule, and a test that named the wrong one

**RESOLVED 2026-09-15.** Equivalent mutant. `BindArtifact` checks
`store.HasObject(addr)` before appending, and the sink applies the
*identical* predicate to the same address on the same store
(`sink.go:102`) when the event's reference is validated. Every
`BindArtifact` call passes its address as that reference, so nothing
reachable today can distinguish the two checks: with `BindArtifact`'s
suppressed, the refusal still happens, still before anything is
recorded, still as `ErrStream`, with a message that also contains
"already-durable".

The enforcing rule is the sink's, and it is properly covered —
suppressing `sink.go:102` fails `TestEventPlaneBehavior`'s
dangling-reference case. `BindArtifact`'s is defence in depth against a
future binding path that does not reach the sink; like `:763`, that is a
guard a test cannot pre-empt, and removing a security-review-mandated
duplicate is not a call to make from a mutation result.

What was wrong here was the *test's claim*. `TestBindArtifactDoor` is
named for the door and asserts only outcomes the sink produces, so it
reads as evidence for a control it never exercises. Comment corrected to
say what it establishes and what it does not.

Fourth instance of one guard standing in for another, after the L10
tamper test, Phase C row C15, and `local.go:153` — and the first where
the redundancy is exact rather than accidental.

### `execution/local.go:243` — the post-condition that had never fired

**CONFIRMED and CLOSED 2026-09-15.** Q-L5-3 asserts rather than assumes:
after `checkout --detach <pin>` reports success, `rev-parse HEAD` must
return the pin. `TestProvisionFailurePaths` covers a checkout that
FAILS (unknown SHA) — a different branch, several lines up. The
post-condition itself had never executed, so "the workspace is at the
pinned commit" rested on git's behavior rather than on a check anyone
had watched work.

The reachable instance needs no fault injection: git resolves object
names case-insensitively, so `checkout --detach <UPPERCASE 40-hex>`
succeeds while `rev-parse` reports the canonical lowercase. That is a
genuine "checkout succeeded, HEAD != pin".

Stated plainly in the test: it exercises the predicate, not a hostile
mirror. A governed spec cannot carry an uppercase pin (`parseSpec`
requires `^[0-9a-f]{40}$`), so the spec is constructed in-package the
same way the existing traversal case constructs `s.Repo`. The test also
asserts its premise — the canonical pin provisions cleanly — so nothing
but the post-condition can explain the refusal, and asserts the failed
provision tears down to `StateDestroyed` with its trace.

Mutation-verified: with the post-condition replaced by `if false`,
provisioning returns success with HEAD not at the pinned string.

### `skills/catalog.go:193` and `:85` — the L9 catalog boundary

**CONFIRMED and CLOSED 2026-09-15.** Both are reachable, both were
unexercised, and both now fail without their guard.

**`:193` — two-way identity.** `TestCompositionHashMismatchRefused`
covers the hash check one line above, which is why this looked covered.
It is not the same control and cannot substitute: `CompositionHash` is
`json:"-"` and DERIVED from the manifest bytes, so a doctored manifest
registered under *its own true hash* satisfies the hash check
completely. The attack it leaves open is registering one skill's bundle
under another's identity — every hash in the chain verifies, and a
caller asking for `investigate-cve@1` runs the other composition under
`investigate-cve@1`'s attribution. `TestManifestSelfDeclarationMustMatch
Registration` doctors the name and the version independently and
asserts the refusal names the two-way check, so a failure of the hash
check would not be mistaken for a pass.

**`:85` — manifest_path containment.** The catalog root bounds every
manifest it registers; `manifest_path` is resolved against it, so an
absolute path or one climbing out points the registration at any file on
the host, and every pin, template, and procedure then resolves against
the attacker's directory. Refused at LOAD, before any entry resolves — a
catalog containing such a row is not a catalog with one bad entry.
`TestCatalogRefusesManifestPathOutsideRoot` covers absolute, leading,
embedded, and trailing traversal plus empty; mutation-verified, all five
load with the guard suppressed. It also asserts its premise — the same
catalog with an in-root path loads.

### `loop.go:382` and `:474` — two invariants, and the gates beneath them

**RESOLVED 2026-09-15.** Both are equivalent mutants: each says so in
its own comment ("assembly refuses this configuration", "declaration-
gated exposure makes this unreachable when assembly held"). Taking that
at face value is the mistake — the question a mutation survivor asks is
whether the gates the claim rests on are themselves exercised.

`:382` rests on one gate, `o.cfg.Verifier == nil` at assembly. It is
tested; suppressing it fails `TestVerificationAssemblyRefusals`. That
one was sound.

`:474` rests on three, and two of them survived:

| Gate | Before |
|---|---|
| loader: an edge's event must be declared | tested |
| assembly: each of the five verification events is declared | **survived** |
| assembly: each has an edge in the exposing phase | **survived** |

Both survived for the same reason `local.go:153` did: the subtest that
covers them strips the declarations AND the edges together, so either
gate alone refuses it and neither is the control under test. Mutual
cover, and it had hidden the premise of a Tier-1 invariant.

Splitting them exposed something the table above cannot show. The
reachability gate splits cleanly — declare all five, remove one edge,
and only it can refuse. The declaration gate does not: an event that is
not declared can have no edge, because the LOADER refuses that first. So
the assembly declaration gate is itself redundant, and so, in turn, is
`loop.go:474`.

The chain is now: loader (tested) → reachability gate (tested) ⇒ no
undeclared verification event can be produced. The third subtest pins
the ordering that makes the redundancy hold, so a future change letting
such a workflow load surfaces as a changed refusal rather than silently
promoting two dead guards into live ones.

Lesson for the Tier-1 ranking itself: neither surviving gate was ON the
Tier-1 list. They were the premise of an item that was.

### `ratchet/package.go:129` — answering the question by choosing it

**CONFIRMED and CLOSED 2026-09-15.** `DeriveResistantUnderSet`
recomputes resistance from the constituent packages and their criteria.
The criterion supplied for that recomputation must be the one each
constituent comparison was CONDITIONED on — its hash is in the
package's conditioning tuple for precisely this reason. Existing tests
cover derivation and the unfavorable case; both pass the *matching*
criterion, so the binding had never been exercised.

The exposure is not subtle. Pair a regressed comparison with a more
permissive criterion and "resistant under S" comes out true:

    candidate 0.30 vs baseline 0.82  →  score_delta -0.52
    conditioning criterion: non_regression_min -0.05  →  regression
    substituted criterion:  non_regression_min -1.00  →  "resistant"

`TestResistantUnderSetRefusesUnboundCriterion` uses a substitute the
registry would accept — only its region is wider — and asserts its own
premise both ways: the substitute really does accept the delta (or
nothing is being bypassed), and the conditioning criterion still
derives (or the check refuses everything rather than substitution).

Mutation-verified: with the binding replaced by `if false`, the
derivation returns `resistant=true` for a comparison that regressed by
0.52.

### `verification/contract.go:249` and `:257` — mutual cover, again

**CONFIRMED and CLOSED 2026-09-15.** D-L10-10 completeness is not
contract-relaxable and two checks say so: the provenance list must have
exactly the required length, and it must contain every required element.
The loader-refusal table covers only the case where BOTH fire — a list
shortened to one element — so either alone refuses it and neither is the
control under test.

One case per dimension:

| Provenance declared | Fires |
|---|---|
| `[execution_record, raw_output, canonical_result, raw_output]` | count only |
| `[execution_record, raw_output, invented_element]` | membership only |

The second is the one that matters on its own: a contract declaring
three elements, one invented, passes any count check and ships with
`canonical_result` never demanded. Mutation-verified — each check falls
to its own case and to no other.

Third instance of mutual cover in this list, after `local.go:153` and
the verification assembly gates. The pattern is now the most common
single finding of the pass: **two adjacent guards, one test that trips
both, and no evidence for either.**

### `cmd/themis-ratchet/main.go:423` — the same control, in the binary Governance runs

**Found and closed 2026-09-15, during the confirmation re-run.** Not on
the Tier-1 list; it surfaced in the partial output while verifying the
twelve, which is the point of re-running rather than trusting the
per-item verifications.

`themis-ratchet derive` reimplements the criterion-to-conditioning-tuple
binding just closed at `ratchet/package.go:129`. The CLI contract suite
never invokes `derive` at all, so the copy had no coverage.

It is the more exposed of the two. The library takes the criterion as an
argument; this one **re-resolves it from the registry** by the package's
own `CriterionRef`. A registry whose criterion at that ref no longer
hashes to the conditioning tuple is an operational state, not a caller
mistake — and `derive` would then report per-field, relation, and
non-regression against a region the comparison was never computed under.

The new subtest swaps in a criterion at the same ref with a wider region
and a registry that is internally consistent over the new bytes, so no
integrity check can be what refuses. Premise asserted first: the
conditioning registry derives. Mutation-verified — with the binding
suppressed, `derive` prints a full derivation against the substituted
criterion.

**This is the argument for the confirmation run.** Twelve items were
each verified individually and all twelve held; the re-run still found a
thirteenth, in a different package, reached by a code path no test
touched.

## What this pass does not establish

- **It does not say 408 controls are broken.** Every one of them is
  present and correct in the source; the finding is about coverage.
- **It does not rank by exploitability.** Tier 1 above is a reading, not
  a computation. Some entries may prove to be equivalent mutants.
- **The operator is narrow by design.** It suppresses refusal guards. It
  does not mutate arithmetic, boundaries, or control flow, so a clean
  result here would not mean the suite is complete.

## Tier 2 — the G1 anchor pin family (closed 2026-09-15)

`orchestration/orchestrator.go:474`, the anchored tool-registry pin, was
a survivor and was **not** on the Tier-1 list. Closing it alone would
have repeated the mistake that produced it: Tier 1 ranked LINES, and the
deployment anchor is one SURFACE enforced in three places. Swept whole
instead — eight guards, every one mutation-verified to fail only its own
subtest. Four of the eight had survived because a neighbour stood in for
them.

Recorded in `docs/development/g1-anchor-pin-sweep-2026-09-15.md`, with
the completeness criterion stated so it can be checked rather than
asserted: every `Anchor` field L7 enforces has a drift test.

## Tier 3 — two more surfaces (closed 2026-09-15)

Same unit of work as the anchor sweep: a surface with a claim attached,
not a list of lines.

**L7 static boundedness (`orchestration/workflow.go`) — 5 guards.**
Q-L7-4 enforces "every walk terminates" entirely at load, so no walk
needs a runtime escape hatch. `TestWorkflowLoaderFailsClosed` covered
the event vocabulary and ONE member of the family (counter-free
cycles); the other five had no lattice violating them. One tested
member had been standing in for its siblings — the same shape as
`local.go:153`, at the level of a rule family rather than a pair. Each
case now violates exactly one rule and asserts the refusal names it.
The sixth rule does not compile when suppressed, so it was never a
survivor.

**L9 substitution boundary (`skills/instantiate.go`) — 5 guards.**
D-L9-5 says L9 narrows and never mints; these guard the claim that a
substituted artifact is one the reviewed template could have produced.
The spec-template placeholder rule is the sharpest: a template carrying
a literal in a Class-3 subject field was reviewed with a subject already
chosen, so instantiation keeps the template author's task, repo, or
commit while the record attributes it to the caller. Wall-deadline
narrowing was open from both directions — declared twice (one copy
narrowed, one left standing) and not declared at all (the caller's
narrowing request applied to nothing, so the task runs at the template's
own bound while the caller believes it asked for less).

One detail from the mutation run worth keeping: with the positive-cap
guard suppressed, grant entries with zero or negative caps still refuse
via the narrowing-completeness rule — but an entry naming **no tool at
all**, with a consistent aggregate, was accepted outright. Partial
mutual cover: four of five cases were caught by a neighbour and exactly
one was not.

Recorded totals for the day: **31 guards closed** — Tier 1 (12), the
confirmation re-run's own find in `themis-ratchet derive` (1), the G1
anchor surface (8), L7 static boundedness (5), and the L9 substitution
boundary (5). Each was mutation-verified individually in a disposable
worktree per the AGENTS.md probe-isolation invariant.

## Disposition

**Tier 1 closed 2026-09-15** — all twelve worked through, one at a time,
each mutation-verified in a disposable worktree per the AGENTS.md
probe-isolation invariant. The ~190 error-propagation survivors remain a
recorded coverage map; triaging them wholesale would still be poor use
of attention.

### What the twelve actually were

| Disposition | Count | Items |
|---|---|---|
| Real coverage gap, closed by a test | 7 | `local.go:153`, `orchestrator.go:547`, `:215`, `:372`, `local.go:243`, `catalog.go:193`, `:85`, `package.go:129`, `contract.go:249/257` |
| Equivalent mutant, premise **unsound** | 2 | `orchestrator.go:763` (digest blind to `ThemisScope`), `loop.go:474` (two untested assembly gates) |
| Equivalent mutant, premise sound | 3 | `task.go:211`, `loop.go:382`, and the assembly declaration gate `:474` rests on |

Three findings could not have come from reading:

1. **`grantAuthorityDigest` omitted `ThemisScope`** — a real defect in a
   shipping control. Two grants with materially different themis-id
   authority recorded an identical `grant_authority`.
2. **Mutual cover is the dominant pattern.** Three of the twelve were
   pairs of adjacent guards with one test tripping both: `local.go:153`,
   the verification assembly gates, and the provenance checks. In every
   case the test was green, named the control, and evidenced neither
   half.
3. **The Tier-1 ranking itself was incomplete.** The two gates beneath
   `loop.go:474` were not on the list. They were the premise of an item
   that was — so a survivor's *dependencies* deserve the same treatment
   as the survivor.

### The rule this pass earned

An equivalent mutant is not a closed question. The right response is not
"unreachable, therefore fine" but **"unreachable because of what, and is
that tested?"** Four of the five equivalents here rested on a premise
nobody had checked; two of those premises were wrong.
