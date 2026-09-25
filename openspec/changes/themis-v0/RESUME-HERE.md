# RESUME-HERE — Themis v0 implementation (T-M1..M5)

Updated at every green milestone. If context was compacted, start here.

## One-line status
2026-09-25 (latest): T-M3 LANDED (admissibility only, per owner) —
`intake.Resolve` D-T-1 → D-T-2 → D-T-4 → D-T-5 under the D-T-6 table,
every refusal typed and link-named, `EvidenceView` by identity; the
owner's key case proven both ways (verify A → egress B refused
`verified bytes are not the bound artifact`; re-verify B → admitted on
the second fact); forged-record tests prove Themis re-establishes
rather than trusts the record's claims; nine probes run, eight killed,
one structurally unreachable. NO Position, NO rsys@6, NO push. Two
record-shape facts recorded as PROPOSED implementation notes under
D-T-4/D-T-5 for the owner (below). Owner ratified both as-recorded
(same day); the L5 witness gap is OPEN as
`openspec/changes/l5-witness-events/`. Next: owner classifies it
(accepted v0 residual | required amendment), then T-M4. Earlier: T-M2 LANDED — store loaders + `Read`, `themis_store`
anchor pin verified at Open (pin ⇔ path, seam hash ⇔ pin), read seam
wired in `themis-run`, `remediate-dependency@3` PROPOSED, preflight and
status print the pin; positive chain anchor → store → seam → L4 →
governed-record → model proven, seven negatives, laundering register;
four new controls mutation-killed. Next: T-M3 (intake) after its Gate 1
section here. Earlier: T-M1 LANDED — module `src/themis` (`themis-app`) in the
workspace, packages `store` and `intake` as boundary-only scaffolds,
`cmd/themis-decide` and `cmd/themis-inspect` as refusing stubs, the
four D-T-10 walls as tests (walls 1–3 mutation-killed; wall 4 is a
decoded-registry check). Next: T-M2 (store loaders, `themis_store`
anchor pin, `themis-run` read seam, `remediate-dependency@3`) after its
Gate 1 section here.

## Decisions a newcomer must not re-derive
`design.md` §2 D-T-1..10 and B-T-1..3 (locked), §4 Gate 0. Anything
outside Gate 0 stops implementation and is classified first.

## Gate 1 — T-M1 (architectural proof, not functional)

Owner's rule for this milestone: build the module scaffold and prove
the dependency graph before any store/intake behaviour; if the
physical graph is wrong, everything above it is built on the wrong
architecture.

- **Module:** `src/themis/go.mod` = `github.com/tofchaliss/themis-app`,
  `require github.com/tofchaliss/themis v0.0.0` with a `replace` to
  `../harness`; `go.work` uses both. The harness `go.mod` is
  untouched — the dependency is one-way by construction, and wall 1
  proves it stays so.
- **Packages:** `store` (declares the `Seam` shape locally so it
  imports no harness execution package; loaders and `Read` in T-M2);
  `intake` (imports `state`, `deployment`, `verification` read-only;
  the `Tuple` type; `Resolve`/`EvidenceView`/`Append`/`Current` in
  T-M3). `cmd/themis-decide`, `cmd/themis-inspect` exist and exit 2
  with "not implemented — T-M4".
- **Walls (`src/themis/walls_test.go`):**
  1. whole-graph dependency: `go list -deps ./...` in the harness
     module contains no `themis-app/*` package (not only `intake`, and
     not only direct imports);
  2. binary: `go list -deps ./cmd/themis-run` reaches `themis-app/store`
     at most, never `intake`;
  3. writer: `store` has no `os` writer; `intake` has at most one
     `os.OpenFile` site and imports none of tools / orchestration /
     execution / runtime/model / instructions / context, nor os/exec,
     net, math/rand, io/ioutil (AST over non-test files);
  4. capability: every anchored registry's DECODED tools carry no
     Position-ish name, only the closed target classes, and only
     `get_finding`/`get_product` mint `governed-record`; the executor
     table has no such verb. Decoded registrations, never prose (a
     description saying "decides what it means" is not a capability).
