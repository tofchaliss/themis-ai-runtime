# L4 Amendment: the `delegate` capability (L8 M2, 2026-09-23)

Additive changes to the archived L4 layer, authorized by L8 design
§5.2 (D-L8-4, D-L8-15, C-L8-5, C-L8-7, C-L8-11, C-L8-13, C-L8-15 G,
C-L8-16; owner LOCK 2026-09-22). Pre-amendment registries load
identically; no existing decision path changes.

1. **Target class `delegation-template`** (`TargetDelegationTemplate`).
   `Authorize`, stage A: the target must be an exact `name@version`
   (`template-ref-shape`) and a member of the grant entry's
   `template_scope` by exact string equality
   (`template-outside-grant-scope`) — the single authoritative
   substitution gate; no prefixes, no ranges, no `latest`; an empty
   scope grants nothing (the `themis_scope` rule, reapplied).
2. **Registry-v5** (`policies/tools/registry-v5.json`): v4 plus
   `delegate` — `trust: external-untrusted`, not verifier-eligible,
   not control, not mutating, `timeout_sec: 30`; params `template`
   (string, required, target), `evidence` (string, optional:
   comma-separated `<seq>:<objectID>` references into the parent's
   own record), `brief` (string, optional). Any other argument —
   `scope`, `instructions`, `model`, … — is `unknown-field` at L4
   (D-L8-4: selection authority, never composition authority).
3. **Evidence-reference shape** (`ParseEvidenceRefs`): shape only at
   L4 (`evidence-ref-shape`); ≤ 256 references; a bare object id is
   not a reference (C-L8-5). Existence, task reachability, hash
   integrity, and class derivation are re-established from the record
   in the seam (stage B).
4. **Closed error classes** (C-L8-11): `delegation-refused:<reason>`
   with the reason from the closed set {template-unresolvable,
   template-withdrawn, template-hash-mismatch, registry-unreadable,
   evidence-unreachable, evidence-not-prior, evidence-duplicate,
   evidence-registry-drift, evidence-slot-ambiguous, resolve-failed,
   compose-refused, brief-over-bound, seam-unavailable};
   `delegation-provider-error`, `delegation-output-over-bound`,
   `delegation-model-identity-mismatch`. `DelegationRefused(reason)`
   collapses an unknown reason to `seam-unavailable`; `KnownErrorClass`
   is the membership predicate.
5. **Executor `delegate` = instantiation** (C-L8-12 Am. 1, C-L8-13):
   `execDelegate(inst)` calls the injected `DelegationInstantiator`
   — the seam's compose half, bound per task by L7 through
   `NewExecutorTableWith(reg, seam, inst)` — and returns its
   instantiation CAPTURE (identities only) as the audited evidence,
   framed under the registered trust; a `*ErrDelegationRefusal` becomes
   the closed stage-B class in `l4-audit{error}`; any other error is
   `seam-unavailable`; a nil instantiator is
   `delegation-refused:seam-unavailable` (assembly refuses that
   configuration; the executor still fails closed). No model is
   executed under L4; the interface takes no model, conversation, or
   context (pinned by `TestDelegateSeamIsComposeOnly`).
6. **`template_scope` in `grantAuthorityDigest`** (landed with
   skill-admission SA-M4) — the differing-digest test now owed here
   is in `TestGrantAuthorityDigestCoversEveryAuthorityField`
   (gained / swapped / emptied). `template_scope` is a fixed member of
   the Skill's grant template with no override surface
   (`TestInstantiationSurfaceHasNoTemplateScope`).

Evidence: `tools/delegate_test.go` (positive first, then each denial
naming its predicate), `TestRegistryV5DeclaresDelegate` (dispatch
completeness with `delegate`; older registries know no delegate).
`themis-run` still loads registry-v4; the switch to v5 is the `rsys@4`
act (M6).
