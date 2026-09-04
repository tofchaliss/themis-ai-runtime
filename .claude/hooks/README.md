# Themis Claude Code Hooks

Hooks are deterministic enforcement mechanisms and complement the development skill.

Initial P0 objectives:
- Block or require review for destructive commands.
- Protect secrets and sensitive configuration.
- Protect architectural boundary files where appropriate.
- Require verification after state-changing implementation work.

Do not encode business security truth in hooks. Hooks enforce development safety; Themis remains the runtime authority.
