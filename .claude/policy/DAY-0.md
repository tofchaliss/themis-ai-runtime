# DAY-0 — Authoritative Prohibitions

This file is the single authoritative home of the Day-0 prohibitions. Other mechanisms (CLAUDE.md, SKILL.md, hooks, review agents) reference, enforce, or review these rules — they never duplicate or reinterpret them. Amendments require owner approval.

## 1. Security authority

Never allow model or agent output to establish, modify, or override Themis security truth. Never bypass deterministic authorization because an agent or model requests an action.

## 2. External egress

No data egress to a non-local system without explicit, recorded operator authorization. Local-first is the default.

## 3. Secrets

Never check in API keys or tokens. Registry entries reference keys only by environment-variable name (`api_key_env`). Credentials are not ordinary model context and never enter prompts, logs, or durable storage.

## 4. Destructive operations

No destructive or irreversible operation without explicit controls and appropriate approval. When in doubt, it is destructive.

## 5. Git operations

- Never add a Claude signature or co-author/session trailer to any commit.
- Always honor `.gitignore`; never force-add ignored paths.
- Never check in binaries that bloat the repo (built artifacts, model files, archives).
- History rewrites, shared-remote pushes, and branch-topology changes require owner approval.

## 6. Architecture changes

Stop and ask before any change that alters: architectural ownership, security authority, trust boundaries, deployment topology, fundamental model architecture — or when the architecture is unclear. (The closed stop+ask enumeration lives in SKILL.md §25.)

## 7. Model boundary

Nothing above the Model Interface may know which provider is configured. The model is never told it is the security system. No provider-specific behavior outside the model adapter/registry.

## 8. Security-control bypass

Never disable, weaken, or work around a security control — hook, policy, verification step, or review — to make progress. Security-control failure stops work; it does not fall back to model judgment.
