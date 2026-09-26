# Integration Completion Matrix — Themis ↔ AI Runtime

Established 2026-09-26 at the owner's direction. **Governing principle
(owner):** build the complete Themis ↔ AI Runtime integration first;
demonstrate it only after the architecture is complete enough to
represent the real system. The demo document (`docs/demo/`) is an
integration design REFERENCE, not the implementation target.

**Project rule (owner, 2026-09-26):** no demo-driven shortcuts; no
demo-specific architecture; no declaring integration complete until the
required Themis and runtime layers and their cross-boundary invariants
are implemented and verified. Architecture first, integration second,
demonstration third.

**Completeness test:** the architecture must survive replacing the
Finding, the vulnerability, the workflow, the model, the skill, the
deployment, the Knowledge source, and eventually the deployment
topology. `rsys@6`, the six-node VM, and the one-CVE workflow are ONE
realization, never the definition of completeness.

Status legend: 🟢 implemented and verified · 🟡 decided, not (fully)
implemented · 🔴 not decided (needs its own grill before any code) ·
"—" not applicable to that side. Facts are as of 2026-09-26; each 🟢
cites its evidence.

| # | Area | Themis | AI Runtime | Integration | Status | What turns it green |
|---|---|---|---|---|---|---|
| 1 | Security truth | Governance owns Findings, Proposals, Positions 🟢 (Phase-3, VM-verified 2026-09-22) | no write path exists; four-class authority vocabulary 🟢 | authority boundary: harness never initiates a Governance act (D-I-1); no Position mechanism outside Themis (D-I-5) | 🟡 | cross-repo walls in code (row 13); `src/themis` stand-in dissolved (I-M4) |
| 2 | Finding read | `GET /findings/{id}` (FindingView) 🟢 | HTTP seam with projection, identity check, seam-local key; `themis_contract` pin verified at Open 🟢 (I-M1, 2026-09-26; proven against an httptest stand-in) | live authority read as execution-time governed record (D-I-3) | 🟡 | I-M5: proven against the running estate; the pin's `themis_commit` host check |
| 3 | Finding scope | UUID identity 🟢 | `themis_scope` `uuid` syntax class in L4; `remediate-dependency@4` scoped to it 🟢 (I-M1) | any-UUID scope now; exact-subject binding (D-I-9) deferred hardening | 🟡 | D-I-9 (post-integration) |
| 4 | Workflow commissioning | Commission = pre-execution Governance act on the Finding (D-C-1..6; not yet implemented) | `skills.Request.Commission` → sealed `origin:commission` 🟢 (I-M1) | Themis mints → runtime carries verbatim → intake extracts from CREATED → Themis equality-checks Finding/state/method/deployment at proposal time | 🟡 | I-M1 (runtime), I-M3 (Themis); `themis-commissioning/tasks.md` |
| 5 | Execution | — | L5 sealed workspace, L6 record plane 🟢 (rsys@3..5 live); class→writer invariant 🟢 (W-M1); L5 witnesses on every edge and op through the L5-scoped handle 🟢 (W-M2, 2026-09-26) | evidence the record plane can replay: five links now exist in every completed record | 🟡 | I-M3 (Themis-side five-link replay); rsys@6 (I-M5) |
| 6 | Verification | — | L10 contracts, reconstruction 🟢 | evidence contract: reproducible PASS over the egressed member (D-T-5 as ratified) | 🟡 | verified in T-M3 over the stand-in; re-proven after the move (I-M3) |
| 7 | Evidence reconstruction | Governance adapter `harness` (decided, D-I-7) | `state`, `deployment`, `verification`, `verification/seam` read-only surface 🟢; `seam.ReconstructEvent` exported 🟢 | `intake.Resolve` D-T-1..6 🟢 in the stand-in (T-M3); five-link replay + constitution-keyed obligation decided (D-W-5); provenance-bearing real-walk fixture 🟢 (I-M2) | 🟡 | I-M3 (move + five-link + witnessing table) |
| 8 | Proposal | `raiseProposal` (human/ai, no evidence field) 🟢 | execution evidence derivable 🟢 | first-class immutable `harness-execution/v1` evidence; derived trust; Business Verification from recorded Finding bytes (D-I-5) | 🟡 | I-M3 + Themis EDR (closes EDR-TRUST-01's deferred payload) |
| 9 | Human decision | `acceptProposal`, server-derived `key:<KeyID>` (EDR-SECURITY-01 D10) 🟢 | — | decision boundary: two authenticated humans, harness outside both acts (D-I-6) | 🟡 | green once row 8 lands (the decision is only complete when the accepted proposal carries the evidence) |
| 10 | Position | immutable versions citing `AcceptedProposalID` 🟢 | — | evidence by reference: Position → proposal → `harness-execution/v1` → record plane (D-I-6) | 🟡 | row 8; cold-replay test (UC9) |
| 11 | Subagents (L8) | evidence rendering + premise re-affirmation (decided, D-L-1..3) | **implemented and verified**: D-L8-1..21 / C-L8-1..21 locked, archived 2026-09-23, live COMPLETED under rsys@5 (Addendum F) 🟢 | **integration surface undecided:** does a delegate inherit the parent's Themis scope (row 3)? may a delegate read Themis at all? how do `l8-delegation` events appear in the intake evidence view (D-T-7 "model turns" today lists `model-turn` only)? | 🟡 | grill CLOSED 2026-09-26 (D-L-1..3, `l8-themis-surface/`): delegations rendered in the model-reasoning fact, referenced by seq; no admissibility rule; premise `l8-delegates-tool-less` re-affirmed per witnessing constitution. Implementation in I-M3 |
| 12 | Ratchet (L11) ↔ Governance | Themis holds no runtime-method registry, by decision (D-R-1) | L11 criteria and regression sets 🟢; registration and reliance doors exist 🟢; decision records two-way bound at both doors, D-R-4 walls 🟢 (I-M1) | L11 → human → decision record → existing door; no automation path (D-R-4); commission validity independent of the doors (D-R-3) | 🟡 | I-M5: host-side rendering of `decision_ref`; then green |
| 13 | Cross-repo dependency walls | depguard + `tests/architecture` 🟢 for its own rings | wall 1 for the stand-in 🟢 | exactly one Themis package imports the harness; harness imports no Themis; no writer/exec across (D-I-7) | 🟡 | I-M3 (Themis side), I-M4 (harness side) |
| 14 | Authentication | API keys, scopes, `authadmin` 🟢; **known gap (D-C-6): `AuthorizeWrite` does not confine `product:<id>` to that product's Findings on Governance writes** | scoped capabilities per task 🟢 | integration identity: read key (harness), product write key (operator), write key (decider); key ids in evidence, never values (D-I-8); separation of duties operational, not a Governance invariant (D-I-6, D-C-6; future grill) | 🟡 | I-M1 (seam key handling), I-M5 (host); the product-scope gap is a Themis security EDR item, kept visible here |
| 15 | Harness module identity | — | renamed to `github.com/tofchaliss/themis-ai-runtime/src/harness` 🟢 (I-M0, 2026-09-26; both constitution hashes pinned and unchanged) | pinned pseudo-version in Themis (D-I-2) | 🟡 | I-M3 (Themis `require`) |

## What already exists on the Themis side (not to be rebuilt)

Findings · Products/Projects/Releases · Governance proposals and
positions · Knowledge and Evidence boundaries · authentication and
authorization · outbox/event-bus semantics · REST APIs per context ·
`authadmin` CLI. All Phase-3, live-verified on the VM (PHASE3-STATUS
2026-09-22). The Themis-side work of this integration is rows 4, 7, 8,
11, 12, 13 — nothing else.

## Order of work (owner)

1. Themis domain/application architecture — exists; close rows 4 and 12
   by grill, then implement Themis-side rows 7, 8, 13.
2. Runtime integration layer — rows 2, 3, 5, 6, 14, 15 (I-M0, W-M1,
   W-M2, I-M1, I-M2).
3. L8 integration surface — row 11 by grill, then implement.
4. Cross-repository invariants — row 13 both directions, verified.
5. Then the real end-to-end path; only then decide whether a demo is
   worthwhile.

## Cross-repository invariants (owner, restated as tests to write)

- Themis never depends on the runtime's orchestration (import wall +
  arch test, Themis side).
- The runtime never acquires authority over Themis security truth (no
  Position/Proposal capability in any registry; wall 4 re-homed).
- The human decision remains in Themis (`acceptProposal` is the only
  Position creator; asserted by Themis's domain tests).
- AI remains advisory (model output enters at the floor; proposal
  proposer is never the model or the harness).
- Evidence remains reconstructable (cold replay from the record plane
  yields the same evidence view; fixture provenance rule).
- Authorization remains deterministic (L4 exact-id scope; Themis scope
  checks; no model-influenced grant).
- Neither repository gains accidental authority through integration
  code (exactly-one-importer rule; seam is HTTP only).
