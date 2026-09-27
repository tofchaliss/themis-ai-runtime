# Design record — two-VM topology (PROPOSED until each Q-V is locked)

## D-V-1 — The authoritative network boundary (LOCKED 2026-09-27, owner: option 1)

> Harness ↔ Themis production communication requires enterprise-private
> network reachability plus TLS with server authentication. The
> deployment verifies private-network routing/firewall constraints; TLS
> protects the bearer key in transit. The harness trusts Themis through
> the certificate identity pinned by the deployment anchor. Themis
> authenticates the harness using the bearer key and additionally
> relies on the permitted network boundary; neither side treats an IP
> address alone as identity. mTLS is an explicitly supported future
> upgrade path when enterprise workload identity exists, not mandatory
> for the current topology.

Network establishes reachability; the TLS certificate establishes
which Themis endpoint; the bearer key authenticates the request; the
anchor binds the expected Themis identity into the governed deployment.
Future shape: certificate identity + workload identity/mTLS +
authorization — no mTLS fields or PKI machinery now. Rejected: mTLS
mandatory (makes production wait on an enterprise PKI decision that is
not the harness's); plain HTTP on a private network (bearer secret in
clear).

## Q-V-2 — What the contract pins about the transport (OPEN)
## Q-V-3 — Where the intake runs; how its write reaches Governance (OPEN)
## Q-V-4 — Stolen credentials across the boundary; rotation (OPEN)
## Q-V-5 — Host procedure and evidence for a two-VM run (OPEN)
