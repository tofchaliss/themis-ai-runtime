# Tasks: Themis v0 — the read door and the decision door

Grill CLOSED 2026-09-25 (D-T-1..10, B-T-1..3; `design.md` §2, Gate 0
§4). Three-state verdicts per milestone. The C17 lesson applies: the
positive path is proven before any negative is trusted. Standing rule:
anything not in Gate 0 stops implementation and is classified first.

## 0. Gate
- [x] Grill held; Q-T-1..10 disposed; owner LOCK on each (2026-09-25)
- [x] Gate 0 whitelist and five registers (design.md §4)
- [ ] Gate 1: implementation design against §4 before each milestone's
      code, recorded in `RESUME-HERE.md`

## 1. T-M1 — Module and walls (Class 3; architectural proof — owner)
- [x] `src/themis` module (`themis-app`) in the workspace; depends on
      the harness module read-only; harness never imports it
      (2026-09-25)
- [x] `store` / `intake` boundary scaffolds; `themis-decide` /
      `themis-inspect` refusing stubs; the four D-T-10 walls as tests,
      walls 1–3 mutation-killed (`RESUME-HERE.md`)
- [x] (moved to T-M2) `store`: `findings.json` / `products.json` loaders
      (append-only, strict keys, closed schema, `state`, `steward`;
      withdrawn unservable), `ThemisSeam.Read`, `themis_store` hash
      (2026-09-25)
- [x] P0 records PROPOSED: `FIND-2026-0001` (vulnerable-dep in
      `PROD-demo-vuln-app`), the Product
- [x] `policies/themis/README.md` (what a Finding/Product is in v0;
      registration checklist; Product is referential only)
- [x] Register A: loaders, withdrawn, key wall, store has no `os` writer
      (AST) — `store/store_test.go` + wall 3

## 2. T-M2 — Anchor pin and the read door in the harness (Class 3/4; G1 amendment)
- [x] `Anchor.ThemisStore` (`themis_store`, sha or `"absent"`);
      `Config.ThemisStorePath`; Open verifies pin ⇔ path like the
      delegation registry; anchor tests
- [x] `cmd/themis-run` constructs the store and passes it as the L4 seam
      (imports `store` only)
- [x] `themis-status` / `themis-preflight` print and verify the pin
- [x] `remediate-dependency@3` PROPOSED: grant template adds
      `get_finding` (`themis_scope: ["FIND-"]`) and `get_product`
      (`["PROD-"]`); ceiling `allowed_tools` extended; procedure reads
      the Finding first; catalog entry
- [x] Register B first (harness side): an anchored @3 walk with a
      scripted model calls `get_finding FIND-2026-0001`, gets a
      `governed-record` framed result, completes with L10 PASS;
      negatives: id outside scope, withdrawn Finding, unknown id, store
      bytes ≠ pin, registry modified after pinning, seam over other
      bytes, seam configured while `absent`, pin without path, class
      other than governed-record refused at load
      (`src/themis/readdoor_test.go`, 2026-09-25)
- [x] Register D: the model's restatement of the Finding re-enters at
      the floor (projection), the Finding at `governed-record`
- [x] Landed 2026-09-25 (RESUME-HERE Gate 1 T-M2; probes killed).
      `rsys@6` deliberately NOT minted — a T-M5 host act.

## 3. T-M3 — Intake (Class 3) — admissibility landed 2026-09-25
- [x] `intake.Resolve(tuple)`: D-T-1 (manifest COMPLETED/VERIFIED,
      anchor equality, event at seq is `artifact-bound`, unanchored
      refused) → D-T-2 (anchor bytes from the record, registry of the
      governed checkout, any lifecycle) → D-T-4 (causal replay over the
      links the record HAS: binding → object → COMPLETED after binding;
      writers; no competing binding — the L5 witness is an owed harness
      amendment, PROPOSED note under D-T-4) → D-T-5 (L10 reconstruction:
      registered contract, Consistent, PASS, authorized audit with
      ResultHash = raw hash, raw bytes a MEMBER of the egress manifest
      (PROPOSED note under D-T-5), verification seq < binding seq)
      → D-T-6 table; every refusal link-named
- [x] `intake.EvidenceView`: model turns shown (object ids), artifact
      identity and verified member, L10 reconstruction; the identities
      rendered, never report content
- [ ] Position record type (D-T-7/8), `Append` under `O_EXCL`, `Current`
      projection, Finding-exists check through the store — DEFERRED to
      T-M4 by the owner ("do not create the Position yet")
- [x] Register A: every refusal reachable with its own test and its
      positive twin; verify-then-mutate refused; PASS on other bytes
      refused; anchor withdrawn after → proceeds and recorded; contract
      withdrawn after → proceeds using stored bytes; forged records
      (claimed PASS over invalid bytes, record/event divergence, audit
      not the producer) refused
- [ ] Register C: Position evidence view re-derives from the record; a
      Position survives later withdrawals; corruption refuses — with
      the Position, T-M4
- [x] Landed 2026-09-25 (RESUME-HERE Gate 1 T-M3; probes 8/9 killed,
      the ninth structurally unreachable). Local only; no rsys@6.

## 4. T-M4 — `themis-decide`, `themis-inspect`, walls (Class 3) — BLOCKED until `openspec/changes/l5-witness-events/` is classified (owner, 2026-09-25)
- [ ] `cmd/themis-decide`: tuple + `--disposition` + `--rationale`;
      refuses `decision.*`; observes uid/user/host/time; records the
      Themis checkout commit and `observed-not-authenticated`; prints
      the Position path and hash
- [ ] `cmd/themis-inspect`: list Positions for a Finding, current
      projection, re-verify one Position against the record
- [ ] The four walls as tests: whole-graph dependency (`go list -deps`
      over every harness package, no `themis-app/intake`), binary wall
      (`themis-run` imports `store` only), writer wall (one `O_EXCL`
      site, no other `os` writer, no execution/model imports), capability
      wall (registry has no Position verb)
- [ ] Register B: the full positive path end to end from a real harness
      record; then the negative twins (design.md §4 B list); mutation
      both ways on every wall and every D-T link
- [ ] G2 row `themis_position` in `execution-chain.md`

## 5. T-M5 — Anchor `rsys@6`, live proof, reviews, close (Class 3/4)
- [ ] `rsys@6` PROPOSED (pin `themis_store`, catalog with @3, skills
      `remediate-dependency@1..3`) — host act; VM procedure extended
- [ ] Register E on the VM: live model reads the Finding, COMPLETED/PASS
      under `rsys@6`; `themis-decide` creates Position 1;
      `themis-inspect` re-verifies CONFIRMED; the demo script recorded
- [ ] Three Class-3 reviews in isolated worktrees; CRITICAL/HIGH
      remediated and mutation-verified
- [ ] `traceability.md`; archive; status doc; CONTEXT-MAP gains the
      Themis context; ARCHITECTURE.md untouched (no ownership change)

## Residuals (each its own gate)
- Human authentication (system-wide) · automated/governed decision
  door · L2 `ThemisReader` composition-time context · Positions in
  model-visible context · Product beyond referential · feeds/KB/SBOM ·
  verify-then-mutate workflows (need a second verification fact).
