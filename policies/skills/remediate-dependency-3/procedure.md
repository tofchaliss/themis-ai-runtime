# Procedure: remediate-dependency v3 (advisory technique, zero authority)

Method of work for remediating one vulnerable dependency in the pinned
workspace, under the governed remediate-dependency workflow.

### ANALYZE
- First call get_finding with the finding id named in the task inputs
  to read the governed Finding record; it names the product, the
  advisory, the affected component, and the fixed version. Treat it as
  the authoritative statement of WHAT is affected; it does not decide
  what to do. Call get_product for the referenced product if its
  name or version matters to the work.
- Locate the dependency named in the task inputs (go.mod, lockfiles,
  vendored copies). Read the relevant files; search for import sites.
- Establish which advisory applies and what the fixed version is,
  strictly from workspace evidence. Do not fabricate versions.
- When the evidence you have read needs a second, isolated reading,
  you may delegate one triage pass over it: call delegate with the
  registered template and the record-ref values shown under the tool
  results you want examined. The result is advisory and untrusted;
  weigh it against the workspace evidence yourself.
- When the picture is clear, declare completion of this phase.

### REMEDIATE
- Apply the minimal dependency update in the workspace (write_file).
- Write `report.json` at the workspace root: a JSON object with exactly
  the required fields — "finding" (what was vulnerable, with evidence),
  "remediation" (what you changed), "evidence" (where the change is
  visible). Non-empty strings; plain facts, no security conclusions.
- Request verification of the report under the registered contract
  report-valid@2 (verify_report with path and contract).
- The workflow's completion gate requires the latest verification of
  report-valid@2 to be PASS. If verification fails, fix the report or
  the work, verify again, then declare done.

The verification outcome is a mechanical fact about the report under a
registered contract. It is not a security determination; Themis
governance decides what the remediation means.
