package context

import (
	"fmt"
	"sort"
)

// Manage is Layer 3: deterministic policy execution between Gather
// and Compose. L3 controls capacity, not authority (design §3,
// Q-L3-2): a context item leaves the model's view because the
// contract explicitly permits its exclusion, never merely because
// capacity became insufficient. Zero runtime judgment: every drop
// decision is computed from ItemRef metadata under the declared
// policy — the ranking never sees evidence bytes.

// DropAction records one L3 decision in the selection trace.
type DropAction string

const (
	ActionKept      DropAction = "kept"
	ActionDropped   DropAction = "dropped_by_budget"
	ActionCollapsed DropAction = "collapsed_duplicate"
)

type Decision struct {
	Slot   string
	Hash   string
	Action DropAction
	Rank   int    // position in the within-slot drop ranking (0 = safest)
	Reason string // deterministic, descriptive
}

// CollapseRef records provenance multiplicity retained through dedup:
// dedup removes delivery redundancy, never provenance (Q-L3-8).
type CollapsedItem struct {
	Source  string
	Kind    string
	Version string
}

type CollapseRef struct {
	Slot      string
	KeptHash  string
	Authority AuthorityClass
	// Collapsed retains every supplier's identity — dedup removes
	// delivery redundancy, never provenance multiplicity, including
	// the losing items' Kind/Version (security review LOW-2).
	Collapsed []CollapsedItem
}

// SelectionTrace is L3's contribution to the delivery trace (L6
// shape).
type SelectionTrace struct {
	PolicyHash  string
	Estimator   string
	BudgetLimit int
	BudgetUsed  int
	Decisions   []Decision
	Collapses   []CollapseRef
}

