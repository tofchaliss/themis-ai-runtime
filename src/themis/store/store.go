// Package store is Themis's governed, anchor-pinned, READ-ONLY store:
// the Findings and Products registries (D-T-9), and the L4 read seam
// that serves them to the harness under the capability's registered
// class. This package has no filesystem writer of any kind — a Themis
// read-store immutability wall, pinned by test. It carries no
// authority class field: the L4 registry's `trust` on get_finding /
// get_product mints `governed-record`; the store only supplies the
// governed bytes.
//
// T-M1 establishes the package and its dependency boundary only; the
// loaders and Read land in T-M2 (design.md §4, Gate 0).
package store

// Seam is the shape the harness's L4 executor table consumes
// (tools.ThemisSeam): Read(kind, id) → exact record bytes. It is
// restated here, not imported, so this package depends on no harness
// execution package; cmd/themis-run asserts the interface match.
type Seam interface {
	Read(kind, id string) ([]byte, error)
}
