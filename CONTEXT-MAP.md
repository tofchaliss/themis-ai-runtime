# Context Map

| Context | Path | CONTEXT.md |
| --- | --- | --- |
| Harness | `src/harness/` | [`src/harness/CONTEXT.md`](./src/harness/CONTEXT.md) |
| Themis (system of record) | its own repository `github.com/tofchaliss/themis` (module `github.com/themis-project/themis`) | integrated 2026-09-26 (`openspec/changes/themis-integration`, EDR-HARNESS-01 there): the harness reaches it only through the HTTP read door (`src/harness/integrations/themis`), Themis reads the harness record plane only through its own `themis-intake`; no Themis code lives in this repository |

System-wide ADRs: `docs/decisions/`. Architecture authority: `ARCHITECTURE.md`.
