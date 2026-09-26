// themis-recheck re-establishes, for EVERY task already recorded under a
// deployment's state root, what deployment governed it — from the record
// and the Governance registry alone, with the binary running now.
//
// Phase D's D4/D5/D6 do this for a task the harness just created. That
// proves the record is re-establishable at the moment it is written. It
// does not prove the property that actually matters operationally:
//
//	a past execution stays interpretable after the binary changes.
//
// That is the G1 claim applied over time — "the record identifies the
// anchor; the registry and the bytes prove what that identity meant" —
// and the only honest way to test it is to change the binary and ask
// the old records again. This program is the asking.
//
// It is READ-ONLY and holds no authority: it opens the state root, reads
// events, resolves objects, and calls the same VerifyAnchorRecord any
// caller would. It creates no task, writes no object, and registers
// nothing. Like the rest of evidence/harness, it is evidence tooling —
// a disagreement here is a finding, never a repair.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tofchaliss/themis-ai-runtime/src/harness/deployment"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/ratchet"
	"github.com/tofchaliss/themis-ai-runtime/src/harness/state"
)

func main() {
	deploy := flag.String("deploy", "", "deployment root (absolute)")
	registry := flag.String("anchors-registry", "", "Governance-active anchors registry (absolute)")
	expect := flag.String("anchor-sha256", "", "optional: the identity every task is expected to carry")
	flag.Parse()
	if *deploy == "" || *registry == "" {
		bail("-deploy and -anchors-registry are required")
	}

	stateRoot := filepath.Join(*deploy, "state")
	root, err := state.OpenRoot(stateRoot)
	if err != nil {
		bail("open state root %s: %v", stateRoot, err)
	}
	ids, err := taskIDs(stateRoot)
	if err != nil {
		bail("enumerate tasks: %v", err)
	}
	if len(ids) == 0 {
		bail("no task records under %s — nothing to re-establish", stateRoot)
	}
	fmt.Printf("state root : %s\n", stateRoot)
	fmt.Printf("registry   : %s\n", *registry)
	fmt.Printf("tasks      : %d\n\n", len(ids))

	var reestablished, other, unanchored, failed int
	for _, id := range ids {
		verdict, detail := recheck(root, *registry, *expect, id)
		switch verdict {
		case "OK":
			reestablished++
			fmt.Printf("  \033[32mOK\033[0m        %-24s %s\n", id, detail)
		case "OTHER":
			// Re-established, just not the deployment asked about.
			other++
			fmt.Printf("  \033[33mOTHER\033[0m     %-24s %s\n", id, detail)
		case "UNANCHORED":
			unanchored++
			fmt.Printf("  UNANCHORED %-24s %s\n", id, detail)
		default:
			failed++
			fmt.Printf("  \033[31mFAILED\033[0m    %-24s %s\n", id, detail)
		}
	}

	fmt.Printf("\n=== RESULT ===\n")
	fmt.Printf("  re-established : %d\n", reestablished+other)
	if *expect != "" {
		fmt.Printf("    of which under %s : %d\n", short(*expect), reestablished)
		fmt.Printf("    under another deployment      : %d\n", other)
	}
	fmt.Printf("  unanchored     : %d (declared so in the record — not a failure)\n", unanchored)
	fmt.Printf("  FAILED         : %d\n", failed)

	// Durability is the question this program asks, and only a genuine
	// D5/D6/L6 failure answers it badly. Report that separately from
	// "these tasks belong to a different deployment", which is what a
	// superseded deployment's record plane is SUPPOSED to look like.
	if failed > 0 {
		fmt.Println("\nA past execution is no longer interpretable from its own record.")
		fmt.Println("That is a finding about record durability across this binary, not")
		fmt.Println("something to repair here.")
		os.Exit(1)
	}
	fmt.Println("\nEvery anchored task still re-establishes its deployment from the")
	fmt.Println("record and the registry alone, under the binary running now.")
	if other > 0 {
		fmt.Printf("\n%d re-established under a deployment other than %s. Each one was\n", other, short(*expect))
		fmt.Println("fully re-established — identity, durable bytes, registry — so this")
		fmt.Println("is not a durability finding. A withdrawn anchor still explains the")
		fmt.Println("executions it governed; that is what append-only registration buys.")
		os.Exit(3)
	}
}

