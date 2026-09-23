# L9 Amendment: skill admission identity (2026-09-22)

Authorized by: D-SA-1..10 (LOCKED, owner, 2026-09-22 —
`openspec/changes/l9-l7-skill-admission-identity/design.md`), D-L10-17
(the amendment protocol). Raised by the 2026-09-15 finding (issue #1):
`verifyAnchoredSkill` compared the envelope's composition seal against
the catalog's manifest hash — two identity domains, equal for no
input. This record is evidence of change to an archived layer; the
original archive stands; additive and correcting only.

## Amendment definition

1. **D-L9-13 wording (D-SA-1):** "L7 does not consult the Skill
   catalog" → "L7 does not consult the Skill catalog for semantics;
   under an anchored deployment it reads the anchor-pinned catalog's
   registered identities for the correspondence check, comparing hashes
   only." Records the supersession G1 already made.
2. **D-L9-11d matrix extended (D-SA-5):** the claimed Skill identity is
   a first-class, schema-validated envelope field `skill = name@version`
   — the sole admission selector. `origin` is attribution only. Any
   `skill`/`skill_*` origin key requires the `skill` field and, where
   `origin.skill` is present, must equal it; `skill` requires the
   commitment. L9 `Instantiate` emits both.
3. **Skill-scope instruction provenance (D-SA-3):** under an anchored
   deployment `skill_procedure_path`/`_sha256` are composition locators
   admissible only through a Skill-attributed envelope whose composition
   corresponds; an unattributed procedure is refused at assembly. Hash
   integrity ≠ Governance provenance. Unanchored behaviour preserved
   (D-SA-7).
4. **Correspondence (D-SA-2):** the seven Skill-fixed members of the
   commitment must EQUAL the manifest's pins, per member, each with its
   own refusal. The seal is never compared against Governance (D-SA-6:
   exactly two consumers — `checkSeal`, the record).
5. **Instantiation (D-SA-4):** `tools.Instantiates(effective, template,
   taskID)` — tool set, `mutating`, `themis_scope`, `template_scope`
   set-equal; quotas/total narrow; workspace/task_id governed bindings;
   `execution.SpecInstantiates` likewise (deadline narrows, other limits
   equal). L7 resolves the reference template from the MANIFEST PIN
   (`Manifest.ResolvePin`), never the commitment claim. L9 mirrors the
   relation after `checkGrantShape`.
6. **Claim 2 evidence (A-SA-11):** `Catalog.Raw` / `Manifest.Raw`; the
   consumed bytes are stored per task as `skill_catalog` /
   `skill_manifest` objects; `skill` is a governed record key.
7. **Withdrawal (D-SA-8):** clarified — forward-only state on the
   identity; running tasks and history untouched; `Resolve`'s existing
   refusal is the gate.

## Affected invariants — none weakened

D-L9-0 (L9 never executes at runtime), D-L9-1 (atomic composition),
D-L9-4 (no interpreters), D-L9-7 (closed instantiation surface),
D-L9-10 (immutable bindings), D-L9-11a/b/c (seal + Claim 1) all stand.
Claim 2 ("the submitted composition is the one registered as X") moves
from "post-hoc only" to "established at anchored assembly by D-SA-2",
which D-L9-11a explicitly left to catalog evidence.

## Evidence

`orchestration/skill_admission_test.go` (positive twin + negatives),
`tools/instantiate_test.go`, `execution` TestSpecInstantiates,
`deployment` TestSkillAllowlistRules; live anchored walk PASS
(qwen2.5:7b, COMPLETED/VERIFIED). Commit dc8034c.
