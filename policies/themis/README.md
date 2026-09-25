# policies/themis — the Themis v0 store

Governance's system of record for the demo-scale Themis
(`openspec/changes/themis-v0`, D-T-7 and D-T-9). Two anchor-pinned,
append-only, READ-ONLY registries that the harness may read through
the L4 seam, and one Governance-output directory it may never touch.

## Layout

- `findings.json` — Findings: `id` (`FIND-…`), `product` (a registered
  Product id), `advisory`, `component`, `affected_version`,
  `fixed_version`, `severity` (`low|medium|high|critical`), `summary`,
  `state` (`active|withdrawn`), `steward`. Immutable once registered; a
  change is a new Finding; withdrawal is forward-only.
- `products.json` — Products: `id` (`PROD-…`), `name`, `version`,
  `state`, `steward`. **A Product is a minimal name-and-version
  referential record a Finding may identify. `get_product` is
  referential context only; a Product has no independent security
  disposition in v0.**
- `positions/<finding-id>/<n>.json` — Enterprise Positions, written
  ONLY by `themis-decide` (D-T-7), never by the harness, never edited,
  never deleted. Not part of the model-visible pin.

## The pin

The deployment anchor's `themis_store` is the SHA-256 of the exact
bytes of `findings.json` and `products.json` concatenated in that
order (`themis-status` prints it). A deployment cannot silently gain or
lose Findings; a change is a new anchor version.

## Read door

`get_finding` / `get_product` (L4, `trust: governed-record`, target
class `themis-id`, scoped by the grant's `themis_scope` prefixes)
return the exact record bytes. The store mints no authority class: the
L4 registration does. Withdrawn records are not servable (they remain
history); unknown ids are `seam-unavailable`.

## Registration checklist

1. Finding `product` names a registered Product.
2. Severity from the closed vocabulary; `summary` is a fact statement
   about the Finding, not a disposition (dispositions are Positions).
3. Steward named.
4. Never edit in place: append, withdraw, re-register.
