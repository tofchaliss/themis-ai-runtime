# G1 Amendment: the `delegation_template_registry` pin (L8 M6, 2026-09-23)

Authorized by: L8 design D-L8-5/6 (Q-L8-7) and §5.2 (owner LOCK
2026-09-22). Additive to the anchor schema; the model-registry posture
reapplied.

- `Anchor.DelegationTemplateRegistry` (`delegation_template_registry`):
  a sha256 of the registry bytes, or the explicit declaration
  `"absent"` — a declaration, never a default (a pre-amendment anchor
  without the field refuses to load).
- `verifyAnchoredInstructionPlane` (Open and per task): `absent` ⇔ no
  `Config.DelegationRegistryPath`; a pin ⇔ the configured file hashes
  to it ("a template enters a deployment only by Governance act").
  `themis-run` builds the delegation seam over the same path and
  checks the registry root's disjointness from every task-writable
  root; `themis-status` prints the pin; `themis-preflight` prints it
  beside the template verification.
- Consequences carried by `rsys@4` (PROPOSED,
  `policies/deployment/rsys4.proposed.json`): `constitution.state`
  re-pinned (M3), `tool_registry` → registry-v5, `contract_registry`
  → contracts.json with `report-valid@2` (a contract pins the L4
  registry hash it is eligible under, so v5 needs its own contract),
  `skill_catalog` with `remediate-dependency@2` (ANALYZE exposes
  `delegate`; grant template scopes `dependency-triage@1`), `skills`
  listing `remediate-dependency@1..2`, and the new pin. Every one of
  these is a Governance act the owner performs by committing the
  proposed files; nothing here ACTIVATES `rsys@4`.

Evidence: `subagents/delegation/seam/anchored_test.go` —
`TestAnchoredDelegationPositivePath` (anchored @2 walk, the model's
evidence reference formed from the record-ref furniture, witness,
reconstruction CONFIRMED) and `TestAnchoredDelegationRefusals`
(pin without path, absent with path, bytes ≠ pin, withdrawn template
in scope); `deployment.TestAnchor…` loader rule.
