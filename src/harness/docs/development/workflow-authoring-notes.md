# Workflow authoring notes

## Verification gates and canonicalizer strength (integration-audit D13)

A gate `{"contract": "<k>@<v>", "outcome": "PASS"}` opens on exactly
what the bound contract's canonicalizer establishes — no more. The
v1 `report-valid@1` canonicalizer (`canonReport`) establishes
WELL-FORMEDNESS: a JSON object with non-empty `finding`,
`remediation`, and `evidence` strings. Its PASS is trivially
model-satisfiable by construction.

Authoring rule: do not gate a completion edge (`@complete`,
`declare_done`) on a well-formedness contract as if it were
substantive verification. Where a walk's completion should mean
"the work was verified", bind a contract whose verifier
deterministically checks the WORK (e.g. a future `run_go_*`-class
verifier under the recorded L5 amendment), or treat the
well-formedness gate as a protocol step and keep the substantive
judgment at the human door.

The gate ladder is exactly as strong as the weakest registered
contract it cites. Registration review of a workflow should read
each cited contract's canonicalizer before approving a gated edge.
