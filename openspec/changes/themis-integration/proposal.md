# Proposal: integrate the harness with the real Themis (one Themis, its own repo)

Status: **OPEN 2026-09-25 — owner direction LOCKED, grill open (owner-led,
one question per turn).** Owner: *"Option A is the way, but make sure we
don't create another repo for Themis inside themis-ai-runtime. We will
pull the latest from the Themis repo and integrate it with
themis-ai-runtime."*

Supersedes the SHAPE of Themis v0 T-M4/T-M5 (`openspec/changes/themis-v0/`):
the locked decisions D-T-1..8 stand as requirements; D-T-9 and D-T-10 are
re-derived against the real Themis below. The L5 witness amendment
(`openspec/changes/l5-witness-events/`) is harness-only and unaffected.

## The fact that forced this

`src/themis` (module `themis-app`) re-implemented Findings, Products, and
Positions as JSON files inside the harness repo — a second security truth,
which CLAUDE.md forbids. The real Themis exists: `~/code/themis`
(`github.com/tofchaliss/themis`, public, main `fb83e82`, module
`github.com/themis-project/themis`, Go 1.25).

## What the real Themis is (facts from its code, 2026-09-25)

- Phase-3 greenfield: six independently deployable bounded-context services
  (Evidence, Registry, Knowledge, Governance, Communication, Intelligence),
  one PostgreSQL server with one database per context plus `bus` and
  `auth`; systemd units; env-configured (`deploy/node.env.example`). The
  v0.3.x monolith is frozen, reference only.
- **Governance is the authority context**: owns Findings and Enterprise
  Positions (EDR-GOVERNANCE-01). Finding identity = opaque UUID, business
  key (Release, Faultline); content = matched components (PURL, name,
  version, ecosystem, source, claim class), CVE alias, exploit signals,
  investigation stage. Position = immutable version {stance, rationale,
  actor, inputs{accepted proposal id, faultline ref, signals, review-by},
  established-at}.
- **Positions are created only by accepting a Governance Proposal**
  (`POST /findings/{id}/proposals` → `POST …/proposals/{pid}/accept`).
  Proposals come from any source; `ActorKind ∈ {human, ai, policy,
  system}`; AI may propose, never decide (DOM-0024). "AI proposes, humans
  decide" is already the constitution (CON-0002 "Proposal before truth").
- **Trust is a property of evidence** (EDR-TRUST-01): `observed |
  asserted | inferred`; AI recommendations enter as `inferred`. The EDR
  explicitly DEFERRED the Decision Proposal payload "until a second
  Decision capability exists to shape it" — the harness is that
  capability.
- **Deciders are authenticated**: platform `auth` (API keys, scopes
  `admin | read | product:<id>`); the recorded actor is derived from the
  key principal as `key:<id>`, and an unauthenticated dev identity is
  stamped `dev:` (EDR-SECURITY-01 D10). Stronger than D-T-8's
  `observed-not-authenticated`.
- **Products live in Registry** (`GET /products/{id}`, the Product →
  Microservice → Deployment → Customer graph).
- **Its "AI harness" is the Intelligence Gateway** (Finding → advisory
  recommendation). Nothing in the Themis repo references this repo.
- Repo rules that bind any change there: greenfield tree only; ADR/EDR
  wins; OpenSpec `phase3-*` changes with an EDR; `make check` before any
  commit; commit/push only on explicit ask; depguard + `tests/architecture`
  enforce ring and platform boundaries.
- **Module-path collision (flagged for Q-I-2):** the harness module is
  named `github.com/tofchaliss/themis` — the URL of the Themis REPO. A
  `go get` of that path fetches Themis, not the harness. The Themis repo
  cannot `require` the harness under its current name.

## Mapping the locked Themis v0 decisions onto the real Themis

