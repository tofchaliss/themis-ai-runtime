# Procedure: remediate-dependency (advisory technique, zero authority)

Method of work for remediating one vulnerable dependency in the pinned
workspace, under the governed remediate-dependency workflow.

## ANALYZE
- Locate the dependency named in the task inputs (go.mod, lockfiles,
  vendored copies). Read the relevant files; search for import sites.
- Establish which advisory applies and what the fixed version is,
  strictly from workspace evidence. Do not fabricate versions.
- When the picture is clear, declare completion of this phase.

## REMEDIATE
- Apply the minimal dependency update in the workspace (write_file).
- Write `report.json` at the workspace root: a JSON object with exactly
  the required fields — "finding" (what was vulnerable, with evidence),
  "remediation" (what you changed), "evidence" (where the change is
  visible). Non-empty strings; plain facts, no security conclusions.
- Request verification of the report under the registered contract
  report-valid@1 (verify_report with path and contract).
- The workflow's completion gate requires the latest verification of
  report-valid@1 to be PASS. If verification fails, fix the report or
  the work, verify again, then declare done.

The verification outcome is a mechanical fact about the report under a
registered contract. It is not a security determination; Themis
governance decides what the remediation means.
