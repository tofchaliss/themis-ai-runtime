# Repository Implementation Conventions

This file defines repository-scoped implementation conventions. It does not redefine architecture or the security constitution.

## Before Changing Code
- Inspect the repository, existing architecture, implementation, interfaces, configuration, and tests.
- Identify existing capabilities before creating new ones.
- Preserve established interfaces and package boundaries unless the change requires otherwise.
- Prefer the smallest coherent implementation.

## Go
- Follow the repository's declared Go version and module configuration.
- Follow existing package naming and structure.
- Keep dependencies minimal and individually justified.
- Avoid introducing frameworks/dependencies for convenience.
- Keep generated code distinguishable from hand-written code.

## Testing
- Add/update tests for behavioral changes.
- Prefer deterministic, repeatable tests.
- Security-sensitive behavior requires negative-path testing.
- Run applicable formatting, build, test, and static-analysis checks.

## Configuration and Secrets
- Never commit secrets or use real credentials in tests.
- Treat behavior-changing configuration as implementation changes.

Architecture and Day-0 rules are governed by `ARCHITECTURE.md` and `.claude/policy/DAY-0.md`.

---

## Repository Specifics (added at install — this repo's concrete conventions)

### Toolchain and commands

Go 1.24+; one module at `src/harness/` (root `go.work`); the Themis application joins under `src/` later. From the repo root:

```bash
go build -o bin/themis-bench ./src/harness/benchmarks/cmd/themis-bench
go build -o bin/themis-serve ./src/harness/cmd/themis-serve
go test ./src/harness/...        # all tests (no live model needed)
go vet ./src/harness/...
gofmt -l .                       # CI fails on any output
cd src/harness/benchmarks && make check   # fmt + vet + test
```

Testing style: table-driven `t.Run`; helpers take `t.Helper()`; HTTP via `httptest`; filesystem fixtures in `t.TempDir()` — never the repo's own data directories. Generated artifacts (`bin/` at any depth, `src/harness/benchmarks/{runs,responses,validation,reports}/`, `models.json`) are gitignored and never committed; registry entries name API keys only by environment variable (`api_key_env`).

### Commit conventions (owner-mandated, from Day-0 of development)

1. Never add a Claude signature or co-author/session trailer to any commit.
2. Always honor `.gitignore`; never force-add ignored paths.
3. Never commit binaries that bloat the repo (built artifacts, model files, archives).
4. Never commit API keys or tokens.
5. Documents lint clean before check-in: staged Markdown must have no merge-conflict markers, no unbalanced code fences, and no broken relative links (mechanically enforced by `doc-lint-guard`; run `.claude/hooks/doc-lint-guard --all` to check staged docs manually).