- **Probes 2026-09-25:** harness package importing `intake` → wall 1
  fails; `themis-run` importing `intake` → wall 2 fails; an `os.WriteFile`
  in `store` → wall 3 fails. Restored after each.
- **Not in T-M1:** any behaviour, any anchor change, any Skill change,
  `rsys@6` (a later host act, tied to T-M5).

## Gate 1 — T-M2 (owner constraints, 2026-09-25)

1. `themis_store` is a G1 anchor pin, verified inside the existing
   instruction-plane check at Open and per task: `absent` ⇔ no path and
   no seam; a pin ⇔ the two registries' exact bytes (findings then
   products) hash to it, AND the wired seam's `StoreHash()` equals it.
   L7 computes the hash from the files and imports nothing of Themis.
   Not a second deployment authority.
2. `store` is read-only: `Lstat`-then-open bounded reads, exact-key /
   no-duplicate wall, closed schemas (`DisallowUnknownFields`), records
   retained as their EXACT bytes; no writer (wall 3).
3. `ThemisSeam.Read(kind, id)` is the only read path; L4 mints
   `governed-record` from the registration; the store carries no class
   field; withdrawn and unknown are `ErrUnavailable` → L4
   `seam-unavailable`; `themis_scope` stays the authorization boundary.
4. L2 `ThemisReader` untouched and asserted unwired (no Themis bytes in
   any `l2-delivery`).
5. `remediate-dependency@3`: ceiling and ANALYZE gain `get_finding` and
   `get_product`; grant template scopes them `FIND-` / `PROD-`; the
   input schema gains `finding`; the procedure says to read the
   Finding first; no new authority class, within every ceiling.
   `investigate-cve` untouched. Finding schema kept at what the demo
   needs: a Finding references a Product; a Product is name + version.

## Gate 1 — T-M3 (owner constraints, 2026-09-25)

Owner's rule: intake/admissibility machinery only; no Position
creation; the key test is L10 PASS on report A with egress of report B
→ `verification-refused: verified bytes are not the bound artifact`,
plus the inverse positive; keep f730afb/a18b901 local; no rsys@6.

