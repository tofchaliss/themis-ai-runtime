# Two-VM topology — the harness ↔ Themis network trust boundary (D-I-1 amendment; grill opened 2026-09-27)

## Why now

D-I-1 (2026-09-25) locked "same host" and I-M5 ran that way (Option A,
reaffirmed 2026-09-27). The enterprise estate runs the harness on one VM
and Themis on another. Before anything is built for production, the
transport and trust boundary between them must be decided: it changes
what the contract pins, how the read door and the intake reach
Governance, and what a stolen credential can do.

## Owner's roadmap (2026-09-27), recorded

1. this grill (D-I-1 amendment) → 2. D-I-9 subject-bound read scope →
3. model governance → 4. D-P-3 evidence continuity, only when a real
skill needs it → 5. rsys@8 hygiene (drop the @1/@2 bundles; one rule
one home in procedures). Answered without a grill: **automatic
commissioning is not wanted; a human remains the commissioner for every
governed run** (D-C-1, D-C-6 stand). Jira may initiate a
delivery/integration workflow; it cannot commission governed
remediation execution.

## Facts from code (2026-09-27)

- What crosses the boundary today: (a) the read door — `GET
  /api/v1/findings/{id}` and `GET /api/v1/products/{id}` with
  `X-API-Key: <read key>`; (b) `themis-intake`, which must run where the
  record plane is (the harness VM) and POSTs the proposal to Governance
  with the operator key; (c) later, CI callbacks into Communication
  (D-N-6), Themis-side only.
- `policies/themis/contract.json` accepts `http` or `https` base URLs;
  the anchor pins the contract's bytes, so the URL is a governed fact.
- The read-door client and the intake use Go's default `http.Client`:
  system CA trust, no certificate pinning, no client certificate.
- Every Themis node serves plain HTTP (`http.ListenAndServe`); there is
  no TLS listener and no mTLS support in the nodes. TLS today means a
  terminator in front of them or a node change.
- Authentication is a bearer secret (`X-API-Key`) on every call.

## Scope of this grill
Q-V-1 the authoritative network boundary (private network + TLS, or
mTLS with workload identity) · Q-V-2 what the contract pins about the
transport (URL scheme, CA / certificate identity) · Q-V-3 where the
intake runs and how its write reaches Governance · Q-V-4 what a stolen
read key or operator key can do across the boundary, and the rotation
rule · Q-V-5 the host procedure and evidence for a two-VM run.