| Locked | Stand-in (`src/themis`) | Real Themis |
|---|---|---|
| D-T-1..6 intake | `intake.Resolve` | unchanged logic; MOVES to the Themis repo (it depends only on harness read-only packages) |
| D-T-7 the act | `themis-decide` appends `positions/<f>/<n>.json` | the harness's verified execution becomes a Governance **Proposal** (proposer `ai`, evidence = the tuple + derived ids, trust `inferred`); the human decision is `acceptProposal` — a Position version, by Themis |
| D-T-8 witness | `observed-not-authenticated` | the key principal `key:<id>` (authenticated) or `dev:` (marked) — Themis's own rule |
| D-T-9 read door | JSON store, `themis_store` = hash of bytes | `get_finding` → Governance API, `get_product` → Registry API, through an HTTP seam in `src/harness/integrations/themis`; the pin must be re-derived (a live system of record has no bytes to hash) |
| D-T-10 walls | one module, four walls | cross-repo: the harness never imports Themis (the seam is HTTP); Themis's intake imports harness read-only packages only (depguard allow-list + arch test); `src/themis` DISSOLVED |
| B-T-1..3 | — | unchanged |

## Grill — question table (dependency order)

| # | Question | Recommendation (PROPOSED) | State |
|---|---|---|---|
| Q-I-1 | Topology: where does intake run, and how does it reach the harness record plane? | **LOCKED 2026-09-25 → D-I-1** (same host confirmed by the owner): Themis-owned `cmd/themis-intake` on the harness VM, tuple in, local read-only record, Proposal out over the authenticated Governance API; Governance never reads harness files; the harness never initiates a Governance act. | LOCKED |
| Q-I-2 | Harness module identity and how Themis consumes it | rename the harness module to `github.com/tofchaliss/themis-ai-runtime/src/harness`; Themis `require`s it at a pinned commit (pseudo-version), never `replace` | OPEN |
| Q-I-3 | The read door and its G1 pin | seam = HTTP client (Governance + Registry), pinned base URLs, key from env; the anchor pins the **contract** (hash of the two OpenAPI specs) not the data; data is live by design | OPEN |
| Q-I-4 | Finding identity, skill scopes, the demo Finding | `themis_scope` becomes UUID syntax; skill inputs derived from Finding content (PURL, CVE); the demo Finding is created through Themis's own pipeline or its dev seed, never hand-written | OPEN |
| Q-I-5 | The proposal act and its evidence payload | `ActorKind = ai`, `proposer_id = harness:<anchor12>@<task>#<seq>`; `RaiseProposalRequest` gains `evidence` (the tuple, derived artifact id, contract identity + reconstruction verdict, production witness); trust `inferred`; an EDR amendment closing EDR-TRUST-01's deferred payload | OPEN |
| Q-I-6 | The decision act and witness | `acceptProposal` as-is; D-T-8 satisfied by EDR-SECURITY-01 D10; `dev:` deciders allowed only outside production | OPEN |
| Q-I-7 | Walls across two repos | harness: no Themis import (wall 1 trivially true, kept as a test); Themis: `internal/governance/adapters/harness` may import only `state`, `deployment`, `verification`, `verification/seam` (depguard) + arch test; `src/themis` removed with its walls re-homed | OPEN |
| Q-I-8 | Demo topology on the VM | Postgres + Registry + Governance (+ Evidence/Knowledge for Finding creation) beside the harness on the enterprise VM; rsys@6 pins the Themis contract | OPEN |

Owner facts: the harness and the Themis services run on the SAME
enterprise VM (confirmed 2026-09-25). Still needed: confirmation the
Themis repo may be changed under this work (its own rules apply there).

## What stays, what goes

- Stays: D-T-1..8, B-T-1..3, the T-M3 intake logic and tests (they move),
  W-M1..W-M3 (harness-only, still required before any Position-eligible
  intake).
- Goes: `src/themis/store`, the `themis_store` byte pin as designed, the
  JSON Position, `cmd/themis-decide`/`themis-inspect` as stand-ins.
- Not before the grill closes: no deletion in `src/themis`, no change in
  the Themis repo.
