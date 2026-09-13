# Design: G1 — Deployment Authority Anchoring

Grill OPEN 2026-09-13 → **CLOSED 2026-09-13: D-G1-1 LOCKED with
owner amendment D-G1-1A** (anchor admission). Source gap:
integration-audit synthesis G1.

## D-G1-1 — The Deployment Anchor (LOCKED 2026-09-13; owner
amendment D-G1-1A folded)

One governed artifact — the Deployment Anchor — pins the
deployment's authoritative artifact set by content hash, consumed
fail-closed at task assembly. Not a new authority class: the fourth
application of the proven Governance registration pattern.

**Q-G1-1 (owner LOCK):** Governance owns deployment identity and
admission; L7 enforces; the submitter owns neither.

**Q-G1-2 (owner LOCK):** the authoritative set is a CLOSED
enumeration, each entry an exact content hash: L4 tool registry;
grant ceiling(s); exec ceiling; workflow ceiling; skill catalog;
L10 contract registry; L11 criteria and regression-set registries;
instruction roots (safety/system/themis) as content hashes; the
MODEL ALLOWLIST (models.json becomes anchor-governed — an external
endpoint can enter a deployment only by Governance act; closes half
the R3 coupling); the door table. Closed schema; nothing defaulted;
unknown fields refused. (Owner: the model allowlist matters because
otherwise the deployment is governed everywhere except where the
model itself is selected; the door table belongs because it
establishes which governed mechanisms constitute the deployment.)

**Q-G1-3 (owner AMEND → D-G1-1A below):** anchor bytes live at
policies/deployment/anchor artifacts; the ACTIVE-registration
relationship is explicit in the Governance plane
(policies/deployment/anchors.json, registry format), entering by
*.proposed.* → owner act → ACTIVE.

**D-G1-1A — Deployment Anchor admission (owner amendment,
constitutional):** *a Deployment Anchor is executable as deployment
authority only when its exact bytes and content hash resolve to a
Governance-ACTIVE Deployment Anchor. A caller-supplied anchor path
or hash may IDENTIFY the requested deployment; it can never
establish its governed status.* Sequence at Open: resolve exact
anchor (bytes unavailable → refusal; hash mismatch → refusal) →
establish Governance admission against the anchors registry
(unregistered → refusal; withdrawn → refusal) → ACTIVE anchor →
immutable Open. Then SubmitTask: bundle artifacts hash-matched
against the ADMITTED anchor → assembly. Two distinct steps, never
collapsed: anchor admission answers "is this a Governance-approved
deployment definition?"; bundle validation answers "does this
submission conform to that definition?" Without this, an operator
manufactures an anchor pinning any artifact set, supplies its own
hash, and every bundle check passes — the bundle is unforged but
the anchor is forged: exactly the C-1 trust-error class (a caller
may identify evidence or configuration; the governed owner must
establish its authority).

**Q-G1-4 (owner LOCK, admission-before-consumption):** Open =
resolve/admit anchor + verify expected operator-supplied hash
(consumption pin) + freeze; SubmitTask = verify bundle ⊆ admitted
anchor. One anchor per Open; no live mutation.

**Q-G1-5 (owner LOCK):** the submitter chooses a task WITHIN the
deployment (workflow among anchored workflows, payload, skill refs
via the anchored catalog); the submitter does not define the
deployment. Ceilings, registries, instruction roots, model list are
structurally out of the submitter's hands.

**Q-G1-6 (owner LOCK):** the anchor's content hash is the
authoritative deployment identity; the version field is
human/reference identity only.

**Q-G1-7 (owner LOCK after amendment):** the assembly record
carries the anchor identity/hash, the admitted-anchor
evidence/reference, and the artifact hashes checked — so replay and
reconstruction establish "executed under this ADMITTED Deployment
Anchor", not merely "under a consistent bundle".

**Q-G1-8 (owner LOCK):** supersession = new anchor version via the
proposed→act discipline; prior anchors retained (append-only,
permanently interpretable); adoption by restart only.

**Q-G1-9 (owner LOCK):** bundle/anchor disagreement → typed
refusal, recorded. No closest match, no partial bundle, no
unpinned fallback; an absent/unreadable/unadmitted anchor fails
Open itself.

**Q-G1-10 (owner LOCK with the admission clarification):** no new
authority store — one more governed registry-format artifact under
the existing Governance act discipline, with the same walls
(read-only machinery, no write API, two-way identity where
applicable, AST-audited).

**Residual (owner-accepted, deliberately outside G1):** submitter
authentication. G1 establishes WHAT deployment is authorized, not
WHO authenticated the caller requesting it. Caller origin is
recorded-not-authenticated (the D-L6-4 structural-trust position),
pointing at the L8/deployment boundary. Owner tightening: *caller
origin may be recorded, but origin/authentication status must never
be interpreted as evidence of Governance authority* — the residual
itself must not become a trust channel.

**Constitutional sentence:** *a caller may identify the deployment
it requests; only Governance admission makes an anchor authority,
and only the admitted anchor makes a bundle governed.*

## Implementation note

Implementation follows the audit's defect pass and its own Class-3
review; the anchor loader reuses the proven registry patterns
(readGoverned, duplicate-key wall, two-way identity, append-only
check, consumption pin). Enforcement lands in
orchestration.Open/SubmitTask. No production wiring of SubmitTask
before the anchor enforcement exists.
