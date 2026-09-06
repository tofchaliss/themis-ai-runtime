// Package execution implements Layer 5 of the Themis AI Harness: the
// Execution Environment. Governing design:
// openspec/changes/layer-05-execution-environment/design.md (§2
// locked decisions D-L5-1..10, §3 grill record Q-L5-1..12).
// Locked: ProvisionSpec ⊆ WorkspaceExecutionCeiling; configuration is
// never de-isolation; provisioning is execution; the lifecycle is a
// single monotonic state machine with an irreversible SEALED state;
// a provider declares properties at delivered strength — never
// "supported"; absence of an enforcement mechanism is never
// represented as enforcement.
package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrCeilingInvalid = errors.New("invalid workspace execution ceiling")
	ErrSpecInvalid    = errors.New("invalid provision spec")
)

// Limit dimensions are a closed vocabulary with explicit units
// (Q-L5-9: no bare "CPU"/"memory"). Absent = the dimension does not
// exist.
const (
	DimWallDeadlineS = "wall_deadline_s"
	DimCPUTimeS      = "cpu_time_s"
	DimMemBytes      = "mem_bytes"
	DimDiskBytes     = "disk_bytes"
	DimFileBytes     = "file_bytes"
	DimProcCount     = "proc_count"
)

var knownDimensions = map[string]bool{
	DimWallDeadlineS: true, DimCPUTimeS: true, DimMemBytes: true,
	DimDiskBytes: true, DimFileBytes: true, DimProcCount: true,
}

// Strength is the declared/required guarantee for a limit dimension.
// "supported" is not in the vocabulary (Q-L5-9).
type Strength string

const (
	StrengthEnforced Strength = "enforced"
	StrengthObserved Strength = "observed"
)

// WorkspaceExecutionCeiling is the governed envelope ceiling: the
// widest environment any spec may request (D-L5-2). Hard floors
// (network denial, empty environment, identity non-elevation) are
// deliberately NOT fields here — they are outside the configuration
// vocabulary entirely (Q-L5-1: configuration is never de-isolation).
type WorkspaceExecutionCeiling struct {
	Version int `json:"version"`
	// MirrorRoot: the only directory local repository mirrors may be
	// provisioned from (Q-L5-3: v1 provisioning sources are local
	// mirrors only). Absolute — a relative root would make the
	// provisioning boundary depend on process cwd.
	MirrorRoot         string `json:"mirror_root"`
	MaxWallDeadlineSec int64  `json:"max_wall_deadline_sec"`
	// Egress bounds (D-L5-5; enforced at the Artifact Egress
	// Contract): specs narrow these, never widen.
	MaxFileBytes  int64 `json:"max_file_bytes"`
	MaxTotalBytes int64 `json:"max_total_bytes"`
	MaxFileCount  int64 `json:"max_file_count"`
	// Ceilings for the remaining closed-vocabulary dimensions, so
	// spec ⊆ ceiling is total — a future enforced-memory provider
	// must not inherit an unconstrained dimension (M1 security
	// review LOW).
	MaxMemBytes   int64 `json:"max_mem_bytes"`
	MaxCPUTimeSec int64 `json:"max_cpu_time_sec"`
	MaxProcCount  int64 `json:"max_proc_count"`

	Hash string `json:"-"`
}

// max returns the ceiling bound for a limit dimension; containment
// is total over the closed vocabulary.
func (c *WorkspaceExecutionCeiling) max(dim string) int64 {
	switch dim {
	case DimWallDeadlineS:
		return c.MaxWallDeadlineSec
	case DimFileBytes:
		return c.MaxFileBytes
	case DimDiskBytes:
		return c.MaxTotalBytes
	case DimMemBytes:
		return c.MaxMemBytes
	case DimCPUTimeS:
		return c.MaxCPUTimeSec
	case DimProcCount:
		return c.MaxProcCount
	}
	return 0 // unknown dimension: no headroom, fail closed
}

// LoadCeiling: fail-closed governed-artifact posture.
func LoadCeiling(path string) (*WorkspaceExecutionCeiling, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCeilingInvalid, path, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var c WorkspaceExecutionCeiling
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrCeilingInvalid, path, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrCeilingInvalid, path)
	}
	if c.Version < 1 {
		return nil, fmt.Errorf("%w: %s: version required", ErrCeilingInvalid, path)
	}
	if !filepath.IsAbs(c.MirrorRoot) {
		return nil, fmt.Errorf("%w: mirror root must be absolute", ErrCeilingInvalid)
	}
	if c.MaxWallDeadlineSec <= 0 || c.MaxFileBytes <= 0 || c.MaxTotalBytes <= 0 ||
		c.MaxFileCount <= 0 || c.MaxMemBytes <= 0 || c.MaxCPUTimeSec <= 0 || c.MaxProcCount <= 0 {
		return nil, fmt.Errorf("%w: all ceiling bounds must be positive", ErrCeilingInvalid)
	}
	sum := sha256.Sum256(raw)
	c.Hash = hex.EncodeToString(sum[:])
	return &c, nil
}