**Owner disposition 2026-09-25 — LOCK / RATIFY AS-RECORDED:** D-T-4
current evidence accepted; L5 witness absence → separate harness
amendment `openspec/changes/l5-witness-events/` (OPEN; Themis never
synthesizes L5 witnesses; a proposed event class must not quietly
become an implemented fact source); D-T-5 manifest-defined egress
artifact accepted (owner's diagram now in design.md); the
`verification/seam` transitive dependency is the documented scope of
Wall 3, no architecture change. **T-M4 does not start until the L5
item is classified** (accepted v0 residual | required amendment).

**Two record-shape facts the implementation had to classify** (both
written as implementation notes under D-T-4 and D-T-5 in `design.md`,
now RATIFIED):

1. **No L5 witness events exist.** `l5-transition`/`l5-op` are
   constitution classes with no writer. D-T-4's replay runs over the
   links that exist (binding → object → COMPLETED-after-binding → no
   competing binding → manifest projection → manifest task id), each
   link-named on refusal. L5 witnessing is an OWED HARNESS AMENDMENT;
   Themis does not simulate it. Recommendation: ratify as-recorded,
   open the harness amendment separately.
2. **The egress artifact is the L5 manifest, not the report.** D-T-5
   (3) is implemented as manifest MEMBERSHIP by content hash and bytes
   (`new_hash == sha256(raw) && content == raw`, non-deleted), with the
   member path recorded. This is the property the decision states.
   Recommendation: ratify as-recorded.

**What landed** (`src/themis/intake/intake.go`, `src/themis/intake_test.go`):
- `Resolve(root, Checkout{anchors, contracts}, Tuple)`: Themis's OWN
  checkout registries are a deployment property, never per-call input.
  Refusal classes `ErrNotReferencable` (D-T-1/D-T-6 rows 1–2),
  `ErrDeployment` (D-T-2), `ErrProvenance` (D-T-4), `ErrVerification`
  (D-T-5); anchor and contract lifecycle STATE AT INTAKE recorded, never
  refused (D-T-6 "proceed" rows).
- D-T-2: anchor bytes come from the record's `materialized-governed-
  artifacts` event (the id must be one the event references), verified
  by the harness's own `deployment.VerifyAnchorRecord` against Themis's
  anchors registry.
- D-T-5: last `l10-verification` before the binding; the seam's
  reconstruction must be Consistent with no missing inputs and PASS;
  contract two-way registered on Themis's contracts registry (bytes
  registered under another name@version refuse); `execution_ref` →
  `l4-audit` by l4, `authorized`, same capability, same authorizing
  registry, `ResultHash == sha256(raw)`; raw bytes a manifest member.
- `Resolution.View()`: the three facts by identity; a test asserts no
  report content appears in the view.
- Harness change (Class 2, read-only): `seam.reconstructOne` exported
  as `seam.ReconstructEvent` (pure); `ReconstructTask` unchanged in
  behaviour.
- Fixture: the scripted parent gained `report` (turn-6 content) and
  `tail` (turns 8..) so a walk can verify one thing and egress another;
  `forgeRecord` builds a second task from a real walk's objects through
  the L6 primitives with one alteration, and its un-altered twin is
  admitted (so refusals are the alteration's).
- Probes (mutation, restored after each): drop member check → killed by
  the key test; skip contract registration → killed; ignore tuple seq →
  killed; accept any status → killed; record-only anchor identity →
  killed; take the FIRST verification instead of the last → killed by
  the inverse positive; trust the recorded outcome (force Consistent)
  → killed by the forged "PASS over an invalid report"; drop audit
  ResultHash equality → killed by the forged audit; drop
  COMPLETED-after-binding → SURVIVED, structurally unreachable
  (`BindArtifact` refuses on a terminal task), kept as defence in
  depth and named here so nobody claims a test for it.
- `intake` imports `verification/seam` directly (D-T-10 whitelist);
  the seam's evaluator half imports orchestration/tools/model, so
  those are TRANSITIVELY in intake's graph. Wall 3 is a direct-import
  wall by design (importing grants no call site); recorded so the
  wall is not later read as claiming more.

## Milestone log
- [x] T-M3 — green 2026-09-25 (`intake_test.go`: positive, key
  negative, inverse positive, 12 refusals + 2 proceed rows, 4
  unavailability rows, 4 forged-record cases; probes 8/9 killed)
- [x] T-M2 — green 2026-09-25 (`store` Register A; `readdoor_test.go`
  positive chain + 7 negatives + laundering; probes: pin-vs-path, seam
  hash, absent-with-seam, withdrawn-served — all killed)
- [x] T-M1 — green 2026-09-25 (walls 1–4; probes killed)

## Gaps found while testing
1. `GOFLAGS=-mod=mod` is illegal in workspace mode; `go list -deps`
   runs plain. Recorded so the wall test is never "fixed" by disabling
   the workspace.
2. Wall 4's first draft substring-matched registry prose and flagged
   "decides what it means" and "delegation-template"; rewritten over
   decoded tool names/targets/trust. The claims-more-than-it-checks
   class, caught before commit.
3. **`os.Open` follows symlinks:** a handle `Stat` alone reported a
   symlinked registry as regular and the store loaded it; the loader
   now `Lstat`s the path first (the harness loaders' rule). Caught by
   the symlink refusal test before commit.
4. **Fixture kept only the last phase's conversation:** the read-door
   positive test looked for the `get_finding` result in the REMEDIATE
   request, where it cannot be (each phase composes afresh); the
   parent now accumulates every tool message across phases. The walk
   was correct throughout.
5. **Wall 1 needed an exclusion:** `cmd/themis-run` legitimately
   reaches `store`; wall 1 now covers every harness package except
   that binary, which wall 2 governs (store only, never intake).
6. **Provenance checks fired in the wrong order:** the competing-
   binding scan ran before the named event was checked, so a tuple
   naming a model-turn seq refused as "competing artifact-bound"
   instead of "not artifact-bound". Link-named means the FIRST failed
   link in causal order; reordered, caught by the refusal tests.
7. **A forged record is the only way to kill "trust the claim":** a
   walk-produced record is always self-consistent, so probes P2/P8
   survived until `forgeRecord` (L6 primitives, one alteration) gave
   the tests a record whose claim is false. Recorded so the next
   milestone does not mistake "no walk can produce it" for "tested".
