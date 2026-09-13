# L9 Amendment: loader duplicate-key wall (integration-audit D8, 2026-09-13)

Executes the RECORDED follow-up (L10 archive tasks.md FOLLOW-UP;
carried through the L11 archive): LoadCatalog and LoadManifest now
refuse JSON with duplicate keys at any depth (skills/strict.go, the
proven checker shape) — reviewed text and decoded semantics can no
longer disagree. Residual executed, not widened.
