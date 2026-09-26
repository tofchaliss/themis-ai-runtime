// Package client is the harness's read door to the live Themis
// authority (D-I-3): an HTTP client behind tools.ThemisSeam. It reads
// a Finding from Governance and a Product from Registry, PROJECTS the
// response to the closed field set the read door serves — positions
// and proposals are never served to a model — verifies the response
// identity before returning, and fails closed on any transport or
// shape failure. The credential is seam-local: it comes from the
// environment into this client and appears in no record, registry,
// anchor, or model context.
//
// This package imports nothing of Themis: the seam is HTTP, never a Go
// dependency (D-I-7).
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/integrations/themis/contracts"
)

// ErrUnavailable: the authority did not serve the record (transport
// failure, non-200, malformed body, identity mismatch). L4 maps any
// seam error to seam-unavailable; the reason is kept for the operator.
var ErrUnavailable = errors.New("themis unavailable")

const (
	maxBodyBytes = 1 << 20
	headerKey    = "X-API-Key"
)

var uuidSyntax = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Client implements tools.ThemisSeam over the pinned contract.
type Client struct {
	contract *contracts.Contract
	key      string
	hc       *http.Client
}

// New binds a client to a loaded contract. The key may be empty only
// where the estate runs without inbound auth (dev); production
// deployments issue a read-scope key (D-I-8).
func New(c *contracts.Contract, key string, hc *http.Client) (*Client, error) {
	if c == nil {
		return nil, fmt.Errorf("%w: no contract", contracts.ErrContract)
	}
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{contract: c, key: key, hc: hc}, nil
}

// ContractHash is the pin the anchor's themis_contract must equal;
// L7 compares it at Open (the seam's own statement of what it was
// built over, beside L7's independent hash of the file).
func (c *Client) ContractHash() string { return c.contract.Hash }

// component is the projected Finding component (Governance's
// Component view minus nothing security-relevant; all fields are
// Finding content, not decisions).
type component struct {
	PURL       string `json:"purl,omitempty"`
	Name       string `json:"name,omitempty"`
	Version    string `json:"version,omitempty"`
	Ecosystem  string `json:"ecosystem,omitempty"`
	Source     string `json:"source,omitempty"`
	ClaimClass string `json:"claim_class,omitempty"`
}

// findingView is the subset of Governance's FindingView the client
// decodes. Unknown fields (current_position, positions, proposals, …)
// are decoded into nothing and never re-serialized: the projection is
// by construction, not by deletion.
type findingView struct {
	ID          string      `json:"id"`
	ReleaseID   string      `json:"release_id"`
	FaultlineID string      `json:"faultline_id"`
	CVE         string      `json:"cve"`
	Stage       string      `json:"stage"`
	Components  []component `json:"components"`
}

type productView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Read serves one governed record by kind and id (tools.ThemisSeam).
// The bytes returned are the PROJECTED record, canonical JSON, which
// L4 frames as governed-record and records by hash.
func (c *Client) Read(kind, id string) ([]byte, error) {
	if !uuidSyntax.MatchString(id) {
		return nil, fmt.Errorf("%w: %s id %q is not a UUID", ErrUnavailable, kind, id)
	}
	var base, path string
	switch kind {
	case "finding":
		base, path = c.contract.GovernanceBaseURL, "/api/v1/findings/"+url.PathEscape(id)
	case "product":
		base, path = c.contract.RegistryBaseURL, "/api/v1/products/"+url.PathEscape(id)
	default:
		return nil, fmt.Errorf("%w: unknown record kind %q", ErrUnavailable, kind)
	}
	body, err := c.get(base + path)
	if err != nil {
		return nil, err
	}
	switch kind {
	case "finding":
		var v findingView
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, fmt.Errorf("%w: finding response unparseable", ErrUnavailable)
		}
		if v.ID != id {
			return nil, fmt.Errorf("%w: finding response identity %q is not the requested %q", ErrUnavailable, v.ID, id)
		}
		if v.Components == nil {
			v.Components = []component{}
		}
		return json.Marshal(v)
	default:
		var v productView
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, fmt.Errorf("%w: product response unparseable", ErrUnavailable)
		}
		if v.ID != id {
			return nil, fmt.Errorf("%w: product response identity %q is not the requested %q", ErrUnavailable, v.ID, id)
		}
		return json.Marshal(v)
	}
}

func (c *Client) get(u string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.hc.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")
	if c.key != "" {
		req.Header.Set(headerKey, c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnavailable, sanitize(err.Error()))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: reading response", ErrUnavailable)
	}
	if len(body) > maxBodyBytes {
		return nil, fmt.Errorf("%w: response exceeds %d bytes", ErrUnavailable, maxBodyBytes)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnavailable, resp.StatusCode)
	}
	return body, nil
}

// sanitize keeps transport errors free of the credential (Go's
// transport never echoes headers, but the rule is stated, not assumed).
func sanitize(s string) string {
	return strings.ReplaceAll(s, headerKey, "")
}
