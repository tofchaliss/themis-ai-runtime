package ratchet

// Persistence: L11 instance artifacts are durably represented as L6
// content-addressed objects under the EXISTING evidence-payload
// class — no L11 event stream, no candidate_state table, no third
// durable plane (D-L11-11, owner amendment). Storing is the
// completion of an accepted invocation (no-discard, D-L11-14 §3);
// reading back is hash-verified by L6 itself (D-L10-11 for free).

import (
	"fmt"

	"github.com/tofchaliss/themis/state"
)

// StoreInstance persists any L11 instance artifact (comparison
// package, refusal fact, regression package, candidate, evaluation
// plan) as an evidence-payload object. Returns the L6 ObjectID and
// the canonical bytes. The ObjectID embeds the content hash — the
// instance identity IS the address.
func StoreInstance(store *state.ObjectStore, v any) (string, []byte, error) {
	canonical, err := CanonicalBytes(v)
	if err != nil {
		return "", nil, err
	}
	id, err := store.StoreObject(state.ObjEvidencePayload, canonical)
	if err != nil {
		return "", nil, fmt.Errorf("%w: durable store: %v", ErrResolve, err)
	}
	return id, canonical, nil
}

// LoadComparison reads a comparison package back by ObjectID with a
// strict parse. L6 verifies content against the address on read.
func LoadComparison(store *state.ObjectStore, id string) (*ComparisonPackage, error) {
	b, err := store.GetObject(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResolve, err)
	}
	dec := jsonDecoder(b)
	var p ComparisonPackage
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("%w: object %s is not a comparison package: %v", ErrResolve, id, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: object %s: trailing content", ErrResolve, id)
	}
	if p.Artifact != "l11-comparison" || p.Schema != 1 {
		return nil, fmt.Errorf("%w: object %s is not an l11-comparison artifact", ErrResolve, id)
	}
	return &p, nil
}
