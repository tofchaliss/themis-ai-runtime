# L1 Amendment: the delegation instruction source (L8 M4, 2026-09-23)

Authorized by: L8 design C-L8-4 §3 (the only source a template adds is
its own registered instruction file, through the existing skill-source
activation); owner LOCK 2026-09-22. No new source kind.

`instructions.ActivateDelegationSource(bytes, pinnedSHA256)` is
`ActivateSkillSource` under the fixed id `skill.delegation` (scope
`skill`): the same byte-integrity rule, the same untrusted tier, the
same unshadowable constitution. A second fixed skill-scope member so a
carried parent procedure (`skill.procedure`) and the template's rules
coexist without one shadowing the other. Fixed, not derived from the
template name — the consumer interprets nothing about templates.

Evidence: `TestRegisteredTemplateLoads` (the pinned instruction
activates), `TestDelegatedEISCarryFilter` (the template instruction is
always present; the parent procedure only where the filter names
`skill`; the safety root regardless).
