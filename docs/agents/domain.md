# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT-MAP.md`** at the repo root — it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/decisions/`** — read ADRs that touch the area you're about to work in. Also check `src/<context>/docs/decisions/` for context-scoped decisions.
- **`ARCHITECTURE.md`** remains the authoritative system architecture; `CONTEXT.md` files carry domain vocabulary and reference it, never redefine it.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## File structure

This is a multi-context repo (`CONTEXT-MAP.md` at the root). ADRs use the repo's existing `docs/decisions/` convention at both levels (not `docs/adr/`):

```
/
├── CONTEXT-MAP.md                     ← points at each context
├── docs/decisions/                    ← system-wide ADRs
└── src/
    ├── harness/
    │   ├── CONTEXT.md                 ← harness domain language (created lazily)
    │   └── docs/decisions/            ← harness-scoped ADRs
    └── themis/                        ← planned; joins the map when it lands
```

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal — either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders) — but worth reopening because…_
