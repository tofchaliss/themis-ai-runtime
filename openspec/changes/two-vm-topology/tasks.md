# Tasks — two-VM topology (V-M1..V-M3)

Grill closed 2026-09-27 (D-V-1..5). Order: N-M0 (explicit scopes) is a
prerequisite wherever the narrowed operator-key blast radius is relied
on; V-M1 and V-M2 may land before it, V-M3 after.

## V-M1 — Harness: contract v2 and the pinned door (Class 3: trust boundary)
- [ ] `policies/themis/contract.json` schema v2: `https` required for
      both base URLs; `themis_tls: {spki_sha256: [≤2], ca_sha256}` (one
      block, or one per URL); `contracts.Parse` refuses v1 shapes under
      v2 and v2 fields under v1
- [ ] Read-door client: TLS config built from the contract only —
      custom verifier checking SAN = URL host, leaf SPKI ∈ pins, chain
      to the pinned CA; system CA store never consulted; any mismatch →
      `ErrIdentity` → L4 `themis-identity-refused` audit carrying the
      observed SPKI; `ErrUnavailable` → `seam-unavailable` only for
      reachability
- [ ] Anchor: `themis_contract` pins the v2 file (no new anchor field);
      Open re-verifies as today
- [ ] Preflight: fetch the live leaf/chain, print observed SPKI/CA and
      whether they reproduce the contract; warn on v1 contracts
- [ ] Tests: httptest TLS servers for the positive path and each
      negative (wrong SPKI, wrong CA, wrong SAN, plain http, two pins
      with rollover); the audit class asserted by name; constitution
      pins asserted unchanged (a new L4 denial class is vocabulary —
      check whether it moves `constitution.orchestration`; if it does,
      say so and mint accordingly)

## V-M2 — Themis: the intake consumes the same pins (Class 3)
- [ ] `cmd/themis-intake` builds its client from
      `harness/integrations/themis/contracts` (already on the depguard
      allow-list as a read-only record contract) with the same verifier;
      `--governance` flag replaced by the contract path; identity
      mismatch is a named refusal, exit 1, nothing raised
- [ ] Deployment assets (not code): a TLS terminator in front of
      Governance and Registry on the Themis VM with a private CA; source
      allowlist for the read key's routes; env-referenced secrets only
- [ ] Docs: INSTALLATION section for the terminator; EDR-TOPOLOGY-01

## V-M3 — Host: the proof set and rsys@8 (Class 4: host)
- [ ] `docs/operations/two-vm-host-sequence.md`: Option B on the
      enterprise estate; the D-V-5 proof set under a test anchor; then
      the candidate `rsys8.proposed.json` (contract v2 pin; the @1/@2
      bundles dropped — rsys@8 hygiene); act; the demo use case once
      more across two VMs
- [ ] Addendum H; matrix row "transport and identity" 🟢 with evidence;
      reviews; archive
