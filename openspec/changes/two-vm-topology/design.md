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

## D-V-2 — What the anchor pins for Themis server identity (LOCKED 2026-09-27, owner: option 1)

> Themis server identity is anchored by DNS/SAN identity, SPKI pin and
> issuing-CA pin. The contract URL host remains the expected DNS
> identity and the certificate SAN must match it. `contract.json` schema
> v2 carries `themis_tls.spki_sha256` (at most two entries, for staged
> key rollover) plus `ca_sha256` for the required issuing CA. The VM
> system CA store is not authoritative and is ignored for Themis
> authentication. Certificate renewal with the same key needs no new
> anchor; server-key rotation, a different CA, or a different hostname
> is a new Governance anchor. Where Governance and Registry share a TLS
> terminator one `themis_tls` block covers both, otherwise each URL has
> its own pin set. The intake uses the same contract-pinned verification.

Rejected: DNS/SAN with system chain validation (trust in an
un-anchored CA store); SPKI only (a same-key certificate for another
name would pass); full certificate hash (every renewal a Governance
act).

## D-V-3 — Where the intake runs; how its write reaches Governance (LOCKED 2026-09-27, owner: option 1)

> `themis-intake` runs on the harness VM and reads the local record
> plane directly. Its only network action is the proposal POST to
> Governance, which crosses the D-V-1 boundary using the D-V-2 pinned
> Themis identity, with the operator's `product:<id>` key for that one
> human-initiated write. No record-plane copy exists on the Themis VM;
> the record plane is not exposed as a network seam. Intake provenance
> (Themis commit, harness pin) stays in the evidence.

Only the proposal crosses the boundary. The intake is an
evidence-producing reader/proposer, not a general network client.
Rejected: copying the task's record to the Themis VM (a second copy
with no rule for which is authoritative); exposing the record plane
read-only over the network (a new seam and contract for no gain).
Consequence carried into Q-V-4: the operator's key is present on the
harness VM at the moment of use.

## Q-V-4 — Stolen credentials across the boundary; rotation (OPEN)
## Q-V-5 — Host procedure and evidence for a two-VM run (OPEN)