// Manage applies the policy to a Gather-validated set and returns a
// new validated set for Compose plus the selection trace. No
// pressure ⇒ byte-identical passthrough of every delivered item.
// Pressure ⇒ drops strictly in policy order over contract-droppable
// slots; if the surviving set still exceeds budget, fail closed
// (ErrBudget) — omitted_for_capacity never becomes silent
// degradation.
func Manage(policy *ManagementPolicy, g *Gathered) (*Gathered, *SelectionTrace, error) {
	if policy == nil || policy.Hash == "" {
		return nil, nil, fmt.Errorf("%w: no validated management policy", ErrPolicyInvalid)
	}
	if g == nil || !g.validated {
		return nil, nil, fmt.Errorf("%w: gathered context did not pass Gather validation", ErrPolicyInvalid)
	}
	// Drop-order slots must be contract-droppable: optional AND
	// explicitly authorized. A policy naming a required or undroppable
	// slot is invalid against this contract — capacity is never
	// authority.
	droppable := map[string]bool{}
	for _, name := range policy.DropOrder {
		slot := g.Contract.slot(name)
		if slot == nil {
			return nil, nil, fmt.Errorf("%w: drop-order slot %q not in contract %s", ErrPolicyInvalid, name, g.Contract.Workflow)
		}
		if slot.Requirement != SlotOptional || !slot.Droppable || slot.Withhold {
			return nil, nil, fmt.Errorf("%w: drop-order slot %q is not contract-droppable (requirement=%s droppable=%t withhold=%t)",
				ErrPolicyInvalid, name, slot.Requirement, slot.Droppable, slot.Withhold)
		}
		droppable[name] = true
	}

	trace := &SelectionTrace{PolicyHash: policy.Hash, Estimator: estimatorVersion, BudgetLimit: policy.Budget}
	managed := &Gathered{Contract: g.Contract, items: map[string][]ContextItem{}, validated: true}

	// Working copy + within-class dedup (delivery redundancy removed,
	// provenance multiplicity retained in the trace).
	work := map[string][]ContextItem{}
	for _, s := range g.Slots {
		items := append([]ContextItem(nil), g.items[s.Slot]...)
		if policy.Dedup == "within-class" && len(items) > 1 {
			type key struct {
				class AuthorityClass
				hash  string
			}
			first := map[key]int{}
			var kept []ContextItem
			collapses := map[key]*CollapseRef{}
			for _, it := range items {
				k := key{it.Authority, it.Hash}
				if idx, dup := first[k]; dup {
					ref := collapses[k]
					if ref == nil {
						keeper := kept[idx]
						ref = &CollapseRef{Slot: s.Slot, KeptHash: it.Hash, Authority: it.Authority,
							Collapsed: []CollapsedItem{{Source: keeper.Provenance.Source, Kind: keeper.Kind, Version: keeper.Version}}}
						collapses[k] = ref
					}
					ref.Collapsed = append(ref.Collapsed, CollapsedItem{Source: it.Provenance.Source, Kind: it.Kind, Version: it.Version})
					trace.Decisions = append(trace.Decisions, Decision{Slot: s.Slot, Hash: it.Hash,
						Action: ActionCollapsed, Reason: "byte-identical within authority class"})
					continue
				}
				first[k] = len(kept)
				kept = append(kept, it)
			}
			for _, ref := range collapses {
				sort.Slice(ref.Collapsed, func(i, j int) bool {
					a, b := ref.Collapsed[i], ref.Collapsed[j]
					if a.Source != b.Source {
						return a.Source < b.Source
					}
					return a.Kind < b.Kind
				})
				trace.Collapses = append(trace.Collapses, *ref)
			}
			items = kept
		}
		work[s.Slot] = items
	}
	sort.SliceStable(trace.Collapses, func(i, j int) bool {
		a, b := trace.Collapses[i], trace.Collapses[j]
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		if a.KeptHash != b.KeptHash {
			return a.KeptHash < b.KeptHash
		}
		return a.Authority < b.Authority
	})

	// Post-dedup refs, index-aligned with work items: budget, ranking,
	// and drop accounting all operate on this one set so phantom drops
	// cannot exist (security review HIGH-1).
	postRefs := map[string][]ItemRef{}
	used := 0
	for slotName, items := range work {
		for _, it := range items {
			postRefs[slotName] = append(postRefs[slotName], ItemRef{
				Slot: slotName, Kind: it.Kind, Source: it.Provenance.Source,
				Producer: it.Producer, Author: it.Provenance.Author, Origin: it.Provenance.Origin,
				Authority: it.Authority, Sensitivity: it.Sensitivity, Version: it.Version,
				Hash: it.Hash, Size: len(it.Evidence), Mechanism: MechanismPlannedConnector,
			})
			used += estimateTokens(len(it.Evidence))
		}
	}

	// Pressure: drop in declared order, worst-ranked items first
	// within each slot, until the budget is met or candidates are
	// exhausted.
	// Drops are marked by post-dedup index — bijective with the work
	// items, so refs and delivered items cannot diverge.
	droppedIdx := map[string]map[int]bool{} // slot -> work index -> dropped
	for _, slotName := range policy.DropOrder {
		if used <= policy.Budget {
			break
		}
		refs := postRefs[slotName]
		if len(refs) == 0 {
			continue
		}
		ranked := rankForDrop(refs, policy.RankKeys)
		if droppedIdx[slotName] == nil {
			droppedIdx[slotName] = map[int]bool{}
		}
		for i := len(ranked) - 1; i >= 0 && used > policy.Budget; i-- {
			droppedIdx[slotName][ranked[i].idx] = true
			used -= estimateTokens(ranked[i].ref.Size)
			trace.Decisions = append(trace.Decisions, Decision{Slot: slotName, Hash: ranked[i].ref.Hash,
				Action: ActionDropped, Rank: i, Reason: "budget pressure, policy drop order"})
		}
	}
	if used > policy.Budget {
		// Everything the contract permits dropping is gone and the
		// undroppable remainder still exceeds the budget: fail closed.
		return nil, nil, fmt.Errorf("%w: %d tokens remain against budget %d after exhausting the drop order",
			ErrBudget, used, policy.Budget)
	}
	trace.BudgetUsed = used

	// Assemble managed slot states with typed omission (Q-L3-3):
	// model-visible state only; counts and mechanics stay in the
	// trace.
	for _, s := range g.Slots {
		state := s
		var keptItems []ContextItem
		var keptRefs []ItemRef
		anyDropped := len(droppedIdx[s.Slot]) > 0
		for i, it := range work[s.Slot] {
			if droppedIdx[s.Slot][i] {
				continue
			}
			keptItems = append(keptItems, it)
			keptRefs = append(keptRefs, postRefs[s.Slot][i])
			trace.Decisions = append(trace.Decisions, Decision{Slot: s.Slot, Hash: it.Hash, Action: ActionKept})
		}
		if s.Availability == AvailabilityDelivered {
			if len(keptItems) == 0 {
				state.Availability = AvailabilityOmittedCapacity
				state.Delivery = DeliveryOmittedCapacity
				state.Items = nil
				state.Omitted = false
			} else {
				state.Omitted = anyDropped
				// Refs are rebuilt from the kept items themselves —
				// bijective with delivery by construction (HIGH-1).
				state.Items = keptRefs
				managed.items[s.Slot] = keptItems
			}
		}
		managed.Slots = append(managed.Slots, state)
	}
	sort.SliceStable(trace.Decisions, func(i, j int) bool {
		a, b := trace.Decisions[i], trace.Decisions[j]
		if a.Slot != b.Slot {
			return a.Slot < b.Slot
		}
		if a.Action != b.Action {
			return a.Action < b.Action
		}
		return a.Hash < b.Hash
	})
	return managed, trace, nil
}

// rankedRef pairs a post-dedup ref with its work-index so drop
// decisions stay bijective with delivered items.
type rankedRef struct {
	ref ItemRef
	idx int
}

// rankForDrop orders a slot's post-dedup refs safest-first by the
// declared rank keys, hash as the terminal tiebreak. It reads ItemRef
// metadata only — the Q-L3-1 type-system boundary: evidence bytes are
// not an input to ranking.
func rankForDrop(input []ItemRef, keys []string) []rankedRef {
	refs := make([]rankedRef, len(input))
	for i, r := range input {
		refs[i] = rankedRef{ref: r, idx: i}
	}
	sort.SliceStable(refs, func(i, j int) bool {
		for _, k := range keys {
			switch k {
			case "size_asc":
				if refs[i].ref.Size != refs[j].ref.Size {
					return refs[i].ref.Size < refs[j].ref.Size
				}
			case "size_desc":
				if refs[i].ref.Size != refs[j].ref.Size {
					return refs[i].ref.Size > refs[j].ref.Size
				}
			case "kind":
				if refs[i].ref.Kind != refs[j].ref.Kind {
					return refs[i].ref.Kind < refs[j].ref.Kind
				}
			case "version":
				if refs[i].ref.Version != refs[j].ref.Version {
					return refs[i].ref.Version > refs[j].ref.Version // newer survives longer
				}
			}
		}
		if refs[i].ref.Hash != refs[j].ref.Hash {
			return refs[i].ref.Hash < refs[j].ref.Hash
		}
		return refs[i].idx < refs[j].idx
	})
	return refs
}
