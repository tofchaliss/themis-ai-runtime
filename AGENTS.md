# AGENTS.md — Repository Engineering Conventions

Concrete conventions for any agent working in this repository. Orientation: `CLAUDE.md`. Architecture: `ARCHITECTURE.md`. Engineering constitution: `.claude/skills/themis-harness-engineering/SKILL.md`. Prohibitions: `.claude/policy/DAY-0.md`.

## Toolchain and structure

- Go 1.24+; the only third-party dependency today is `spf13/cobra`. New dependencies are minimal and individually justified.
- Monorepo: one Go module at `src/harness/` (root `go.work`); the Themis application joins under `src/` later. Run root-level Go commands with `./src/harness/...` patterns.

## Build

```bash
go build -o bin/themis-bench ./src/harness/benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./src/harness/cmd/themis-serve
```

## Test / lint

```bash
go test ./src/harness/...                                  # all tests
go test ./src/harness/internal/service/ -run TestRouter    # single test
go vet ./src/harness/...
gofmt -l .                                                 # CI fails on any output
cd src/harness/benchmarks && make check                    # fmt + vet + test
```

No live model is needed to build or test: unit tests use `httptest` mock backends and `t.TempDir()` fixtures. The harness fails fast on model errors rather than saving error payloads as runs.

## Testing conventions

- Table-driven `t.Run` tests; helpers take `t.Helper()`.
- Anything that talks HTTP gets an `httptest` server — never a live endpoint in unit tests.
- Filesystem tests build fixtures in `t.TempDir()` — never the repo's own data directories.
- Tests must not depend on a live model endpoint; provider-specific behavior is mocked behind the model interface.
- A new validator needs: happy path, scoring edge cases, malformed-spec error tests, a dispatch case in `validator.go`, and a line in the READMEs.

## Generated artifacts (never commit)

`bin/` (any depth), `src/harness/benchmarks/{runs,responses,validation,reports}/`, and `models.json` are generated or machine-specific and gitignored. Registry entries name API keys only by environment variable (`api_key_env`) — never values.

## Benchmark suite specifics

- Pipeline: `run → evaluate → validate → report` (+ `compare`, `gate`), date/model-scoped directories; cross-stage file contracts are covered by `src/harness/benchmarks/internal/benchmark/e2e_test.go` — check it when changing the on-disk format.
- Each benchmark = `definitions/BXXX.json` + `prompts/BXXX.md` + `expected/BXXX.json`; the loader validates specs at run time (no Go test needed for a new benchmark).
- Full pipeline: `cd src/harness/benchmarks && make bench MODEL=<model> [ENDPOINT=..] [DATE=..] [VARIANT=..]`.
- Service: `./bin/themis-serve -addr :8080 -benchmarks-root src/harness/benchmarks`.

## Documentation to keep in sync

`README.md`, `INSTALLATION.md`, `TESTING.md`, `src/harness/benchmarks/README.md`, `ARCHITECTURE.md`, and `docs/` (architecture baseline and project report — amend, don't silently override).
