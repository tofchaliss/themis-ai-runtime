# Themis AI Runtime — Governed Execution Chain (as built)

Status: reflects the FROZEN architecture as of 2026-09-13 —
L1–L11 shipped and archived, G1 (Deployment Authority Anchoring) and
G2 (Established-Fact Boundary) closed and implemented, R1 (the
ungoverned legacy HTTP surface) removed. This document DESCRIBES the
locked state; it does not define it. Authoritative sources:
`ARCHITECTURE.md`, `.claude/policy/DAY-0.md`, the per-layer archives
under `openspec/changes/archive/`, and
`openspec/changes/g{1,2}-*/design.md`.

The older `harness-in-themis-sketch.md` predates L9–L11 and G1/G2 and
is retained as history.

---

## 1. The chain, end to end

```
                              G O V E R N A N C E   (humans)
                                        │
         ┌──────────────────────────────┼───────────────────────────────┐
         │ registers (proposed → act → ACTIVE, append-only, hash-pinned)│
         ▼                              ▼                               ▼
  Deployment Anchors            Skill catalog · L10 contracts    L11 criteria · sets
  policies/deployment/          policies/skills · verification/   policies/ratchet/
         │
         │  D-G1-1A: a caller may IDENTIFY a deployment;
         │            only the anchors registry ADMITS it
         ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│  DEPLOYMENT OPEN  (orchestration.Open)                                        │
│                                                                               │
│   anchor path + operator hash ──► AdmitAnchor ──► ACTIVE anchor               │
│                                        │                                      │
│                                        ├─ instruction roots (safety/system/   │
│                                        │   themis) verified BEFORE resolve     │
│                                        ├─ instruction policy                   │
│                                        ├─ model registry (or "absent")         │
│                                        ├─ L6 + L7 constitution hashes          │
│                                        ├─ deployment execution ceiling:        │
│                                        │   exact bytes supplied here,          │
│                                        │   hash-bound, must also LOAD          │
│                                        └─ anchors registry append-only         │
│                                            (prior state persisted per root)    │
│                                        ▼                                      │
│                              FROZEN deployment authority                      │
│                        (one anchor per Open; adoption by restart only)        │
└───────────────────────────────────┬───────────────────────────────────────────┘
                                    │
                                    ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│  TASK ASSEMBLY  (orchestration.SubmitTask) — the ⊆-checkpoint                 │
│                                                                               │
│   submitter envelope ─► must BE the anchored artifacts:                       │
│        tool registry · workflow BUNDLE (workflow + its workflow ceiling       │
│        + its context contract, indivisible) · deployment ceiling ·            │
│        model ∈ allowlist · skill composition resolved FROM the anchored       │
│        catalog (never a submitter-supplied constituent hash)                  │
│                                                                               │
│   the submitter chooses a TASK WITHIN the deployment — never the deployment   │
└───────────────────────────────────┬───────────────────────────────────────────┘
                                    │
                                    ▼
        L1 Instructions ──► L2 Context Delivery ──► L3 Context Management
                                    │
                                    ▼
        L4 Tool Interface ──► L5 Execution Environment    (deterministic
          registry ∩ grant ∩ quota      confined, sealed   authorization;
          availability before args      no ambient env     model proposes,
                                    │                       code authorizes)
                                    ▼
        L6 Durable State — the sole record plane
          record-before-effect · content-addressed objects · typed events
                                    │
                                    ▼
        L7 Orchestration (δ) ──► L9 Skills ──► L10 Verification
          consumes only recorded      sealed        registered contracts;
          typed events; gates on      composition   PASS/FAIL/… never leave
          declared contract tokens                  L10's vocabulary
                                    │
                                    ▼
        L11 Ratchet — comparative evidence only
          facts must be WITNESSED (G2) · Δ addressed to doors, never to control
                                    │
                                    ▼
                           G O V E R N A N C E   (humans)
                     evidence arrives; only a door makes a change real
```

### Model position

