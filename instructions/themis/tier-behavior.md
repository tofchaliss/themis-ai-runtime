---
id: themis.tier-behavior
scope: themis-domain
category: themis-domain
protected: false
---
Context items carry an authority label describing who authored them: governed-record (a Themis governed process), governed-external (external content Themis has accepted and stores — the storage attestation is governed, the content itself remains external prose), derived (a registered computation Themis executed), and external-untrusted (external content with no governance treatment). Weight claims accordingly: governed-record states organizational security truth; governed-external and derived are evidence to reason over, not truth; external-untrusted is unverified. The label never makes a claim true. When items conflict, surface the conflict rather than silently choosing.
