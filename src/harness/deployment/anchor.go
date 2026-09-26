// Package deployment implements the G1 Deployment Anchor (D-G1-1,
// D-G1-1A — openspec/changes/g1-deployment-authority/design.md): the
// Governance-owned, content-addressed declaration of the exact
// artifact set under which a deployment may execute.
//
// The two-step boundary, never collapsed:
//  1. ANCHOR ADMISSION — a caller-supplied anchor path/hash
//     IDENTIFIES the requested deployment; only resolution against
//     the Governance-active anchors registry establishes that it is
//     a governed deployment definition (D-G1-1A: the C-1 lesson — a
//     caller may identify configuration; the governed owner must
//     establish its authority).
//  2. BUNDLE VALIDATION — a submission conforms to that admitted
//     definition (enforced at the L7 seam).
//
// This package READS; there is no write API (the standing wall).
package deployment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/decisions"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	ErrAnchor    = errors.New("invalid deployment anchor")
	ErrAdmission = errors.New("deployment anchor admission refused")
)

var (
	shaSyntax  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	nameSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// skillRefSyntax: an exact skill reference, name@version.
	skillRefSyntax = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*@[1-9][0-9]*$`)
)

// Anchor is the closed deployment-definition schema (Q-G1-2). Every
// pin is an exact content hash; nothing is defaulted; unknown fields
// refuse. Pins L7 loads are enforced at Open/SubmitTask; pins for
// other planes (skill catalog, contract/criteria/set registries,
// model allowlist, door table) are the consumption pins those
// planes' consumers verify — admission enforced at consumption, the
// ratified pattern.
type Anchor struct {
	Version    int    `json:"version"`
	Name       string `json:"name"`
	Deployment int    `json:"deployment_version"`

	// Enforced by L7 at Open (dir-tree hashes; policy file hash):
	InstructionSafetyRoot string `json:"instruction_root_safety"`
	InstructionSystemRoot string `json:"instruction_root_system"`
	InstructionThemisRoot string `json:"instruction_root_themis"`
	InstructionPolicy     string `json:"instruction_policy"`

	// Constitution pins the compiled control vocabularies whose change
	// changes what an anchored deployment can DO (owner test: "can
	// changing this artifact change the behavior or authority of an
	// anchored deployment?"). They are code identities, not files, so
	// a rebuilt binary with different constitutions cannot open under
	// an anchor that pinned the old ones.
	Constitution ConstitutionPin `json:"constitution"`

	// ExecutionCeiling pins the DEPLOYMENT's execution ceiling by the
	// hash of its exact bytes (owner decision, 2026-09-13). The
	// ceiling carries deployment-specific configuration (mirror_root
	// and friends), so it is supplied at deployment Open — never
	// committed as a repo artifact with placeholders, and never
	// selected by a submitter. Two hosts may run the same governed
	// workflows under different ceilings; each is a different
	// deployment identity because each pins different bytes.
	//
	// Deployment-scoped, not workflow-scoped: the ceiling describes
	// where and under what limits THIS deployment executes, which is
	// a property of the deployment, not of a workflow.
	ExecutionCeiling string `json:"execution_ceiling"`

	// Enforced by L7 at SubmitTask (bundle artifact bytes):
	ToolRegistry string `json:"tool_registry"`
	// Workflows is the anchored workflow set, each entry a COMPLETE
	// bundle: the workflow and the ceilings/contract that govern it.
	// Per-workflow arity (close-review M-6): scalar ceilings made
	// multi-workflow deployments unexpressible, and a submitter could
	// pair any anchored workflow with any anchored ceiling.
	Workflows []WorkflowBundle `json:"workflows"`
	Models    []string         `json:"models"` // the model allowlist (names)
	// ModelRegistry pins the model-registry bytes (models.json): the
	// allowlist governs NAMES, and this pin governs what those names
	// RESOLVE TO — runtime, endpoint, credential env (close-review
	// HIGH-3; Q-G1-2's endpoint clause). "absent" is the explicit
	// declaration that the deployment ships no model registry, i.e.
	// local-only resolution; it is a declaration, never a default.
	ModelRegistry string `json:"model_registry"`

	// SkillCatalog is AUTHORITATIVE for skill composition resolution
	// under an anchored deployment: the submitter selects an anchored
	// skill identity, never one of its constituent hashes (owner
	// disposition, finding 1).
	SkillCatalog string `json:"skill_catalog"`
	// Skills is the deployment's Skill allowlist (D-SA-9): the exact
	// name@version set THIS deployment may run, checked at anchored
	// SubmitTask before catalog resolution — the finer-grained gate
	// beside the bundle-level Workflows gate, mirroring Models. Catalog
	// membership establishes identity; this pin establishes deployment
	// admissibility; neither implies the other. Absent or empty admits
	// no skill-attributed task: a skill enters a deployment only by
	// Governance act, never by existing in the global catalog.
	Skills []string `json:"skills,omitempty"`
	// Consumption pins for planes consumed outside L7 (each verified
	// by that plane's own consumer; see the residual record):
	ContractRegistry      string `json:"contract_registry"`
	CriteriaRegistry      string `json:"criteria_registry"`
	RegressionSetRegistry string `json:"regression_set_registry"`
	// DelegationTemplateRegistry pins the L8 delegation-template
	// registry bytes (D-L8-5/6, Q-L8-7): a deployment cannot silently
	// acquire or lose available templates. "absent" is the explicit
	// declaration that the deployment ships no delegation registry —
	// no phase may then expose `delegate`; a declaration, never a
	// default.
	DelegationTemplateRegistry string `json:"delegation_template_registry"`
	// ThemisContract pins the Themis interface contract (D-I-3, amending
	// D-T-9): the SHA-256 of policies/themis/contract.json — the
	// authorized Governance/Registry endpoints and their OpenAPI spec
	// identities at a named Themis commit. The explicit declaration
	// "absent" means this deployment reads no Themis authority. The
	// contract pins the interface, never live Finding/Product data.
	ThemisContract string `json:"themis_contract"`

	SHA256 string `json:"-"` // of the exact anchor bytes — the deployment identity
	Raw    []byte `json:"-"`
}

// WorkflowBundle is one anchored workflow with the artifacts that
// govern it — an indivisible unit, never a menu of interchangeable
// parts.
type WorkflowBundle struct {
	Workflow        string `json:"workflow"`
	WorkflowCeiling string `json:"workflow_ceiling"`
	ContextContract string `json:"context_contract"`
}

// ConstitutionPin names the compiled control vocabularies.
type ConstitutionPin struct {
	State         string `json:"state"`
	Orchestration string `json:"orchestration"`
}

const maxAnchorBytes = 1 << 20

// maxWorkflowBundles bounds the enumeration: an anchor is a CLOSED
// declaration, so its workflow set is finite and reviewable.
const maxWorkflowBundles = 64

// ParseAnchor validates anchor bytes fail-closed.
func ParseAnchor(raw []byte, origin string) (*Anchor, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrAnchor, origin, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var a Anchor
	if err := dec.Decode(&a); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrAnchor, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrAnchor, origin)
	}
	if a.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrAnchor, origin)
	}
	if !nameSyntax.MatchString(a.Name) {
		return nil, fmt.Errorf("%w: %s: bad deployment name %q", ErrAnchor, origin, a.Name)
	}
	if a.Deployment < 1 {
		return nil, fmt.Errorf("%w: %s: deployment_version must be a positive integer", ErrAnchor, origin)
	}
	for field, v := range map[string]string{
		"instruction_root_safety":    a.InstructionSafetyRoot,
		"instruction_root_system":    a.InstructionSystemRoot,
		"instruction_root_themis":    a.InstructionThemisRoot,
		"instruction_policy":         a.InstructionPolicy,
		"tool_registry":              a.ToolRegistry,
		"execution_ceiling":          a.ExecutionCeiling,
		"constitution.state":         a.Constitution.State,
		"constitution.orchestration": a.Constitution.Orchestration,
		"skill_catalog":              a.SkillCatalog,
		"contract_registry":          a.ContractRegistry,
		"criteria_registry":          a.CriteriaRegistry,
		"regression_set_registry":    a.RegressionSetRegistry,
	} {
		if !shaSyntax.MatchString(v) {
			return nil, fmt.Errorf("%w: %s: %s must be a sha256 hex digest — nothing is defaulted", ErrAnchor, origin, field)
		}
	}
	if a.ModelRegistry != "absent" && !shaSyntax.MatchString(a.ModelRegistry) {
		return nil, fmt.Errorf("%w: %s: model_registry must be a sha256 hex digest or the explicit declaration \"absent\"", ErrAnchor, origin)
	}
	if a.DelegationTemplateRegistry != "absent" && !shaSyntax.MatchString(a.DelegationTemplateRegistry) {
		return nil, fmt.Errorf("%w: %s: delegation_template_registry must be a sha256 hex digest or the explicit declaration \"absent\" — nothing is defaulted", ErrAnchor, origin)
	}
	if a.ThemisContract != "absent" && !shaSyntax.MatchString(a.ThemisContract) {
		return nil, fmt.Errorf("%w: %s: themis_contract must be a sha256 hex digest or the explicit declaration \"absent\" — nothing is defaulted", ErrAnchor, origin)
	}
	if len(a.Workflows) == 0 {
		return nil, fmt.Errorf("%w: %s: the anchored workflow set must not be empty", ErrAnchor, origin)
	}
	if len(a.Workflows) > maxWorkflowBundles {
		return nil, fmt.Errorf("%w: %s: the anchored workflow set exceeds %d bundles — an anchor is a closed, reviewable declaration", ErrAnchor, origin, maxWorkflowBundles)
	}
	seenW := map[string]bool{}
	for _, w := range a.Workflows {
		for field, v := range map[string]string{
			"workflow": w.Workflow, "workflow_ceiling": w.WorkflowCeiling,
			"context_contract": w.ContextContract,
		} {
			if !shaSyntax.MatchString(v) {
				return nil, fmt.Errorf("%w: %s: workflow bundle %s must be a sha256 hex digest", ErrAnchor, origin, field)
			}
		}
		if seenW[w.Workflow] {
			return nil, fmt.Errorf("%w: %s: duplicate workflow pin — one workflow, one bundle", ErrAnchor, origin)
		}
		seenW[w.Workflow] = true
	}
	if len(a.Models) == 0 {
		return nil, fmt.Errorf("%w: %s: the model allowlist must not be empty — an unlisted model is not a deployment default", ErrAnchor, origin)
	}
	seenM := map[string]bool{}
	for _, m := range a.Models {
		if m == "" || seenM[m] {
			return nil, fmt.Errorf("%w: %s: empty or duplicate model allowlist entry", ErrAnchor, origin)
		}
		seenM[m] = true
	}
	// The Skill allowlist is a closed, exact set: name@version only (no
	// "latest", no ranges — the same discipline as the catalog), no
	// duplicates, bounded like the workflow set.
	if len(a.Skills) > maxWorkflowBundles {
		return nil, fmt.Errorf("%w: %s: the skill allowlist exceeds %d entries — an anchor is a closed, reviewable declaration", ErrAnchor, origin, maxWorkflowBundles)
	}
	seenS := map[string]bool{}
	for _, s := range a.Skills {
		if !skillRefSyntax.MatchString(s) {
			return nil, fmt.Errorf("%w: %s: skill allowlist entry %q must be an exact name@version", ErrAnchor, origin, s)
		}
		if seenS[s] {
			return nil, fmt.Errorf("%w: %s: duplicate skill allowlist entry %q", ErrAnchor, origin, s)
		}
		seenS[s] = true
	}
	a.SHA256 = hashBytes(raw)
	a.Raw = raw
	return &a, nil
}

// registryEntry is one anchors-registry binding (the same governed
// registry shape as every other plane).
type registryEntry struct {
	Name     string `json:"name"`
	Version  int    `json:"version"`
	Artifact string `json:"artifact_sha256"`
	State    string `json:"state"`
	Steward  string `json:"steward,omitempty"`
	// Door provenance (D-R-2): the reliance decision record, two-way
	// bound. Required from registry version 2 when loaded from the
	// governed tree; the observed copy is compared by identity only.
	DecisionRef    string `json:"decision_ref,omitempty"`
	DecisionSHA256 string `json:"decision_sha256,omitempty"`
}

// AdmitAnchor performs the D-G1-1A two-step for Open:
//
//	resolve exact anchor bytes (unavailable → refusal)
//	verify the operator's expected hash (mismatch → refusal)
//	establish Governance admission against the anchors registry
//	    (unregistered → refusal; withdrawn → refusal)
//	→ ACTIVE anchor, ready to freeze.
//
// The caller-supplied path and expectedSHA identify the REQUEST;
// only the registry resolution establishes governed status.
func AdmitAnchor(anchorPath, expectedSHA, anchorsRegistryPath string) (*Anchor, error) {
	raw, err := readGoverned(anchorPath, maxAnchorBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: anchor bytes unavailable: %v", ErrAdmission, err)
	}
	if !shaSyntax.MatchString(expectedSHA) {
		return nil, fmt.Errorf("%w: the operator's expected anchor hash must be a sha256 hex digest", ErrAdmission)
	}
	if hashBytes(raw) != expectedSHA {
		return nil, fmt.Errorf("%w: anchor bytes do not match the operator's expected hash", ErrAdmission)
	}
	a, err := ParseAnchor(raw, anchorPath)
	if err != nil {
		return nil, err
	}
	reg, err := LoadRegistry(anchorsRegistryPath)
	if err != nil {
		return nil, err
	}
	var admitted *registryEntry
	for i := range reg.Entries {
		if reg.Entries[i].Artifact == a.SHA256 {
			admitted = &reg.Entries[i]
		}
	}
	if admitted == nil {
		return nil, fmt.Errorf("%w: anchor %s is not a Governance-registered deployment anchor — a matching hash is an identifier, never an admission claim", ErrAdmission, a.SHA256[:12])
	}
	if admitted.State == "withdrawn" {
		return nil, fmt.Errorf("%w: anchor %s@%d is withdrawn — a superseded deployment definition cannot open", ErrAdmission, admitted.Name, admitted.Version)
	}
	if admitted.Name != a.Name || admitted.Version != a.Deployment {
		return nil, fmt.Errorf("%w: anchor self-declaration (%s@%d) disagrees with the registration (%s@%d) — two-way identity", ErrAdmission, a.Name, a.Deployment, admitted.Name, admitted.Version)
	}
	return a, nil
}

// HashFile returns the sha256 of a file's exact bytes.
func HashFile(path string) (string, error) {
	b, err := readGoverned(path, 64<<20)
	if err != nil {
		return "", err
	}
	return hashBytes(b), nil
}

// maxPinnedFileBytes bounds any single file entering a pin
// computation (close-review LOW-2: an unbounded read at Open is a
// memory-exhaustion surface).
const maxPinnedFileBytes = 8 << 20

// HashDir fingerprints a directory tree deterministically: sha256
// over each regular file's slash-relative path and contents, sorted
// by path — the instruction-root content identity the anchor pins.
//
// A non-regular entry (symlink, device, socket, fifo) is a REFUSAL,
// never a skip (close-review CRITICAL-1): consumers walk these trees
// with different selection rules — the instruction loader follows
// symlinks and reads them — so an entry this function skipped but a
// consumer reads would be unpinned content inside a pinned tree.
// Hashing and reading must never disagree about the file set.
func HashDir(root string) (string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%w: %s: non-regular entry in a pinned tree — content pins admit regular files only", ErrAnchor, p)
		}
		files = append(files, p)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(files)
	h := sha256.New()
	// LENGTH-PREFIXED framing (close-review M-4): NUL separators alone
	// admit collisions — {a:"", b:"c"} and {a:"\0b\0c"} produce the
	// same stream, so an attacker able to write one file and delete a
	// sibling could impersonate the original tree. Lengths make the
	// encoding injective.
	writeField := func(b []byte) {
		var n [8]byte
		l := uint64(len(b))
		for i := 0; i < 8; i++ {
			n[7-i] = byte(l >> (8 * i))
		}
		h.Write(n[:])
		h.Write(b)
	}
	var count [8]byte
	c := uint64(len(files))
	for i := 0; i < 8; i++ {
		count[7-i] = byte(c >> (8 * i))
	}
	h.Write(count[:])
	for _, f := range files {
		rel, err := filepath.Rel(root, f)
		if err != nil {
			return "", err
		}
		b, err := readGoverned(f, maxPinnedFileBytes)
		if err != nil {
			return "", err
		}
		writeField([]byte(filepath.ToSlash(rel)))
		writeField(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func readGoverned(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: not a regular file", path)
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%s: exceeds %d bytes", path, maxBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("%s: exceeds %d bytes", path, maxBytes)
	}
	return raw, nil
}

func checkNoDuplicateKeys(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	type frame struct {
		object    bool
		keys      map[string]bool
		nextIsKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.nextIsKey {
				if key, ok := tok.(string); ok {
					if top.keys[key] {
						return fmt.Errorf("duplicate key %q", key)
					}
					top.keys[key] = true
					top.nextIsKey = false
					continue
				}
			}
		}
		switch d := tok.(type) {
		case json.Delim:
			switch d {
			case '{':
				stack = append(stack, &frame{object: true, keys: map[string]bool{}, nextIsKey: true})
			case '[':
				stack = append(stack, &frame{})
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1].object {
					stack[len(stack)-1].nextIsKey = true
				}
			}
			continue
		}
		if len(stack) > 0 && stack[len(stack)-1].object {
			stack[len(stack)-1].nextIsKey = true
		}
	}
}

// --- Append-only wall (owner finding 2) ------------------------------

// Registry is a loaded anchors-registry state. Loading it separately
// from admission lets a caller hold a PRIOR observed state and prove
// the registry only ever grew: deployment@N → H must mean the same
// thing forever, or the identity semantics collapse.
type Registry struct {
	Version int             `json:"version"`
	Kind    string          `json:"kind"`
	Entries []registryEntry `json:"entries"`

	Hash string `json:"-"`
}

// LoadRegistry reads and validates an anchors registry fail-closed.
func LoadRegistry(path string) (*Registry, error) {
	raw, err := readGoverned(path, maxAnchorBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: anchors registry unavailable: %v", ErrAdmission, err)
	}
	r, err := parseRegistry(raw)
	if err != nil {
		return nil, err
	}
	if r.Version >= 2 {
		// D-R-2: the governed tree's registry carries provenance for every
		// reliance act; the binding is checked here, where the tree is.
		ddir := decisions.Dir(path)
		for _, e := range r.Entries {
			if e.DecisionRef == "" || !shaSyntax.MatchString(e.DecisionSHA256) {
				return nil, fmt.Errorf("%w: anchors registry: %s@%d: decision_ref and decision_sha256 are required from registry version 2", ErrAdmission, e.Name, e.Version)
			}
			if _, err := decisions.Bind(ddir, e.DecisionRef, e.DecisionSHA256, e.Name, e.Version, e.Artifact); err != nil {
				return nil, fmt.Errorf("%w: anchors registry: %s@%d: %v", ErrAdmission, e.Name, e.Version, err)
			}
		}
	}
	return r, nil
}

func parseRegistry(raw []byte) (*Registry, error) {
	if err := checkNoDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%w: anchors registry: %v", ErrAdmission, err)
	}
	var reg Registry
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&reg); err != nil || dec.More() {
		return nil, fmt.Errorf("%w: anchors registry unparseable", ErrAdmission)
	}
	if reg.Version < 1 || reg.Kind != "deployment-anchors" {
		return nil, fmt.Errorf("%w: not a deployment-anchors registry", ErrAdmission)
	}
	seen := map[string]bool{}
	seenArtifact := map[string]bool{}
	for i := range reg.Entries {
		e := &reg.Entries[i]
		if !nameSyntax.MatchString(e.Name) || e.Version < 1 || !shaSyntax.MatchString(e.Artifact) {
			return nil, fmt.Errorf("%w: anchors registry: malformed entry", ErrAdmission)
		}
		if e.State != "active" && e.State != "withdrawn" {
			return nil, fmt.Errorf("%w: anchors registry: unknown state %q", ErrAdmission, e.State)
		}
		key := fmt.Sprintf("%s@%d", e.Name, e.Version)
		if seen[key] {
			return nil, fmt.Errorf("%w: anchors registry: duplicate registration %s", ErrAdmission, key)
		}
		seen[key] = true
		if seenArtifact[e.Artifact] {
			return nil, fmt.Errorf("%w: anchors registry: artifact %s registered more than once — admission must not depend on entry order", ErrAdmission, e.Artifact[:12])
		}
		seenArtifact[e.Artifact] = true
	}
	reg.Hash = hashBytes(raw)
	return &reg, nil
}

// CheckAppendOnly verifies this registry state against a previously
// observed one: every prior registration must still be present with
// the SAME anchor hash, and state may only advance active→withdrawn
// (owner finding 2 — deployment@N → H1 must never become H2).
// Deletion, rebinding, and un-withdrawal are refusals: out-of-band
// registry mutation is DETECTED rather than trusted (Register T).
func (r *Registry) CheckAppendOnly(prior *Registry) error {
	if prior == nil {
		return nil
	}
	current := map[string]registryEntry{}
	for _, e := range r.Entries {
		current[fmt.Sprintf("%s@%d", e.Name, e.Version)] = e
	}
	for _, p := range prior.Entries {
		key := fmt.Sprintf("%s@%d", p.Name, p.Version)
		cur, ok := current[key]
		if !ok {
			return fmt.Errorf("%w: %s disappeared — anchors are append-only so past deployments stay interpretable", ErrAdmission, key)
		}
		if cur.Artifact != p.Artifact {
			return fmt.Errorf("%w: %s rebound to a different anchor — deployment identity is immutable", ErrAdmission, key)
		}
		if p.State == "withdrawn" && cur.State != "withdrawn" {
			return fmt.Errorf("%w: %s un-withdrawn — state advances active→withdrawn only", ErrAdmission, key)
		}
	}
	return nil
}

// ObservedRegistryPath is where an orchestrator persists the last
// observed anchors-registry state under its own record root, so the
// append-only wall spans restarts.
func ObservedRegistryPath(stateRoot string) string {
	return filepath.Join(stateRoot, "deployment", "anchors-observed.json")
}

// LoadObserved reads a previously observed registry state, if any.
// A missing file is not an error: the first Open has no prior.
func LoadObserved(stateRoot string) (*Registry, error) {
	raw, err := readGoverned(ObservedRegistryPath(stateRoot), maxAnchorBytes)
	if err != nil {
		return nil, nil
	}
	return parseRegistry(raw)
}

// --- Read-path re-verification (owner finding 3) ---------------------

// VerifyAnchorRecord re-establishes, from durable evidence alone,
// what deployment governed a recorded task: the recorded identity
// must be the hash of the recorded anchor BYTES, those bytes must
// still parse, and they must still resolve to a Governance-admitted
// anchor in the registry. Recording the anchor is not sufficient —
// the record identifies the anchor; the registry and the bytes prove
// what that identity meant (the G2 principle applied to G1).
func VerifyAnchorRecord(recordedHash string, anchorBytes []byte, anchorsRegistryPath string) (*Anchor, error) {
	if recordedHash == "unanchored" {
		return nil, fmt.Errorf("%w: the task record declares an unanchored run — no deployment authority to verify", ErrAdmission)
	}
	if !shaSyntax.MatchString(recordedHash) {
		return nil, fmt.Errorf("%w: task record carries no well-formed deployment-anchor identity", ErrAdmission)
	}
	if len(anchorBytes) == 0 {
		return nil, fmt.Errorf("%w: anchor bytes for %s are not available in the record", ErrAdmission, recordedHash[:12])
	}
	if hashBytes(anchorBytes) != recordedHash {
		return nil, fmt.Errorf("%w: recorded anchor bytes do not hash to the recorded identity", ErrAdmission)
	}
	a, err := ParseAnchor(anchorBytes, "recorded-anchor")
	if err != nil {
		return nil, err
	}
	reg, err := LoadRegistry(anchorsRegistryPath)
	if err != nil {
		return nil, err
	}
	for _, e := range reg.Entries {
		if e.Artifact != a.SHA256 {
			continue
		}
		if e.Name != a.Name || e.Version != a.Deployment {
			return nil, fmt.Errorf("%w: recorded anchor's self-declaration disagrees with its registration", ErrAdmission)
		}
		// A withdrawn anchor still EXPLAINS a past execution:
		// withdrawal stops new opens, it does not rewrite history.
		return a, nil
	}
	return nil, fmt.Errorf("%w: the anchor that governed this task is no longer registered — its deployment is uninterpretable", ErrAdmission)
}
