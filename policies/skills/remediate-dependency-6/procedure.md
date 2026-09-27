# Procedure: remediate-dependency v6 (advisory technique, zero authority)

Method of work for remediating one vulnerable dependency in the pinned
workspace, under the governed remediate-dependency workflow. Version 4
reads the Finding from the live Themis authority by its UUID and holds
no product read: the Finding names everything the work needs.

The workflow has phases and each phase begins a fresh conversation:
nothing you read or said in an earlier phase is shown to you again. The
Context delivered with each phase names the current phase, the phases
already completed, and the capabilities available in this phase. Work
the current phase from that Context and the task inputs; a tool not
named there is unavailable here, whatever this procedure describes.

### ANALYZE
- This phase reads and corroborates; it changes nothing.
- First call get_finding with the finding id named in the task inputs
  (a UUID) to read the governed Finding record: its release, faultline,
  CVE, investigation stage, and the matched components (package URL,
  name, version, ecosystem). Treat it as the authoritative statement of
  WHAT is affected; it does not decide what to do. The target version is
  a task input supplied by the operator, never inferred from the record.
- Corroborate: the dependency named in the task inputs must appear
  among the Finding's components and the advisory must be the Finding's
  CVE. If they do not correspond, say so and call the declare_done
  capability without changing anything.
- Locate the dependency in the workspace (go.mod, lockfiles, vendored
  copies). Read the relevant files; search for import sites.
- When the evidence you have read needs a second, isolated reading,
  you may delegate one triage pass over it: call delegate with the
  registered template and the record-ref values shown under the tool
  results you want examined. The result is advisory and untrusted;
  weigh it against the workspace evidence yourself.
- When the picture is clear, call the declare_done capability. That call
  is the only way a phase completes; stating that the phase is complete
  does not complete it.

### REMEDIATE
- This phase begins fresh: ANALYZE is complete and its reading is not
  shown again; get_finding is not available here. Work from the declared
  task inputs — the dependency, the advisory, and the operator-supplied
  target version — and from the workspace you can read.
- Apply the minimal dependency update in the workspace: read go.mod, then
  write it back (write_file) with the dependency at the target version,
  changing nothing else.
- Write `report.json` at the workspace root: a JSON object with exactly
  the required fields — "finding" (what was vulnerable, with evidence),
  "remediation" (what you changed), "evidence" (where the change is
  visible). Non-empty strings; plain facts, no security conclusions.
- Request verification of the report under the registered contract
  report-valid@2 (verify_report with path and contract).
- The workflow's completion gate requires the latest verification of
  report-valid@2 to be PASS. If verification fails, fix the report or
  the work, verify again, then call the declare_done capability. Do not
  change the report
  after its last verification: only the bytes that were verified can
  be accepted downstream.

The verification outcome is a mechanical fact about the report under a
registered contract. It is not a security determination; Themis
governance decides what the remediation means.
