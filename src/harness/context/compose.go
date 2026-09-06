package context

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tofchaliss/themis/instructions"
	"github.com/tofchaliss/themis/runtime/model"
)

// Payload is one composed model payload plus its complete delivery
// trace — with the tool-execution trace (L4-era), it must
// reconstruct every model-visible byte (design §3, Q-L2-1 amendment
// 4). There is no append-to-conversation API: a new epoch is a new
// Compose (Q-L2-1/Q-L1-5).
type Payload struct {
	Messages     []model.Message // [system (EIS verbatim), user (context)]
	ContractHash string
	EISHash      string
	RenderHash   string // hash of the delivered system-message text
	PayloadHash  string // hash of the canonical length-framed record
	Slots        []SlotState
}

// Model-facing furniture: descriptive only, never behavioral
// (L2-5.7). The renderer-furniture rule, data-plane edition — any
// directive text here fails review.
const contextHeading = "## Context"

// Compose renders the gathered evidence with the EIS into the model
// payload. The EIS system message is produced by the L1 seam
// (verbatim, policy-hash-bound); the user message renders each
// contract slot in order — delivered items inside content-bound
// fences, absent slots as typed availability markers with zero
// substantive content. Dual-reader discipline: the model view uses
// visible delimiters with collision refusal; the canonical record is
// length-framed and is what PayloadHash covers.
func Compose(set *instructions.EffectiveSet, policy *instructions.Policy, g *Gathered) (*Payload, error) {
	if g == nil || g.Contract == nil {
		return nil, fmt.Errorf("%w: no gathered context", ErrCompose)
	}
	sysMsg, renderHash, err := set.SystemMessage(policy)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompose, err)
	}

	if !g.validated {
		// A Gathered that did not come out of Gather carries no
		// Plan ⊆ Contract guarantee — refuse (security review LOW-1).
		return nil, fmt.Errorf("%w: gathered context did not pass Gather validation", ErrCompose)
	}
	// One composition-wide, content-derived fence guards every frame
	// (security review HIGH-1/2). Candidates derive deterministically
	// from all delivered content; the first candidate appearing
	// nowhere in any evidence or metadata is selected. Embedding a
	// candidate skips to the next; embedding all of them is a sha256
	// fixed-point across every derived candidate — not constructible.
	// No secrecy (L2-5.6), no rewriting (L2-5.5): fence *selection*
	// adapts, evidence never changes.
	fence, err := chooseFence(g)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCompose, err)
	}

	var b strings.Builder
	b.WriteString(contextHeading + "\n")
	slots := append([]SlotState(nil), g.Slots...)
	for i := range slots {
		s := &slots[i]
		if s.Availability != AvailabilityDelivered {
			// Typed absence: existence + state only — no content, no
			// rationale, no behavioral guidance (Q-L2-3/4).
			fmt.Fprintf(&b, "\n[slot: %s | availability: %s]\n", s.Slot, s.Availability)
			continue
		}
		if s.Omitted {
			// State-only marker: some items were omitted for capacity;
			// counts and mechanics are trace-only (Q-L3-3).
			fmt.Fprintf(&b, "\n[slot: %s | availability: delivered | omitted_for_capacity]\n", s.Slot)
		} else {
			fmt.Fprintf(&b, "\n[slot: %s | availability: delivered]\n", s.Slot)
		}
		for _, it := range g.items[s.Slot] {
			version := ""
			if it.Version != "" {
				version = "version: " + it.Version + "\n"
			}
			fmt.Fprintf(&b, "%s--\nkind: %s\nsource: %s\nauthority: %s\nhash: %s\n%s%s--\n",
				fence, it.Kind, it.Provenance.Source, it.Authority, it.Hash, version, fence)
			b.Write(it.Evidence)
			if !bytes.HasSuffix(it.Evidence, []byte("\n")) {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "%s-end--\n", fence)
		}
	}
	// Composed-size cap: the rendered view including framing overhead
	// (security review HIGH-3 — evidence-sum caps alone allow
	// amplification through many tiny items).
	if b.Len() > MaxComposedBytes {
		return nil, fmt.Errorf("%w: %w: composed context is %d bytes (cap %d)", ErrCompose, ErrContextTooLarge, b.Len(), MaxComposedBytes)
	}

	userMsg := model.Message{Role: model.RoleUser, Content: b.String()}
	p := &Payload{
		Messages:     []model.Message{sysMsg, userMsg},
		ContractHash: g.Contract.Hash,
		EISHash:      set.Hash,
		RenderHash:   renderHash,
		Slots:        slots,
	}
	p.PayloadHash = payloadHash(p, g)
	return p, nil
}

// chooseFence derives the composition-wide fence deterministically
// from all delivered content: candidate i is a hash over every item
// hash plus the counter; the first candidate absent from every
// evidence body and metadata field wins. Exhaustion of all candidates
// (not constructible — see Compose) refuses with ErrFramingCollision.
func chooseFence(g *Gathered) (string, error) {
	var seed bytes.Buffer
	seed.WriteString("themis-fence-v1")
	for _, s := range g.Slots {
		for _, ref := range s.Items {
			seed.WriteString(ref.Hash)
		}
	}
	present := func(candidate string) bool {
		for _, s := range g.Slots {
			if strings.Contains(s.Slot, candidate) {
				return true
			}
			for _, it := range g.items[s.Slot] {
				if bytes.Contains(it.Evidence, []byte(candidate)) ||
					strings.Contains(it.Kind, candidate) ||
					strings.Contains(it.Provenance.Source, candidate) ||
					strings.Contains(it.Version, candidate) {
					return true
				}
			}
		}
		return false
	}
	return pickFence(seed.Bytes(), present)
}

const maxFenceCandidates = 32

func pickFence(seed []byte, present func(string) bool) (string, error) {
	for i := 0; i < maxFenceCandidates; i++ {
		sum := sha256.Sum256(append(seed, byte(i)))
		candidate := "--ctx-" + hex.EncodeToString(sum[:])[:12]
		if !present(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("%w: all fence candidates collide", ErrFramingCollision)
}

// payloadHash covers the canonical length-framed record — the
// mechanical reader's ground truth (L2-5.3/5.4): every field is
// length-prefixed, so no evidence byte can alter frame structure and
// reconstruction never interprets content.
func payloadHash(p *Payload, g *Gathered) string {
	var b bytes.Buffer
	b.WriteString("themis-ctx-v1\n")
	frame := func(s string) {
		fmt.Fprintf(&b, "%d:", len(s))
		b.WriteString(s)
	}
	frame(p.ContractHash)
	frame(p.EISHash)
	frame(p.RenderHash)
	for _, m := range p.Messages {
		frame(string(m.Role))
		frame(m.Content)
	}
	for _, s := range p.Slots {
		frame(s.Slot)
		frame(string(s.Availability))
		frame(string(s.SourceStatus))
		frame(string(s.Delivery))
		for _, ref := range s.Items {
			frame(ref.Kind)
			frame(ref.Source)
			frame(string(ref.Authority))
			frame(ref.Hash)
		}
	}
	sum := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(sum[:])
}
