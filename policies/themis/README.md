# Themis interface contract

`contract.json` is the governed statement of which Themis authority a
deployment reads and against which interface (D-I-3, amending D-T-9).
The deployment anchor pins its SHA-256 as `themis_contract`; the
harness's read door (`src/harness/integrations/themis/client`) is built
over this exact file and reports the same hash at Open.

| Field | Meaning |
|---|---|
| `governance_base_url` | Governance read API (`GET /api/v1/findings/{id}`) — loopback on the shared host |
| `registry_base_url` | Registry read API (`GET /api/v1/products/{id}`) |
| `governance_spec_sha256`, `registry_spec_sha256` | SHA-256 of the two OpenAPI specifications at `themis_commit` |
| `themis_commit` | The Themis source commit those specifications come from; the host verifies the deployed estate is at it before minting the anchor (D-I-8) |

The contract pins the INTERFACE. It does not pin Finding or Product
bytes (they are read live and captured as execution-time governed
records) and does not attest the running Themis binary. The read-scope
credential is never in this file: it reaches the client from the
environment (`THEMIS_API_KEY_READ`) and appears in no record.

Regenerating the spec hashes (from the Themis checkout at the commit):

```bash
shasum -a 256 api/governance.openapi.yaml api/registry.openapi.yaml
```
