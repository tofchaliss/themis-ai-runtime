# L2 Amendment: record-object sources, several sources per slot, tool:* kind (L8 M4, 2026-09-23)

Authorized by: L8 design §5.2 (new source kind, C-L8-7) and C-L8-6
(one source per evidence reference, all in the template's evidence
slot), C-L8-17 F; owner LOCK 2026-09-22. Additive; the parent path
(one inline source per slot) is unchanged.

1. `KindRecordObject` — a lazy read of ONE content-addressed L6
   object through an `ObjectReader` seam: metadata-only construction
   (object id, item kind/version, DERIVED class and sensitivity from
   the witnessing event); `collect()` fetches and verifies the bytes
   against the address, so L2 pulls evidence under its own caps. A
   read failure is a hard error (a named reference that cannot be read
   refuses the gather), and a corruption verdict propagates. An
   `ObjectID == ""` source with no items is a DECLARED ABSENCE, so the
   seam names every non-withheld slot and the contract's requirement
   decides. Permitted classes: all four — the registration is the
   event, and the seam that read it is deterministic machinery.
2. A slot may receive several DISTINCT sources (the same source twice
   is still "assigned twice"); items are merged and ordered by
   `(Kind, Hash)` across sources, so presentation stays the contract's
   deterministic function of the set (Q-L3-4, C-L8-6 §7).
3. Slot kind `tool:*` matches any `tool:<name>` item (as `file:*`
   matches `file:` items). `KindMatches` is exported for the seam's
   slot routing (C-L8-14 C).

Evidence: `context/record_object_test.go` (two sources in one slot in
either order → identical items; duplicate source refused; class and
kind outside the slot refused; missing object hard error; address
mismatch refused). Mutation: removing the address verification → the
mismatch case reads through and fails (probed 2026-09-23).
