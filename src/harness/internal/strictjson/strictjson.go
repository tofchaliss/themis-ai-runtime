// Package strictjson is the key wall for authority-bearing JSON
// artifacts (grants, specs, and the grant an envelope submits).
//
// encoding/json is lenient in two ways that let reviewed text and
// decoded authority disagree: duplicate keys are last-value-wins, and
// field matching is case-insensitive — DisallowUnknownFields accepts
// "Workspace" as the workspace field. A guard that inspects the raw
// document by exact key (as L7's assembly does for the @workspace
// placeholder) therefore never sees what the loader binds. The wall is
// applied to the bytes BEFORE any decode: every object key must be
// exactly lowercase snake_case, and no object may state a key twice.
// (Security review of dc8034c, CRITICAL-1 and LOW-2.)
package strictjson

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var exactKey = regexp.MustCompile(`^[a-z0-9_]+$`)

// Check refuses duplicate keys and keys that are not exact lowercase
// snake_case. It reads the token stream only; it decodes nothing and
// authorizes nothing.
func Check(raw []byte) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	type frame struct {
		object    bool
		keys      map[string]bool
		nextIsKey bool
	}
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.nextIsKey {
				if key, ok := tok.(string); ok {
					if !exactKey.MatchString(key) {
						return fmt.Errorf("key %q is not an exact lowercase key — a case variant would bind through the decoder but escape every exact-key guard", key)
					}
					if top.keys[key] {
						return fmt.Errorf("duplicate key %q — the decoder keeps the last value, the reader sees the first", key)
					}
					top.keys[key] = true
					top.nextIsKey = false
					continue
				}
			}
		}
		switch d := tok.(type) {
		case json.Delim:
			switch d {
			case '{':
				stack = append(stack, &frame{object: true, keys: map[string]bool{}, nextIsKey: true})
			case '[':
				stack = append(stack, &frame{})
			case '}', ']':
				stack = stack[:len(stack)-1]
				if len(stack) > 0 && stack[len(stack)-1].object {
					stack[len(stack)-1].nextIsKey = true
				}
			}
			continue
		}
		if len(stack) > 0 && stack[len(stack)-1].object {
			stack[len(stack)-1].nextIsKey = true
		}
	}
}
