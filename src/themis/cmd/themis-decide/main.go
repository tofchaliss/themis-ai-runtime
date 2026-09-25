// themis-decide is the ONLY mechanism that creates an Enterprise
// Position (D-T-7): a human runs it with a referencable execution
// tuple, a disposition from the closed vocabulary, and a rationale.
// It resolves the tuple through intake, renders the evidence, and
// appends the Position with an observed decision witness (D-T-8).
// Any decision.* argument is refused: identity is observed, never
// asserted.
//
// T-M1: the command exists and refuses; behaviour lands in T-M4.
package main

import (
	"fmt"
	"os"

	"github.com/tofchaliss/themis-app/intake"
)

func main() {
	_ = intake.Tuple{}
	fmt.Fprintln(os.Stderr, "themis-decide: not implemented — Themis v0 T-M4 (design.md §4)")
	os.Exit(2)
}
