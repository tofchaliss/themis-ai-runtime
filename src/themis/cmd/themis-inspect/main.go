// themis-inspect is read-only: it lists a Finding's Positions, derives
// the current one (highest sequence), and re-verifies a Position
// against the harness record (D-T-7, Register C). It creates nothing.
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
	fmt.Fprintln(os.Stderr, "themis-inspect: not implemented — Themis v0 T-M4 (design.md §4)")
	os.Exit(2)
}
