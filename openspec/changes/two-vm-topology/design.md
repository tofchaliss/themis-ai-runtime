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

## D-V-4 — Credential custody, blast radius, rotation (LOCKED 2026-09-27, owner: option 1)

> Credentials are held per holder, presented per act, bounded by scope
> and TTL, and identified in records by key id only. The harness read
> key is deployment-held (mode 0600, the harness service's environment,
> outside `$REPO`/`$DEPLOY`), usable only for reads and accepted by the
> terminator only from the harness VM. The operator key is human-held,
> supplied only for an individual `themis-intake` act, short-lived and
> revocable. The decider key is never present on the harness VM and is
> used only for Governance decision acts. Credential values never enter
> records, evidence, bundles or shell history. Key rotation needs no
> new anchor (credentials are not part of the anchored server
> identity). Revocation is immediate at Themis.

| Credential | Holder | Authority | If stolen |
|---|---|---|---|
| read key | harness service | read | estate read access until revoked |
| operator key | human | product-scoped Governance acts after N-M0 | can commission / propose / withdraw as permitted, never decide |
| decider key | human | Governance decision | never crosses onto the harness VM |

Acknowledged, not hidden: until N-M0 lands, a non-read key can perform
broader Governance writes than the post-hardening model; N-M0 is a
prerequisite for relying on the narrowed operator-key blast radius, and
that narrower scope is NOT enforced today. Custody upgrade path, with
mTLS: a secrets agent injecting keys into both CLIs — a later change.
Rejected: one long-lived shared key.

## D-V-5 — The proof set and the identity refusal class (LOCKED 2026-09-27, owner: option 1)

> A contract-v2 two-VM deployment must prove private-network
> reachability, TLS server identity, authenticated read, authenticated
> proposal, the negative identity/transport cases, credential-rotation
> behaviour and certificate-renewal behaviour before the new anchor is
> minted. The read-door client and `themis-intake` consume the same
> `themis_tls` pins and ignore the system CA store. A
> certificate/SAN/SPKI/CA mismatch is classified
> `themis-identity-refused`, recorded in the L4 audit with the observed
> SPKI, never retried as `seam-unavailable`; `seam-unavailable` is
> reserved for genuine reachability failures.

Proof set under a test anchor, then `rsys@8`: private-network
reachability (route, the terminator refusing the public interface);
positive HTTPS read; wrong SPKI → `themis-identity-refused`; `http` URL
→ refused at contract/Open (v2 requires `https`); read from outside the
allowlist → terminator refusal; HTTPS proposal from the intake with the
operator key → succeeds; read-key rotation → no anchor change; same-SPKI
certificate renewal → no anchor change. Evidence: handshake facts
(observed SPKI, CA) in preflight output and Addendum H; the bundle as
before. Classification boundary: cannot reach Themis →
`seam-unavailable`; reached an endpoint that is not the identity the
anchor names → `themis-identity-refused`. An impersonating endpoint must
never look like an outage.

## Grill state (2026-09-27)
Q-V-1..5 LOCKED. Implementation: V-M1 (harness: contract v2, pinned
TLS in the door, the refusal class, preflight), V-M2 (Themis: intake
consumes the same pins; the terminator and allowlist on the Themis VM
are deployment assets), V-M3 (host: the proof set, `rsys@8`, Addendum
H). Sequenced after N-M0 where the operator-key blast radius is relied
on. See `tasks.md`.