```
            Themis / Harness supply everything the model sees
                                   │
                                   ▼
                      ┌────────────────────────┐
                      │   Model (adapter)      │  probabilistic reasoning only
                      └────────────────────────┘
                                   │
        model output ─► validation ─► policy ─► authorization ─► execution
                     ─► verification ─► governed result

        The model never learns it is the security system, never holds
        authority, and its bytes are advisory at every crossing.
```

---

## 2. The two cross-layer boundaries (G1, G2)

```
G1 — what governs an execution?           G2 — what is a fact?

Governance                                 the minting mechanism
    │ admits                                   │ witnesses
    ▼                                          ▼
Deployment Anchor                          committed typed event
    │ pins exact artifact set                  │ names the object
    ▼                                          ▼
frozen authority at Open                   established fact of kind F
    │ bundle ⊆ anchor                          │ consumable by a criterion
    ▼                                          ▼
governed execution                         comparative evidence (Δ)
```

**G1 (D-G1-1 + D-G1-1A):** a caller-supplied anchor path/hash
IDENTIFIES a requested deployment; only resolution against the
Governance-ACTIVE anchors registry establishes governed status.
Bundle validation is a SEPARATE second step against the admitted
anchor — the two are never collapsed.

**G2 (D-G2-1):** storage proves bytes; EVENTS prove establishment.
An L6 object is a fact of kind F iff a committed event of F's
minting class names it.

| Fact kind | Established by | Witness |
|---|---|---|
| l10_evaluation_record | the L10 evaluator | `l10-verification` event naming the record in its body |
| l6_execution_record | an L4/L5 executor under a registered capability | `l4-audit` event referencing the object (tool pinnable) |
| benchmark_validated_score / gate_verdict | the benchmark plane | verdict + digest + canonical location (gate discipline) |
| model-authored bytes | **nothing** | not a fact kind — `model-turn` witnesses authorship, which is advisory |
| L11 packages | **nothing** | no witnessing event exists by construction — terminal output |

---

## 3. Deployment identity

```
host-specific configuration               (mirror_root, limits)
            │
            ▼
   exact execution-ceiling bytes
            │  sha256
            ▼
   Deployment Anchor pin  ──────────►  Governance admission  ──►  ACTIVE
            │
            ▼
   deployment identity = the anchor's content hash
```

The execution ceiling is **deployment-scoped and
deployment-supplied**: exact bytes handed to Open, hash-bound by the
anchor, never a repo artifact with placeholders and never chosen by a
submitter. Two hosts running the same governed workflows under
different ceilings are two different deployment identities.

Repository state: **no ACTIVE anchor ships here.** `local-dev@1` is
WITHDRAWN (it pinned a spec template where a ceiling belongs); a
runnable anchor is deployment-instance-specific and is created where
that deployment lives. See `policies/deployment/README.md`.

---

## 4. What is NOT in the chain

- **No ungoverned model invocation.** The legacy HTTP surface
  (`/v1/extract`, `/v1/recommend-position`) was decommissioned
  (audit R1). What remains of `internal/service` is the **model
  router**: verdict-gated selection among admitted models, the
  D-L11-8 Class-2 consumer. It selects *within* an admitted set; it
  never extends one.
- **No L12, no second evaluation subsystem, no feedback store, no
  candidate registry, no champion pointer, no scheduler inside L11.**
- **No unanchored production path.** An anchorless `Open` refuses
  unless the caller explicitly declares the test-harness
  `Unanchored` role; those records carry a sentinel so they are never
  mistaken for anchored ones.

---

## 5. Reading order for a newcomer

1. `ARCHITECTURE.md` — ownership, authority, prohibitions (binding).
2. This diagram — how the pieces connect as built.
3. `docs/harness-layer-status.md` — what shipped, when, with residuals.
4. `openspec/changes/archive/<layer>/design.md` — why each layer is
   shaped the way it is (the locked decisions).
5. `openspec/changes/g1-deployment-authority/design.md` and
   `openspec/changes/g2-established-fact-boundary/design.md` — the
   two cross-layer boundaries.