// recheck performs D4/D5/D6 over one already-recorded task, plus the L6
// record verdict. Each step is reported by name so a failure says which
// link broke rather than that something did.
func recheck(root *state.Root, registry, expect, id string) (string, string) {
	events, err := root.ReadEvents(id)
	if err != nil {
		return "FAILED", fmt.Sprintf("read events: %v", err)
	}
	// D4: the identity the record carries.
	recorded := ""
	for _, e := range events {
		var body struct {
			GovernedHashes map[string]string `json:"governed_hashes"`
		}
		if json.Unmarshal(e.Body, &body) == nil && body.GovernedHashes != nil {
			if h, ok := body.GovernedHashes["deployment_anchor"]; ok {
				recorded = h
			}
		}
	}
	switch {
	case recorded == "":
		return "FAILED", "D4: record carries no deployment identity"
	case recorded == "unanchored":
		// Explicit, and distinguishable from a dropped key by design
		// (close-review MEDIUM-1). Not a failure — a fact.
		return "UNANCHORED", "D4: record declares an unanchored run"
	}
	// An identity that is not the operator's expectation is NOT a
	// durability failure. Say so only after D5/D6 have run: the honest
	// report is "re-established, and it belongs to another deployment",
	// which is the normal state of a superseded deployment's record
	// plane. Conflating the two would report a working append-only
	// registry as a broken one.
	mismatch := expect != "" && recorded != expect
	// D5: the anchor BYTES, durable in the record, addressed by their
	// own hash. Recording the identity is not enough — the bytes are
	// what make the identity interpretable.
	var anchorBytes []byte
	for _, e := range events {
		for _, ref := range e.Refs {
			b, gerr := root.Store().GetObject(ref.ID)
			if gerr == nil && ratchet.InstanceID(b) == recorded {
				anchorBytes = b
			}
		}
	}
	if anchorBytes == nil {
		return "FAILED", "D5: anchor bytes are not durable in the record"
	}
	// D6: the registry says what that identity MEANT. A withdrawn
	// anchor still explains a past execution.
	a, err := deployment.VerifyAnchorRecord(recorded, anchorBytes, registry)
	if err != nil {
		return "FAILED", fmt.Sprintf("D6: %v", err)
	}
	// The L6 record must also still verify on its own terms.
	v := root.Verify(id)
	if v.Verdict != state.VerdictVerified {
		return "FAILED", fmt.Sprintf("L6 verdict %s (%s)", v.Verdict, v.Detail)
	}
	man, err := root.ReadManifest(id)
	if err != nil {
		return "FAILED", fmt.Sprintf("read manifest: %v", err)
	}
	detail := fmt.Sprintf("%s@%d  %s  %s  events=%d",
		a.Name, a.Deployment, short(recorded), man.Status, len(events))
	if mismatch {
		return "OTHER", detail + fmt.Sprintf("  (not the expected %s)", short(expect))
	}
	return "OK", detail
}

func taskIDs(stateRoot string) ([]string, error) {
	tasks := filepath.Join(stateRoot, "tasks")
	es, err := os.ReadDir(tasks)
	if err != nil {
		// Older layouts keep task dirs directly under the state root.
		es, err = os.ReadDir(stateRoot)
		if err != nil {
			return nil, err
		}
	}
	var out []string
	for _, e := range es {
		if !e.IsDir() {
			continue
		}
		if _, serr := os.Stat(filepath.Join(tasks, e.Name(), "manifest.json")); serr == nil {
			out = append(out, e.Name())
			continue
		}
		if _, serr := os.Stat(filepath.Join(stateRoot, e.Name(), "manifest.json")); serr == nil {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

func short(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func bail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\nthemis-recheck: "+format+"\n", args...)
	os.Exit(2)
}
