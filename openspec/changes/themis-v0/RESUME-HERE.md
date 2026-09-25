# RESUME-HERE — Themis v0 implementation (T-M1..M5)

Updated at every green milestone. If context was compacted, start here.

## One-line status
2026-09-25 (later): T-M2 LANDED — store loaders + `Read`, `themis_store`
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

## Milestone log
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
