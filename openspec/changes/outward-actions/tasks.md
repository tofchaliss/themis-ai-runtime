# Tasks — outward actions (N-M0..N-M5)

Grill closed 2026-09-27 (D-N-1..7); extended 2026-10-01 with the
remediation cycle (D-N-8..D-N-12, owner feedback). All code lands in the
Themis repository (`~/code/themis`, greenfield tree, EDR + `phase3-outward-
actions` change, `make check`, commit/push only on explicit ask). The
harness repository changes nothing through N-M4: no network tool, no new
seam. The valuation-complete notification of D-N-8 does not change that —
it is a Themis-owned seam the harness subscribes to, never a harness call
outward, and the read door stays the only harness-facing data seam.
**One exception, designed 2026-10-07 and not implemented:** the poller of
N-M2j (section N-M5 below) reads the Governance cursor endpoint D-N-13
locks. It subscribes and does nothing else — no Jira, no CI, no mail, no
outward credential, no Governance act.

## N-M0 — Explicit scope authorization (Themis platform/auth; Class 3: security) — **UNCHANGED by D-N-8..D-N-12**
- [ ] `AuthorizeWrite` retired: Governance write endpoints require `admin`
      or `product:<id>` explicitly; a `delivery:callback` key is refused
      on every Governance write (tests: each write route × each scope)
- [ ] `product:<id>` confined to that product's Findings on Governance
      writes (the row-14 gap, second half) — route resolves Finding →
      release → product before authorizing
- [ ] `authadmin` mints `delivery:callback`; scope vocabulary closed

N-M0 stands exactly as recorded. Implemented Themis-side 2026-09-30
(`phase3-outward-actions` Group 1, `EDR-DELIVERY-01` D1–D7). The
remediation cycle adds no scope, no relaxation and no new Governance
write path: `delivery:callback` stays refused on every Governance
write, `product:<id>` stays confined to its own product, and the
`ci_rebuild` callback enters through the Communication boundary only.

## N-M1 — Delivery intents and workers (Themis Communication; Class 3)
- [ ] Store: `delivery_intents` (event identity, lineage, snapshot,
      content-addressed payload, destination name, type, state:
      pending / delivered / dead-letter, attempts, last_error, result
      evidence); append-only history of attempts
- [ ] Intent creation on `finding_opened` (`jira_issue`),
      `proposal_accepted` (`ci_build` when the evidence is a harness
      execution; `email` always), CI callback (`email`), dead-letter
      (`email` to the operations audience) — inside the existing
      Governance-stream reader, writing intents only (D-N-2: the stream
      never waits on the outside)
- [ ] Per-system workers with backoff; failure isolation from the bus
      reader; dead-letter after N attempts; human retry/cancel API;
      `vm-verify.sh` and status render pending / dead-letter counts
- [ ] Jira worker (REST, token from env by destination name; issue key
      recorded as result); mail worker (SMTP from env; audiences →
      addresses from configuration; plain text; idempotent by intent id)
- [ ] Payload materializers derive from the snapshot only; tests assert
      no model output / workspace content / key material can enter

## N-M2 — CI build delivery and callback (Themis Communication; Class 3)

Numbering note: this is the `ci_build` milestone (an accepted artifact,
the `proposal_accepted` path). The **N-M2a..N-M2j** steps of D-N-13 are
a different N-M2 — the plan's rebuild cycle, whose kind is `ci_rebuild`
and which carries no artifact. The two must not be conflated (D-N-10).
- [ ] `ci_build` snapshot carries the recorded artifact members + hashes
      (read through Themis's own intake of the harness record — the
      evidence the proposal already cites), Finding/release lineage,
      destination pipeline name
- [ ] Trigger worker (pipeline API from env); callback endpoint accepting
      `{intent_id, build_id, image_digest, git_ref}` under
      `delivery:callback` or the HMAC variant; result stored as
      governed-external evidence on the intent; never a Finding change
- [ ] Reference CI job (branch from the release repo, apply the artifact
      bytes exactly, build, push, register the new release version in
      Registry, upload the image SBOM citing the intent id, call back with
      `{intent_id, build_id, image_digest, git_ref, release_version}`;
      never merge)
- [ ] `remediated_in` on the original Finding (D-N-7): written by Themis
      only from the new SBOM's evaluation (fault absent / still present),
      as governed-external evidence with release, digest, intent,
      evaluated_at; shown in the Finding view and release posture
      ("affected, fixed in R.x.y+1"); `finding_resolved` stays a human act;
      stance vocabulary note (`mitigated` ≠ `fixed`) in the API docs

## N-M3 — Host wiring and the extended demo (Class 4: host)
- [ ] Destinations configured by name on the Communication node (Jira
      project, SMTP relay, CI pipeline); keys in env only
