# L4 Amendment: tool-result framing (integration-audit D6, 2026-09-13)

Authorized by: D-L4-7 (already locked — "results framed into the
conversation as an L2-disciplined item"), which the original
implementation left unrealized; audit finding S1-S4/F-3. Additive
implementation correction, no constitutional change.

tools.Handle now frames every AUTHORIZED result with a
content-derived fence and a kind/authority/hash header
(frameToolResult, tools/execute.go) — untrusted fetched bytes are no
longer visually indistinguishable from harness protocol text. Error
results remain unframed (harness-authored protocol text). Evidence
bytes, hashes, audit events, and the verifier seam consume the RAW
evidence unchanged; only the model-visible conversation surface
gained the frame. Proofs: unframeResult assertions in
tools_test/mutate_test; live walk re-run PASS.
