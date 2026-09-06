package context

// CreateMode confinement (L5, D-L5-4 / Q-L5-4): the mutation-side
// twin of ConfinePath. One canonical confinement implementation, two
// explicit modes — ResolveMode (reads, confinedPath) and CreateMode
// (mutations, this file) — sharing lexical normalization,
// absolute/".." rejection, root canonicalization, containment, the
// VCS deny-list, and error construction; differing only where
// existence semantics genuinely differ. The shipped ResolveMode
// nonexistent-target lexical fallback is demonstrably escapable for
// writes (symlinked parent + nonexistent target ⇒ out-of-workspace
// write), which is why CreateMode exists.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// vcsDenied reports whether any path component is named or prefixed
// ".git" — deliberately broad (Q-L5-4.3): .git, .gitignore, .github,
// .gitmodules all refuse. Classifying which .git* files are "safe"
// would itself become a security-maintenance surface. Applies to
// every mutation mechanism: write, delete, rename-from, rename-to,
// patch.
func vcsDenied(rel string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(filepath.Clean(rel)), "/") {
		if strings.HasPrefix(seg, ".git") {
			return true
		}
	}
	return false
}

// ConfineCreatePath validates a mutation target under root and
// returns the absolute path to create/overwrite. CreateMode rules
// (all locked, Q-L5-4):
//   - absolute paths, "", ".", and lexical escapes refuse (shared
//     with ResolveMode);
//   - any ".git*" component refuses;
//   - the parent chain must exist, fully resolved, with no symlink
//     anywhere — even one resolving inside the workspace (symlinks
//     are read-legitimate, write-refused);
//   - the final component must not be a symlink or a directory; it
//     may not exist (creation) or be a regular file (overwrite).
//
// TOCTOU between this check and the mutation is a documented
// threat-model limitation against the defined non-concurrent local
// attacker — never a claim of kernel-enforced resolution (Q-L5-4.5).
func ConfineCreatePath(root, rel string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("%w: mutation has no confinement root", ErrConfinement)
	}
	if filepath.IsAbs(rel) || rel == "" {
		return "", fmt.Errorf("%w: %q", ErrConfinement, rel)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrConfinement, err)
	}
	joined := filepath.Join(rootAbs, rel)
	if joined == rootAbs {
		return "", fmt.Errorf("%w: mutation target cannot be the confinement root", ErrConfinement)
	}
	if !strings.HasPrefix(joined, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q", ErrConfinement, rel)
	}
	if vcsDenied(rel) {
		return "", fmt.Errorf("%w: %q: .git* is never a mutation target", ErrConfinement, rel)
	}
	// Root must exist and canonicalize: mutations against a phantom
	// root fail closed.
	rootResolved, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrConfinement, err)
	}
	// Walk the parent chain component-by-component: every ancestor
	// must exist, be a real directory, and not be a symlink.
	relClean := filepath.Clean(rel)
	segs := strings.Split(relClean, string(filepath.Separator))
	cur := rootResolved
	for _, seg := range segs[:len(segs)-1] {
		cur = filepath.Join(cur, seg)
		info, err := os.Lstat(cur)
		if err != nil {
			return "", fmt.Errorf("%w: %q: parent chain must exist for a mutation target", ErrConfinement, rel)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%w: %q: symlink in write path", ErrConfinement, rel)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%w: %q: parent component is not a directory", ErrConfinement, rel)
		}
	}
	final := filepath.Join(cur, segs[len(segs)-1])
	info, err := os.Lstat(final)
	switch {
	case err != nil && os.IsNotExist(err):
		// Creation of a new file under a verified parent chain.
	case err != nil:
		return "", fmt.Errorf("%w: %v", ErrConfinement, err)
	case info.Mode()&os.ModeSymlink != 0:
		return "", fmt.Errorf("%w: %q: mutation target is a symlink", ErrConfinement, rel)
	case info.IsDir():
		return "", fmt.Errorf("%w: %q: mutation target is a directory", ErrConfinement, rel)
	case !info.Mode().IsRegular():
		return "", fmt.Errorf("%w: %q: mutation target is not a regular file", ErrConfinement, rel)
	}
	return final, nil
}
