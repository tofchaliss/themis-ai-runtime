package context

import (
	"fmt"
	"sort"

	"github.com/tofchaliss/themis/instructions"
)

// Assignment binds one registered source to one contract slot — the
// L7 plan's unit. Gather enforces Plan ⊆ Contract mechanically; it
// exercises no judgment about usefulness (design §3, Q-L2-1).
type Assignment struct {
	Slot   string
	Source Source
}

// ItemRef is the trace reference for one delivered item.
type ItemRef struct {
	Slot        string
	Kind        string
	Source      string
	Producer    string
	Author      string
	Origin      string
	Authority   AuthorityClass
	Sensitivity Sensitivity
	Version     string
	Hash        string
	Size        int // evidence byte length — envelope metadata, a legal rank dimension
	Mechanism   Mechanism
}

// SlotState is the typed per-slot outcome: availability for the
// model, source/delivery statuses for the trace.
type SlotState struct {
	Slot         string
	Kind         string
	Requirement  SlotRequirement
	Availability Availability
	SourceStatus SourceStatus
	Delivery     DeliveryStatus
	// Omitted marks a delivered slot that lost items to capacity —
	// model-visible as a state flag only; counts stay in the trace
	// (Q-L3-3 minimum disclosure).
	Omitted bool
	Items   []ItemRef
}

// Gathered is the validated, classified evidence set awaiting
// composition. Constructible through Gather (and, as a managed
// successor, through Manage): the
// unexported validated flag means an externally assembled Gathered
// carries no Plan ⊆ Contract guarantee and Compose refuses it.
type Gathered struct {
	Contract  *Contract
	Slots     []SlotState
	items     map[string][]ContextItem // slot -> delivered items
	validated bool
	managed   bool // set by Manage; a managed set cannot be re-managed
}

// Gather validates the plan against the contract and collects every
// assigned source. Fail closed on: unknown slot, assignment to a
// withheld slot, class or sensitivity outside the slot/ceiling,
// unrecognized source, confinement escape, caps, kind mismatch,
// uncovered non-withheld slot, and any required slot that cannot
// deliver. A retrieval failure on an optional slot becomes typed
// unavailability, never silence. Inline (task-payload) failures are
// intake-classified via instructions.ErrIntake.
func Gather(contract *Contract, assignments []Assignment) (*Gathered, error) {
	if contract == nil || contract.Hash == "" {
		return nil, fmt.Errorf("%w: no validated contract — composition without the contract ceiling does not happen", ErrContractInvalid)
	}
	wrap := func(src Source, err error) error {
		if err != nil && src.Kind == KindInline {
			return fmt.Errorf("%w: %w", instructions.ErrIntake, err)
		}
		return err
	}
	assigned := map[string]bool{}
	g := &Gathered{Contract: contract, items: map[string][]ContextItem{}}
	states := map[string]*SlotState{}

	for _, a := range assignments {
		slot := contract.slot(a.Slot)
		if slot == nil {
			return nil, fmt.Errorf("%w: no slot %q in contract %s", ErrPlanOutsideContract, a.Slot, contract.Workflow)
		}
		if slot.Withhold {
			return nil, fmt.Errorf("%w: slot %q is withheld by the contract; the plan must not gather it", ErrPlanOutsideContract, a.Slot)
		}
		if assigned[a.Slot] {
			return nil, fmt.Errorf("%w: slot %q assigned twice", ErrPlanOutsideContract, a.Slot)
		}
		assigned[a.Slot] = true
		if err := checkSource(a.Source); err != nil {
			return nil, wrap(a.Source, err)
		}
		if !slot.classPermitted(a.Source.Authority) {
			return nil, fmt.Errorf("%w: slot %q does not permit class %q", ErrPlanOutsideContract, a.Slot, a.Source.Authority)
		}
		if sensitivityRank[a.Source.Sensitivity] > sensitivityRank[contract.SensitivityCeiling] {
			return nil, fmt.Errorf("%w: source %s (%s) exceeds ceiling %s", ErrSensitivityCeiling, a.Source.Name, a.Source.Sensitivity, contract.SensitivityCeiling)
		}

		items, available, err := a.Source.collect()
		if err != nil {
			return nil, wrap(a.Source, err)
		}
		state := &SlotState{Slot: slot.Name, Kind: slot.Kind, Requirement: slot.Requirement, SourceStatus: SourceAvailable}
		if !available || len(items) == 0 {
			state.SourceStatus = SourceUnavailable
			state.Availability = AvailabilityUnavailable
			state.Delivery = DeliveryNone
			if slot.Requirement == SlotRequired {
				return nil, fmt.Errorf("%w: slot %q source %s unavailable", ErrRequiredMissing, slot.Name, a.Source.Name)
			}
			states[slot.Name] = state
			continue
		}
		for _, it := range items {
			if !kindMatches(slot.Kind, it.Kind) {
				return nil, fmt.Errorf("%w: item kind %q does not fill slot %q (kind %q)", ErrPlanOutsideContract, it.Kind, slot.Name, slot.Kind)
			}
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Kind != items[j].Kind {
				return items[i].Kind < items[j].Kind
			}
			return items[i].Hash < items[j].Hash
		})
		state.Availability = AvailabilityDelivered
		state.Delivery = DeliveryDelivered
		for _, it := range items {
			state.Items = append(state.Items, ItemRef{
				Slot: slot.Name, Kind: it.Kind, Source: it.Provenance.Source,
				Producer: it.Producer, Author: it.Provenance.Author, Origin: it.Provenance.Origin,
				Authority: it.Authority, Sensitivity: it.Sensitivity, Version: it.Version,
				Hash: it.Hash, Size: len(it.Evidence), Mechanism: MechanismPlannedConnector,
			})
		}
		g.items[slot.Name] = items
		states[slot.Name] = state
	}

	total, count := 0, 0
	for _, items := range g.items {
		for _, it := range items {
			total += len(it.Evidence)
			count++
		}
	}
	if total > MaxContextBytes {
		return nil, fmt.Errorf("%w: %d bytes (cap %d)", ErrContextTooLarge, total, MaxContextBytes)
	}
	if count > MaxContextItems {
		return nil, fmt.Errorf("%w: %d items (cap %d)", ErrContextTooLarge, count, MaxContextItems)
	}

	// Contract-order slot states; every non-withheld slot must be
	// covered by the plan.
	for _, slot := range contract.Slots {
		if slot.Withhold {
			// SourceStatus here is convention, not observation: L2
			// never queries a withheld slot's source; existence is
			// implied by withheld_by_contract itself.
			g.Slots = append(g.Slots, SlotState{
				Slot: slot.Name, Kind: slot.Kind, Requirement: slot.Requirement,
				Availability: AvailabilityWithheld, SourceStatus: SourceAvailable, Delivery: DeliveryWithheld,
			})
			continue
		}
		state, ok := states[slot.Name]
		if !ok {
			return nil, fmt.Errorf("%w: slot %q has no assignment — the plan must cover every non-withheld slot", ErrPlanOutsideContract, slot.Name)
		}
		g.Slots = append(g.Slots, *state)
	}
	g.validated = true
	return g, nil
}
