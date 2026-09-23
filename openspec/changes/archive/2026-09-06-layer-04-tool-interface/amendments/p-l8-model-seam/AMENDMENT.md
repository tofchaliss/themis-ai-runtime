# Model-interface seam amendment: provider-response ceiling and reported identity (2026-09-23)

Authorized by: L8 design §5.3 P-L8-1 and P-L8-2 (owner closure, LOCKED
2026-09-22). DEC-05 stands: nothing provider-specific crosses the seam.
Not an L8 artifact — parent runtime hardening recorded here because
the Model Interface's decision record lives in the L4 archive.

## P-L8-1 — hard byte ceiling on the provider response body

- `runtime/model.DefaultMaxResponseBytes = 256 KiB`, the compiled hard
  bound; `readBounded` reads `LimitReader(body, max+1)` and refuses
  typed (`ErrResponseOverCeiling`) when the body is longer. The bytes
  are discarded: a truncated body is a failed turn, never a turn.
- Both adapters (`OllamaChat`, `OpenAIChat`) carry `MaxResponseBytes`,
  set by their constructors to the default; a zero or over-default
  value refuses rather than reading unbounded.
- Governed narrowing: `models.json` entry `max_response_bytes` (0 =
  default; > default or negative refused at `model.Resolve`). The
  registry bytes are anchor-pinned (`model_registry`), so the
  effective ceiling is a deployment input; on the `absent`-registry
  path the compiled bound applies.
- `DefaultMaxResponseBytes ≤ context.MaxItemBytes` is pinned by
  `context.TestProviderCeilingWithinItemCap` (C-L8-7).
- L7 consequence: an over-ceiling body surfaces as a provider error
  turn (`model-turn{fact: provider-error}`, no output object) —
  unchanged path.

## P-L8-2 — provider-reported identity

- `Identity.Reported` carries the model identifier the provider
  reported in its payload (`model` field, both adapters), parsed inside
  the adapter. `Reported ≠ WireModel` is observable upstream (L8 stage
  C `model-identity-mismatch`); parent-loop handling is an L7 residual.

Evidence: `runtime/model/ceiling_test.go` (`TestResponseCeiling` —
at-limit ordinary turn, one-over typed termination, zero and
over-default refused, both adapters; `TestRegistryNarrowsResponseCeiling`).
Mutation: removing the limiter and the over-limit check → the
over-limit cases read through as content and the test fails (probed
2026-09-23).
