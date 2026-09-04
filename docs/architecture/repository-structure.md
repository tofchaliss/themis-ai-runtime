# Repository Structure — Authoritative Target Map

Owner-defined (2026-09-04, build-sequence step 3). The full directory skeleton exists from day one (`.gitkeep` placeholders); each directory gains real content as its code/artifacts land. Amendments to this map are architecture changes.

```
themis-ai-runtime/
├── ARCHITECTURE.md                  # authoritative system architecture
├── CLAUDE.md                        # Claude Code entry point
├── AGENTS.md                        # repository conventions
├── README.md · LICENSE · INSTALLATION.md · TESTING.md
├── go.work                          # workspace: per-component Go modules (see note)
│
├── src/
│   ├── themis/                      # EXISTING THEMIS APPLICATION (joins the repo here,
│   │                                #   bringing its own tree: api/ cmd/ domain/ ...)
│   └── harness/                     # AI AGENT HARNESS (Go module lives here today)
│       ├── runtime/                 # core agent runtime: agent_loop/ lifecycle/ model/
│       ├── instructions/            # L1: loader/ resolver/ validator/ versioning/
│       ├── context/                 # L2+L3: delivery/ retrieval/ ranking/ compaction/
│       │                            #        provenance/ isolation/
│       ├── tools/                   # L4: registry/ schemas/ validation/ permissions/
│       │                            #     filesystem/ git/ shell/ build/ testing/
│       │                            #     security/ themis/
│       ├── execution/               # L5: manager/ docker/ worktree/ credentials/
│       │                            #     providers/{local,daytona,e2b}/
│       ├── state/                   # L6: task/ checkpoint/ session/ memory/ persistence/
│       ├── orchestration/           # L7: planner/ scheduler/ router/ hooks/ retry/
│       │                            #     approval/ handoff/
│       ├── subagents/               # L8: runtime/ delegation/ isolation/ roles/
│       ├── skills/                  # L9 runtime: registry/ loader/ resolver/ executor/
│       ├── verification/            # L10: build/ test/ lint/ security/ evidence/
│       ├── observability/           # L10: tracing/ metrics/ logging/ events/
│       ├── ratchet/                 # L11: feedback/ evaluations/ regression/
│       │                            #      candidates/ promotion/
│       └── integrations/
│           └── themis/              # client/ contracts/ events/
│
├── agents/                          # runtime agent definitions (L8):
│   └── planner/ engineer/ security-analyst/ reviewer/
├── skills/                          # runtime SKILL.md artifacts (L9):
│   └── investigate-cve/ analyze-sbom/ remediate-dependency/
│       security-code-review/ release-security-review/
│       (each: SKILL.md + scripts/ references/ templates/ as needed)
├── instructions/                    # instruction artifacts (L1 sources):
│   └── global/ themis/ templates/
├── policies/                        # enforcement policies:
│   └── security/ tools/ execution/ credentials/ themis/
├── schemas/                         # contracts:
│   └── tasks/ agents/ tools/ context/ execution/ verification/ results/
├── evaluations/                     # harness evaluation data:
│   └── scenarios/ datasets/ regression/ security/ models/
│
├── tests/                           # cross-cutting only — Go unit tests stay
│   └── integration/ evaluation/     #   beside their source packages
│
├── docs/
│   ├── architecture/
│   │   ├── themis/
│   │   └── harness/                 # 01-instructions.md … 11-ratchet.md + baselines
│   ├── decisions/
│   └── development/
│
├── migrations/                      # Themis DB migrations (arrive with the app)
├── openspec/                        # Themis specification system (arrives with the app)
├── runtime/                         # runtime/configuration
├── scripts/                         # build/dev/operational scripts
├── infrastructure/                  # docker/ local/ cloud/ observability/
└── .claude/                         # Claude Code development standard (Stage 5)
```

## Decisions recorded with this map

- **No `pyproject.toml`** — a Python-era perception; the backend is Go (DEC-01). Python returns only with a concrete, recorded requirement.
- **Modules:** working default is `go.work` + per-component Go modules (`src/harness` today; `src/themis` keeps `github.com/themis-project/themis` on arrival — no import-path rewrites). Consolidating into a single root module is a recorded architecture decision if ever taken.
- **Existing-code destinations** (relocation is its own reviewed step, not part of skeleton creation):
  - `src/harness/internal/llm` → `src/harness/runtime/model/`
  - `src/harness/benchmarks` (Go pipeline) → `src/harness/ratchet/evaluations/`; benchmark definitions/prompts/expected → `evaluations/`
  - `src/harness/internal/service` (themis-serve) → `src/harness/integrations/themis/` — the Themis-facing seam, absorbed or retired as the real harness API lands
- **Go tests** live beside their packages (Go idiom, existing 2,000 test LOC); `tests/` holds only cross-cutting integration and evaluation suites.
- **`src/themis/` interior** is not scaffolded — the application arrives with its own tree.
- Layer documents follow `docs/architecture/harness/NN-<layer>.md` (01–11).
```