- [ ] Demo: new SBOM → Findings → Jira defects; commission → walk →
      Position → CI branch + image → mail; new image SBOM → Themis marks
      the part fixed; dead-letter drill (Jira down) visible and recovered
- [ ] Addendum H; matrix row for outward actions; reviews; archive

## N-M4 — Remediation cycle (documentation only in this change; Class 4: architecture)

Recorded 2026-10-01 from owner feedback. Nothing here is code: the
deliverable of this change is the decision record. Implementation is
Themis-side (`phase3-outward-actions` Group 5 and the milestones that
follow it).

- [x] 4.1 `design.md` D-N-8..D-N-12: valuation-complete gate + the
      Themis-owned pub/sub notification; Jira one per Release with CVE
      ids for Critical/High only; `ci_rebuild` approved with its
      callback members; default max-attempts 2; ownership recap and
      the invariants the loop must not erode
- [x] 4.2 `proposal.md`: the owner's six-step workflow and what it
      settles; doc-only scope; N-M0 unchanged; what stays open
- [x] 4.3 Themis side mirrored: `EDR-DELIVERY-01` Revision 2
      (RC-1..RC-8) and `openspec/changes/phase3-outward-actions`
      (design acceptance block, proposal, tasks Group 5)
- [x] 4.4 Gates (no code touched — verifying cleanliness):
      `gofmt -l src/harness` (empty), `go vet ./src/harness/...`,
      `go test ./src/harness/...`, `go build ./src/harness/...`,
      `.claude/hooks/doc-lint-guard --all`
- [x] 4.5 **Settled 2026-10-07 (D-N-13).** The seam is a **polled
      Governance cursor read API**:
      `GET /api/v1/governance/events/release-evaluated?after=<sequence>&limit=<n>`,
      `X-API-Key` **read** scope, at-least-once, **dedupe by event
      id**, cursor = the sequence number, `limit` default 100 / max
      500, events stored in a new Governance table; no SSE, no webhook.
      Event names: `knowledge.release_correlation_completed.v1` →
      `governance.release_evaluated.v1`; owning context **Governance**.
      Recorded Themis-side as `EDR-DELIVERY-01` Revision 5 (N-M2); the
      API change itself is N-M2d, Themis-side
- [x] 4.6 **Settled 2026-10-07.**
      `THEMIS_COMMUNICATION_REBUILD_MAX_ATTEMPTS`, default **2**,
      Communication-side, **no per-Release override**
- [x] 4.7 **Settled 2026-10-07.** Strictly the immediately-previous
      SBOM of the same Release **by upload order**; no configured
      window. The targeted Critical+High set is fixed at cycle start
      and does not grow mid-loop
- [x] 4.8 `design.md` **D-N-13** + the renumbered build steps
      **N-M2a..N-M2j** (API/schema steps marked: N-M2d, N-M2h);
      `proposal.md` records the owner's six decisions. Themis side
      mirrored: `EDR-DELIVERY-01` Revision 5 (M2-1..M2-9) and
      `openspec/changes/phase3-outward-actions` (design acceptance
      block, proposal, tasks Group 6). Documentation only
- [x] 4.9 Gates (no code touched): `gofmt -l src/harness` (empty),
      `go vet ./src/harness/...`, `go test ./src/harness/...`,
      `go build ./src/harness/...`

## N-M5 — Harness poller (N-M2j; the ONLY harness-side code in the cycle)

Designed 2026-10-07 (D-N-13), not implemented. A new outbound READ seam
on the harness is an architecture decision — **owner approval before
implementation**, and the class is set then. Everything else in the
cycle stays Themis-side.

- [ ] 5.1 Poll
      `GET /api/v1/governance/events/release-evaluated?after=<sequence>&limit=<n>`
      with `X-API-Key` at read scope; persist a local high-water mark
      (the last `seq` consumed) so progress is the harness's own state
- [ ] 5.2 At-least-once handling: **dedupe by event id**, and filter
      `cause=new_sbom` (a `rediscovery` event is ignored — it starts and
      advances nothing)
- [ ] 5.3 Test:
      `TestHarnessPoller_PollingWithCursor_AtLeastOnce_DedupeAndFilterNewSBOM`
      against an `httptest` Governance stub — paging with the cursor, a
      redelivered page proving dedupe is harmless, a `rediscovery`
      event ignored
- [ ] 5.4 **Subscribes only**: no network tool, no Jira / CI / mail
      client, no outward credential, no Governance act (D-N-1, D-N-12).
      The read door stays the only harness-facing seam for Themis DATA;
      this endpoint carries a signal, not data the harness reasons over
