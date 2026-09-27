# Tasks — outward actions (N-M0..N-M3)

Grill closed 2026-09-27 (D-N-1..7). All code lands in the Themis
repository (`~/code/themis`, greenfield tree, EDR + `phase3-outward-
actions` change, `make check`, commit/push only on explicit ask). The
harness repository changes nothing: no network tool, no new seam.

## N-M0 — Explicit scope authorization (Themis platform/auth; Class 3: security)
- [ ] `AuthorizeWrite` retired: Governance write endpoints require `admin`
      or `product:<id>` explicitly; a `delivery:callback` key is refused
      on every Governance write (tests: each write route × each scope)
- [ ] `product:<id>` confined to that product's Findings on Governance
      writes (the row-14 gap, second half) — route resolves Finding →
      release → product before authorizing
- [ ] `authadmin` mints `delivery:callback`; scope vocabulary closed

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
