package context

// Confinement delegation: the canonical two-mode implementation lives
// in the leaf package confine (D-L5-4: one implementation, two
// explicit modes — never per-layer twins). These wrappers preserve
// the exported L2 contract; ErrConfinement is the same error value,
// so errors.Is holds across the delegation.

import "github.com/tofchaliss/themis/confine"

// ConfineCreatePath validates a mutation target under root
// (CreateMode). See confine.CreatePath for the locked rules.
func ConfineCreatePath(root, rel string) (string, error) {
	return confine.CreatePath(root, rel)
}
