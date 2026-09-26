# Tasks: Themis integration (I-M0..I-M5), interleaved with the L5 witness amendment

Grill CLOSED 2026-09-26 (D-I-1..9). Milestones are gated one at a time
by the owner; each lands hermetically green with probes killed and a
Gate 1 note in `themis-v0/RESUME-HERE.md` (the single resume point).
Harness-repo work follows this repo's rules; Themis-repo work follows
THAT repo's rules (greenfield tree, EDR + `phase3-*` change, `make
check`, commit/push only on explicit ask). No push without the owner's
word. `rsys@6` is a host act (I-M5), never minted from the laptop.

## Order (locked by D-I-2 sequencing and D-I-8)
I-M0 → W-M1 → W-M2 → I-M1 → I-M2 → I-M3 (absorbs W-M3) → I-M4 → I-M5 (absorbs W-M4).
`themis-v0` T-M4/T-M5 are SUPERSEDED by I-M3/I-M5.

## I-M0 — Harness module rename (D-I-2; Class 2, mechanical) — LANDED 2026-09-26
- [x] `github.com/tofchaliss/themis` → `github.com/tofchaliss/themis-ai-runtime/src/harness`
      in one commit: 110 Go files, 3 `go.mod` (`go.work` needs no change: it lists directories), 7 doc/procedure refs;
      openspec decision records keep the old name as history
- [x] Both constitution hashes asserted UNCHANGED by new pin tests
      (`state/constitution_pin_test.go` 33c6f6c5…, `orchestration/constitution_pin_test.go` 008be050…); anchors untouched
- [x] Hermetic suites green in both modules (live Ollama proofs skipped via unreachable endpoint); evidence tools build with `GOWORK=off`

## W-M1, W-M2 — see `openspec/changes/l5-witness-events/tasks.md` (harness-only)

## I-M1 — Harness read seam and contract pin (D-I-3, D-I-4; Class 3)
- [ ] `src/harness/integrations/themis/client`: HTTP `ThemisSeam` over
      Governance `GET /findings/{id}` and Registry `GET /products/{id}`;
      projection (id, release, faultline, CVE, stage, components / id,
      name); response-id equality before returning; key from env only;
      HTTP error/404 → `ErrUnavailable`
- [ ] Anchor field `themis_store` → `themis_contract`; `policies/themis/contract.json`
      (URLs, two spec SHA-256s, Themis commit); Open verifies the file
      hash against the pin and configures the seam from it; `absent` kept
- [ ] Tool registry v6: `themis_scope` `uuid` syntax; `remediate-dependency@4`
      (UUID-scoped `get_finding`, no `get_product`); catalog entry
- [ ] `skills.Request.Commission` → `origin:commission`; `themis-instantiate --commission` (D-C-5; `themis-commissioning/tasks.md`)
- [ ] `themis-run`, `themis-status`, `themis-preflight`, runbook updated
- [ ] Tests against an httptest Governance/Registry stand-in: projection,
      identity mismatch refused, pin mismatch refused, unavailable fails
      closed, key never in any record body; the T-M2 read-door tests
      re-homed here; probes killed
- [ ] Remove `policies/themis/{findings,products}.json` and the byte-pin path

## I-M2 — Fixture with provenance (D-I-7; Class 2)
- [ ] A harness test generates a REAL completed record (post-W-M2, five-link)
      into `evidence/harness/fixtures/themis-intake/<constitution12>/` with
      provenance metadata (harness commit, constitution hash, anchor hash,
      task id, artifact-bound seq, generator test name)
- [ ] Documented regeneration rule: whenever the constitution hash moves

## I-M3 — Themis repo: intake, evidence, walls (D-I-1, D-I-5..7, D-W-5; Class 3 there)
- [ ] EDR (Decision Proposal payload — closes EDR-TRUST-01's deferral) +
      `phase3-*` OpenSpec change
- [ ] `internal/governance/domain`: `harness-execution/v1` evidence value
      type (plain data); proposal gains immutable evidence; migration
      `finding_proposals.evidence JSONB`; API `RaiseProposalRequest.evidence`
      (shape-validated); trust class validated against the derivation rule
- [ ] `internal/governance/adapters/harness`: moved `intake.Resolve`
      (D-T-1..6) + five-link replay + witnessing-constitution table
      (D-W-5) + deterministic trust derivation + Resolution → evidence
      mapping; depguard allow-list (four harness packages) and denies;
      no `os` writer / `os/exec`
- [ ] Commissioning (D-C-1..6): `Commission` on the Finding aggregate, migration, API, events, `raiseProposal` correspondence check (`themis-commissioning/tasks.md`)
- [ ] `cmd/themis-intake`: tuple + `--stance` + `--rationale` (commission id DERIVED from CREATED, no flag)
      (+ `--review-by`); renders the evidence view; raises the proposal
      with the operator's key; Business Verification refs from the
      RECORDED Finding bytes; consumes the adapter only
- [ ] Arch tests: exactly one package imports the harness module; the
      binary's transitive deps contain no harness execution package
- [ ] Tests over the I-M2 fixture (constitution hash asserted) + forged
      records via `state` primitives; `make check` green

## I-M4 — Dissolve `src/themis` (D-I-7; Class 2)
- [ ] One commit mapping each former component to its new owner; harness
      Wall 1 rewritten as "no import of `github.com/themis-project/themis`";
      `go.work`, evidence tools, docs cleaned; hermetic suites green

## I-M5 — Host acts, demo, reviews, archive (D-I-8, D-W-4; Class 4: host)
- [ ] Themis side: keys (three holders), auth required, contract commit
      check, demo module + SBOM + Finding through the pipeline (fixture
      feed if needed); provenance recorded
- [ ] Harness side: `rsys@6` single mint (W-M1 constitution, `themis_contract`,
      registry v6, catalog @4); opened, validated; `rsys@5` withdrawn;
      demo execution under `rsys@6`
- [ ] Decision side: `themis-intake` by the operator; `acceptProposal` by
      the decider; no `dev:` witness
- [ ] Evidence: vm-verify before/after, phasec, Addendum G, Themis records;
      VM procedure extended; architecture/security/test reviews (worktrees);
      archive `themis-v0`, `l5-witness-events`, `themis-integration`
