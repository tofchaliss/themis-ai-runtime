// Package contracts loads the governed Themis interface contract the
// deployment anchor pins as `themis_contract` (D-I-3): the authorized
// Governance and Registry endpoints and the identity of the OpenAPI
// specifications they serve, at a named Themis commit. The contract
// pins the INTERFACE, never live data and never the running Themis
// binary; Findings and Products are read live and captured as
// execution-time governed records.
package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/internal/strictjson"
)

// ErrContract: the contract file is absent, malformed, or outside its
// closed schema. Fail closed: a deployment that pins a contract it
// cannot read does not open.
var ErrContract = errors.New("invalid themis contract")

const maxContractBytes = 64 << 10

var (
	shaSyntax    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	commitSyntax = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Contract is the closed schema of policies/themis/contract.json.
type Contract struct {
	Version int `json:"version"`
	// Base URLs of the two Themis read APIs the harness is authorized
	// to consume (loopback on the shared host, D-I-8).
	GovernanceBaseURL string `json:"governance_base_url"`
	RegistryBaseURL   string `json:"registry_base_url"`
	// SHA-256 of the two OpenAPI specifications at ThemisCommit —
	// the interface identity this deployment integrates against.
	GovernanceSpecSHA256 string `json:"governance_spec_sha256"`
	RegistrySpecSHA256   string `json:"registry_spec_sha256"`
	// ThemisCommit names the Themis source commit those specifications
	// come from; the host checks the deployed estate is at it (D-I-8).
	ThemisCommit string `json:"themis_commit"`

	Hash string `json:"-"` // sha256 of the exact file bytes — the anchor pin
	Raw  []byte `json:"-"`
}

// Load reads the contract fail-closed: a regular file, bounded, no
// duplicate keys, no unknown fields, every field well-formed.
func Load(path string) (*Contract, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, path, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s: not a regular file", ErrContract, path)
	}
	if fi.Size() > maxContractBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", ErrContract, path, maxContractBytes)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, path, err)
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxContractBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, path, err)
	}
	return Parse(raw, path)
}

// Parse validates exact bytes.
func Parse(raw []byte, origin string) (*Contract, error) {
	if len(raw) > maxContractBytes {
		return nil, fmt.Errorf("%w: %s: exceeds %d bytes", ErrContract, origin, maxContractBytes)
	}
	if err := strictjson.Check(raw); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, origin, err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var c Contract
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrContract, origin, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: %s: trailing content", ErrContract, origin)
	}
	if c.Version != 1 {
		return nil, fmt.Errorf("%w: %s: version must be 1", ErrContract, origin)
	}
	for name, u := range map[string]string{"governance_base_url": c.GovernanceBaseURL, "registry_base_url": c.RegistryBaseURL} {
		p, err := url.Parse(u)
		if err != nil || (p.Scheme != "http" && p.Scheme != "https") || p.Host == "" || p.RawQuery != "" || p.Fragment != "" || p.User != nil {
			return nil, fmt.Errorf("%w: %s: %s must be an absolute http(s) URL with host and no query, fragment, or credentials", ErrContract, origin, name)
		}
	}
	for name, h := range map[string]string{"governance_spec_sha256": c.GovernanceSpecSHA256, "registry_spec_sha256": c.RegistrySpecSHA256} {
		if !shaSyntax.MatchString(h) {
			return nil, fmt.Errorf("%w: %s: %s must be a sha256 hex digest", ErrContract, origin, name)
		}
	}
	if !commitSyntax.MatchString(c.ThemisCommit) {
		return nil, fmt.Errorf("%w: %s: themis_commit must be a full 40-hex commit", ErrContract, origin)
	}
	sum := sha256.Sum256(raw)
	c.Hash = hex.EncodeToString(sum[:])
	c.Raw = raw
	return &c, nil
}