// LimitReq is one required limit dimension in a spec. Strength
// defaults to enforced — an observed provider can never silently
// satisfy an enforced requirement (Q-L5-9).
type LimitReq struct {
	Dimension string   `json:"dimension"`
	Value     int64    `json:"value"`
	Strength  Strength `json:"strength,omitempty"`
}

// pinnedSHA: full 40-hex commit SHA only. A branch name is
// structurally not a pin (Q-L5-3: pinned SHA mandatory, never a
// branch).
var pinnedSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// repoName confines repository identity to a mirror-relative name;
// path traversal and URL shapes are rejected before any confinement
// arithmetic runs.
var repoName = regexp.MustCompile(`^[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)*$`)

// ProvisionSpec is the per-task instantiation of the ceiling
// (ProvisionSpec ⊆ WorkspaceExecutionCeiling — the cross-layer
// principle: a caller may narrow a governed ceiling, never define
// one).
type ProvisionSpec struct {
	Version   int        `json:"version"`
	TaskID    string     `json:"task_id"`
	Repo      string     `json:"repo"`
	PinnedSHA string     `json:"pinned_sha"`
	Limits    []LimitReq `json:"limits"`

	Hash string `json:"-"`
}

// LoadSpec: fail-closed; validates the spec in isolation. Ceiling
// containment is checked separately at admission (ValidateAgainst) so
// the two failure classes stay distinct in the trace.
func LoadSpec(path string) (*ProvisionSpec, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrSpecInvalid, path, err)
	}
	return parseSpec(raw, path)
}

func parseSpec(raw []byte, src string) (*ProvisionSpec, error) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var s ProvisionSpec
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrSpecInvalid, src, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrSpecInvalid, src)
	}
	if s.Version < 1 || s.TaskID == "" {
		return nil, fmt.Errorf("%w: %s: version and task_id required", ErrSpecInvalid, src)
	}
	if !repoName.MatchString(s.Repo) {
		return nil, fmt.Errorf("%w: bad repository name %q", ErrSpecInvalid, s.Repo)
	}
	// The character class admits "." and ".." as segments; refuse
	// them explicitly so traversal dies here, not only at the
	// confinement layer (defense in depth, M1 security review LOW).
	for _, seg := range strings.Split(s.Repo, "/") {
		if seg == "." || seg == ".." {
			return nil, fmt.Errorf("%w: bad repository name %q", ErrSpecInvalid, s.Repo)
		}
	}
	if !pinnedSHA.MatchString(s.PinnedSHA) {
		return nil, fmt.Errorf("%w: pinned_sha must be a full 40-hex commit SHA, got %q — branch names are not pins", ErrSpecInvalid, s.PinnedSHA)
	}
	seen := map[string]bool{}
	hasDeadline := false
	for i := range s.Limits {
		l := &s.Limits[i]
		if !knownDimensions[l.Dimension] {
			return nil, fmt.Errorf("%w: unknown limit dimension %q", ErrSpecInvalid, l.Dimension)
		}
		if seen[l.Dimension] {
			return nil, fmt.Errorf("%w: duplicate limit dimension %q", ErrSpecInvalid, l.Dimension)
		}
		seen[l.Dimension] = true
		if l.Value <= 0 {
			return nil, fmt.Errorf("%w: limit %q must be positive", ErrSpecInvalid, l.Dimension)
		}
		switch l.Strength {
		case "":
			l.Strength = StrengthEnforced // default: enforced (Q-L5-9)
		case StrengthEnforced, StrengthObserved:
		default:
			return nil, fmt.Errorf("%w: limit %q has unknown strength %q", ErrSpecInvalid, l.Dimension, l.Strength)
		}
		if l.Dimension == DimWallDeadlineS {
			hasDeadline = true
			if l.Strength != StrengthEnforced {
				return nil, fmt.Errorf("%w: wall_deadline_s must be required at enforced strength", ErrSpecInvalid)
			}
		}
	}
	if !hasDeadline {
		return nil, fmt.Errorf("%w: %s: wall_deadline_s is mandatory — an unbounded environment is not a valid request", ErrSpecInvalid, src)
	}
	sum := sha256.Sum256(raw)
	s.Hash = hex.EncodeToString(sum[:])
	return &s, nil
}

// Limit returns the requested limit for a dimension, if present.
func (s *ProvisionSpec) Limit(dim string) (LimitReq, bool) {
	for _, l := range s.Limits {
		if l.Dimension == dim {
			return l, true
		}
	}
	return LimitReq{}, false
}

// ValidateAgainst checks spec ⊆ ceiling — total over the closed
// dimension vocabulary. Refusal here is the
// environment-provision-failed class: L5 refusing an environment,
// never a request.
func (s *ProvisionSpec) ValidateAgainst(c *WorkspaceExecutionCeiling) error {
	for _, l := range s.Limits {
		if m := c.max(l.Dimension); l.Value > m {
			return fmt.Errorf("%w: %s %d exceeds ceiling %d", ErrSpecInvalid, l.Dimension, l.Value, m)
		}
	}
	return nil
}
