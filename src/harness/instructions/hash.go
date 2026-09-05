package instructions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// canonicalVersion prefixes the serialization so any future change to
// the canonical form changes every hash loudly instead of colliding
// silently.
const canonicalVersion = "themis-eis-v1\n"

// Injectivity note: the separators \x1f/\x1e cannot appear in any
// field because validate() admits only idSyntax ids, fixed category
// constants, recognized scopes (String() never hits its fallback for
// admitted instructions), and fixed-length hex body hashes. validate()
// is the enforcer of this invariant — never feed unvalidated
// instructions to eisHash.
//
// canonical serializes the resolved, ordered instruction set for
// hashing. The EIS hash identifies the delivered instruction
// environment: id, scope, category, protection, and content revision
// of every admitted instruction, in resolved order. Conflicts,
// exemptions, and source paths are trace data recorded alongside —
// they do not alter what the model receives, so they do not alter the
// version.
func canonical(instructions []Instruction) []byte {
	var b bytes.Buffer
	b.WriteString(canonicalVersion)
	for _, inst := range instructions {
		fmt.Fprintf(&b, "%s\x1f%s\x1f%s\x1f%t\x1f%s\x1e",
			inst.ID, inst.Scope, inst.Category, inst.Protected, inst.BodyHash)
	}
	return b.Bytes()
}

func eisHash(instructions []Instruction) string {
	sum := sha256.Sum256(canonical(instructions))
	return hex.EncodeToString(sum[:])
}
