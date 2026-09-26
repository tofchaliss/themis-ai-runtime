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

## I-M1 — Harness read seam and contract pin (D-I-3, D-I-4, D-C-5, D-R-2/4; Class 3) — LANDED 2026-09-26
- [x] `integrations/themis/contracts`: closed-schema loader for
      `policies/themis/contract.json` (URLs, two spec hashes, Themis
      commit; Lstat-bounded, no dup keys, no unknown fields)
- [x] `integrations/themis/client`: HTTP `ThemisSeam` over Governance
      `GET /api/v1/findings/{id}` and Registry `GET /api/v1/products/{id}`;
      projection by construction (id, release, faultline, CVE, stage,
      components / id, name — positions and proposals never decoded);
      response-id equality; UUID-only ids; key from env, presented as
      `X-API-Key`, never in served bytes or errors; HTTP error/404 →
      `ErrUnavailable` → L4 `seam-unavailable`
- [x] Anchor `themis_store` → `themis_contract`; Open verifies the
      contract file hash against the pin AND the seam's `ContractHash()`;
      `absent` ⇔ no path and no seam; per-task re-verification kept
- [x] L4 `themis_scope` gains the `uuid` syntax class (grant-template
      value interpreted by L4 — NOT a tool-registry change; registry-v5
      stays: the scope is a grant property)
- [x] `remediate-dependency@4`: UUID-scoped `get_finding`, no `get_product`,
      procedure reads the Finding by UUID and corroborates dependency/CVE;
      catalog entry `be639874…`
- [x] `skills.Request.Commission` → sealed `origin["commission"]`;
      `themis-instantiate --commission`; syntax-only validation; never in
      the payload (D-C-5)
- [x] Door provenance (D-R-2): `decisions` package (closed schema,
      read-only, two-way `Bind`); catalog and both anchors registries at
      `version: 2` with `decision_ref` + `decision_sha256` on every entry;
      11 records under `policies/decisions/` written from the archived
      ratifications (actor `commit:tofchaliss`, evidence `[]` stated);
      loaders refuse missing/mismatched/retargeted records; v1 observed
      copies still parse (identity comparison only)
- [x] D-R-4 walls: door loaders (`skills` catalog/manifest, `deployment`,
      `decisions`, the Themis client/contracts) have no `os` writer, no
      `os/exec`, no `net`; `ratchet` and `themis-ratchet` import no door
      package (L9 instantiation's envelope writer is named and excluded)
- [x] `themis-run` builds the door from the contract + `THEMIS_API_KEY_READ`;
      `themis-status` prints `themis_contract` (and its embedded helper now
      uses the renamed module — an I-M0 miss); preflight validates the
      contract and warns when the read key is unset; runbook pin row
- [x] Old JSON store removed (`policies/themis/{findings,products}.json`,
      `src/themis/store`); the stand-in's world now opens over an httptest
      Governance/Registry serving the full FindingView; positive path proves
      projection, credential presented and never recorded, laundering;
      refusals: pin/no door, absent/door, bytes≠pin, modified after
      pinning, door over another contract, 404 and down → seam-unavailable,
      prefixed id → L4 scope denial, class minting, L2 unwired
- [x] Hermetic suites green in both modules; evidence tools build

## I-M2 — Fixture with provenance (D-I-7; Class 2) — LANDED 2026-09-26
- [x] `evidence/harness/fixtures/themis-intake/1df0e28548a4/`: a real
      `remediate-dependency@4` walk (test anchor `test-rsys@6`, HTTP read
      door, commission `c0a8e3d6-…` cited in origin, report verified once
      and egressed unchanged) — `state/` (36 files, 160 KB), `anchors.json`,
      `verification/`, `contract.json`, `provenance.json`
- [x] Generator + verifier `src/themis/fixture_test.go` (moves to the
      harness integration suite in I-M4): default mode fails loudly on a
      stale constitution hash, an unverified record, or a missing link
- [x] Regeneration rule in the fixtures README

## I-M3 — Themis repo: intake, evidence, walls (D-I-1, D-I-5..7, D-W-5, D-C-*, D-L-*; Class 3 there) — IMPLEMENTED 2026-09-26, uncommitted there by that repository's rule
Branch `feat/harness-integration` in `~/code/themis` (commit only on the owner's explicit ask).
- [x] `docs/engineering/decisions/EDR-HARNESS-01.md` (D1–D9) + `openspec/changes/phase3-harness-integration/` + BACKLOG note
- [x] Domain: `Commission` on the Finding aggregate (premise, forward-only withdrawal, human only, Archived refused, no stage change); `HarnessExecution` evidence (`Validate`, `DerivedTrust` = inferred, `Corresponds`); `NewHarnessProposal`; thin events
- [x] App: `Commission`, `WithdrawCommission`, `RaiseHarnessProposal` (shape → trust → commission on this Finding → correspondence → Business Verification from recorded Finding refs → raise)
- [x] Store: migration `000014_harness_commissions`; commissions and proposal evidence round-trip (integration test under embedded Postgres, green); both event types frozen with schemas (contract test green)
- [x] API: commission paths, `HarnessExecutionEvidence`, `RaiseProposalRequest.evidence`+`evidence_trust`, views; regenerated; handler routing with problem statuses
- [x] `internal/governance/adapters/harness`: moved intake (D-T-1..6) + five-link replay + witnessing-constitution table with `l8-delegates-tool-less` + evidence mapping; the ONLY importer of the harness (depguard allow-list of four read-only packages; `tests/architecture/harness_test.go`)
- [x] `cmd/themis-intake`: tuple in, evidence view (three facts incl. delegations), proposal out with the derived trust; commission DERIVED from CREATED, no flag
- [x] Tests over the copied fixtures: table guard, positive five-link resolve, refusals, historical branch, verify-A/egress-B refusal over the real mismatch fixture, forged records via L6 primitives (claimed PASS over invalid bytes, audit not the producer, egress witness omitted, wrong seal reason) — all green; lint 0 issues; `make clean-arch` green
- [ ] `go.mod` pin `require …/src/harness v0.0.0-<pseudo>` — AFTER the harness push (until then the Themis repo builds only inside its local, untracked `go.work`)
- [ ] `make check` end to end on a host with the pin resolvable

## I-M4 — Dissolve `src/themis` (D-I-7; Class 2) — LANDED 2026-09-26
- [x] Moved into `src/harness/integration`: the Themis-door world and read-door tests (`themisdoor_test.go`), the L5 Register B walk test and walk helpers (`themisl5_test.go`), the fixture generator/verifier now producing the positive AND the verify-A/egress-B fixtures (`themisfixture_test.go`), and the two surviving walls — no harness package imports `github.com/themis-project/themis`; the capability wall (`themiswalls_test.go`)
- [x] The stand-in's `intake` package, its Resolve tests, `themis-decide`/`themis-inspect` stubs, walls 2–3, `go.mod`: deleted; their consumer-side equivalents live in the Themis repository (I-M3)
- [x] `go.work` lists `src/harness` only; harness `go.mod` carries no `themis-app`; CONTEXT-MAP, repository-structure, fixtures README updated
- [x] Full hermetic harness suite green; evidence tools build

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
