package context

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// The ManagementPolicy is Layer 3's governed artifact (design §3,
// Q-L3-1/7): all selection discretion lives here — versioned,
// reviewed, hashed into the trace. The runtime executes it without
// judgment. The contract says what evidence matters; this policy says
// how much fits; L7 (future) selects which policy governs a task.
// Budget overrides are governance-plane acts, not parameters.

var (
	ErrPolicyInvalid = errors.New("invalid management policy")
	ErrBudget        = errors.New("required and undroppable context exceeds the budget")
)

// estimatorVersion is the fixed deterministic token estimator
// (D-L3-5): ceil(bytes/4). Provider-exact tokenizers arrive later as
// registered computations; the composed-size caps remain the hard
// backstop.
const estimatorVersion = "bytes/4-v1"

func estimateTokens(n int) int { return (n + 3) / 4 }

// rankKeys legal set: mechanically computable from ItemRef metadata
// only (Q-L3-1 type-system boundary). Every chain implicitly ends
// with the hash tiebreak so ordering is total and deterministic —
// no runtime "best judgment" tie-break exists.
var legalRankKeys = map[string]bool{
	"size_asc": true, "size_desc": true, "kind": true, "version": true,
}

// ManagementPolicy is the loaded, validated artifact.
type ManagementPolicy struct {
	Version   int      `json:"version"`
	Name      string   `json:"name"`
	Budget    int      `json:"budget"`              // estimator tokens
	DropOrder []string `json:"drop_order"`          // slots droppable by this policy, dropped-first order
	RankKeys  []string `json:"rank_keys,omitempty"` // within-slot drop ranking (worst-ranked dropped first)
	Dedup     string   `json:"dedup"`               // "none" | "within-class"
	Hash      string   `json:"-"`
}

// LoadManagementPolicy: fail-closed artifact posture (L1/L2 pattern —
// unknown fields, trailing bytes, floors, legal enums).
func LoadManagementPolicy(path string) (*ManagementPolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPolicyInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var p ManagementPolicy
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrPolicyInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrPolicyInvalid, path)
	}
	if p.Version < 1 || p.Name == "" || p.Budget <= 0 {
		return nil, fmt.Errorf("%w: %s: version, name, and positive budget required", ErrPolicyInvalid, path)
	}
	if p.Dedup != "none" && p.Dedup != "within-class" {
		return nil, fmt.Errorf("%w: %s: unknown dedup mode %q", ErrPolicyInvalid, path, p.Dedup)
	}
	seen := map[string]bool{}
	for _, s := range p.DropOrder {
		if s == "" || seen[s] {
			return nil, fmt.Errorf("%w: %s: empty or duplicate drop-order slot %q", ErrPolicyInvalid, path, s)
		}
		seen[s] = true
	}
	for _, k := range p.RankKeys {
		if !legalRankKeys[k] {
			return nil, fmt.Errorf("%w: %s: rank key %q is not in the declared legal set", ErrPolicyInvalid, path, k)
		}
	}
	sum := sha256.Sum256(raw)
	p.Hash = hex.EncodeToString(sum[:])
	return &p, nil
}

// CompressorRegistration is the Q-L3-6 SEAM ONLY: no compressor
// implementation exists in v1 and none may activate without a
// dedicated grill of its proposition vocabulary. A future compressor
// produces derived-class items referencing original hashes; originals
// always remain reconstructable in the canonical trace.
type CompressorRegistration struct {
	ProducerID string
	Version    string
	ConfigHash string
	// PermittedVocabulary is intentionally absent until the dedicated
	// grill defines it — its absence is what makes activation
	// impossible, not an oversight.
}
