# L7 Amendment: themis root wiring + grant staging (integration-audit D7/D10, 2026-09-13)

Additive implementation corrections under locked decisions; no
constitutional change.

- D7: Config gains ThemisRoot (optional until the deployment anchor
  pins it); Open and resolveTaskEIS resolve ScopeThemisDomain with
  the same discipline as the other roots — the L2 authority
  vocabulary now ships with its interpretive half (the L2 archive's
  recorded dependency). Proof: TestThemisRootWiring resolves the
  repo's instructions/themis root.
- D10: instantiateGrant stages the effective grant under the state
  root instead of shared host tmp — live authority bytes stay
  inside the disjoint-root discipline.
