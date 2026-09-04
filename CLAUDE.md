# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is this?

This repository hosts **Themis** — a security application — and its **AI Harness** runtime, developed under the Themis Claude Code Development Standard.

- Themis is the security application and system of record.
- The Harness is a Themis-owned execution capability (11 layers; see `ARCHITECTURE.md`).
- DeepSeek is a replaceable model provider behind the Harness model interface.
- Human authority is final; all AI output is advisory.
- No AI output ever becomes authoritative security truth.

## Where the rules live (single authoritative home each)

| Question | Read |
|---|---|
| What is the system architecture? | `ARCHITECTURE.md` (authoritative) |
| How must Claude Code engineer it? | `.claude/skills/themis-harness-engineering/SKILL.md` — **load before any harness implementation work** |
| What are this repo's concrete engineering conventions (build, test, lint, structure)? | `AGENTS.md` |
| What is absolutely prohibited? | `.claude/policy/DAY-0.md` |

Hooks in `.claude/hooks/` (registered in `.claude/settings.json`) mechanically enforce the applicable safety rules; the reviewer agents in `.claude/agents/` assess architecture, security, and tests after changes. Standards process and accepted stage texts: `docs/claude-code-development-standard.md`.
