# Harness

The AI execution control plane owned by Themis. It runs model-assisted tasks under deterministic controls; it never owns security truth.

## Language

### Instructions (Layer 1)

**Instruction**:
A rule delivered to the model — never a fact. One Markdown file (or inline task payload entry) with an id, scope, and category.
_Avoid_: prompt, system prompt, directive

**Scope**:
An instruction's rank in the fixed seven-level precedence order. A lower scope may specialize a higher one, never contradict it.
_Avoid_: priority, level

**Protected instruction**:
An instruction no lower scope can shadow or override; attempts are rejected and recorded.

**Effective Instruction Set (EIS)**:
The deterministic, hashed, reconstructable output of resolving all recognized sources for one task. The only instruction artifact the model ever receives, delivered via Context Delivery.
_Avoid_: final prompt, merged instructions

**Resolution epoch**:
The lifetime of one EIS — from its resolution at an orchestration-owned boundary to the next boundary or termination. Instruction change is succession of epochs, never mutation within one.
_Avoid_: refresh, reload, update (of instructions)

**Source**:
A registered root or payload explicitly recognized as feeding instructions. Content the agent merely encounters is data, never a source.

**Conflict**:
A recorded resolution event — a shadowed or rejected instruction with its reasons. Trace data, not a log line.

### Boundary vocabulary

**Advisory output**:
Model-generated content of any form. Inert data; it proposes and never transitions state.
_Avoid_: result, decision, finding (unqualified)

**Decision surface**:
The view Themis governance composes for a human exercising a governed decision: authoritative state, evidence, model analysis, and decision controls. Not addressable from task instructions or model output.
_Avoid_: model response, report

**Capability**:
An operation available to the model through the Tool Interface, defined by a registered contract. Capabilities exist and authorize independently of instructions and model output.
_Avoid_: permission, tool (when meaning the grant rather than the mechanism)

**Capability boundary**:
A deterministic control with no instruction-addressable surface — nothing any instruction says, at any scope, can reach its configuration. Day-0 prohibitions are capability boundaries, never precedence contestants.
_Avoid_: highest-priority instruction

**Task payload**:
The envelope a task arrives in. Its structure — not interpretation — splits instructions (to Layer 1) from asserted facts (to Context Delivery as untrusted context).
