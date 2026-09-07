# Proposal: Layer 6 — Durable State

**Change ID:** layer-06-durable-state · **Status:** DRAFT for grilling · 2026-09-07
**Owner gate:** ships only after the grill closes Q-L6-1..n (design.md) and the owner accepts.

## Why

Five layers of accumulated IOUs point here, all explicitly recorded:

- **L4:** AuditEvent timing + executor identity "join at the L6 trace-sink era"; audit events exist per call but live in memory.
- **L2:** refusal-status trace sink deferred; delivery records (PayloadHash, typed absence) are in-memory structs.
- **L1:** conflicts/exemptions are trace data "not a log line" — with nowhere durable to land.
- **L5:** the environment Trace (transitions, ops, seal reason, artifact address, teardown verification) is durable-by-intent, in-memory in fact; ArtifactStore retention/GC is an explicit L6 IOU; v1 retain-all.
- **Cross-cutting:** three-state discipline demands evidence that survives the process; today a crash forgets everything but the artifact store.

The harness produces records; nothing yet *keeps* them. L6 is the durable record plane.

## What

A durable-state layer at `src/harness/state`:

- **Task record:** one durable envelope per task — identity, governed-artifact hashes (EIS, contract, policy, registry, grant, ceiling, spec), status, terminal outcome — reusing the shipped RunRecord provenance pattern (manifest-first writes, options retained end-to-end).
- **Trace sink:** the append-only destination for the typed events every layer already emits (L1 conflicts, L2 delivery/refusal records, L3 selection traces, L4 audit events, L5 transitions/ops) — persisted at defined boundaries, hash-bound to the task record.
- **Timing enters here:** the deliberate L4 deferral lands — wall timestamps join durable events at the sink, never inside the deterministic decision paths.
- **ArtifactStore lifecycle:** retention posture beyond v1 retain-all; the store's addresses referenced from task records.
- **Not a security database:** Themis remains the system of record; the harness trace is the *operational* record of what the harness did — it feeds Themis governance, never competes with it.

## What this change does NOT do

- No task resumability/checkpoint-restart unless the grill decides otherwise (crash = task failed, re-run is orchestration).
- No L7 loop, no scheduling state.
- No second evaluation subsystem, no metrics/telemetry pipeline.
- No secret material in any durable record, ever (credential contamination handling precedes persistence — locked at Q-L5-5).

## Success criteria

Every model-visible byte and every authorization decision of a completed task is reconstructable from durable state alone (task record + trace sink + artifact store + governed artifacts at recorded hashes); crash mid-task leaves a typed, attributable partial record — never a silent absence; three-state verdicts per milestone; live proof: a full L1→L5 task run whose complete trace is read back from disk and verified against the in-memory events.
