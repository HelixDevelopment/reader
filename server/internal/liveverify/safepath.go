package liveverify

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// Protected-path semantics follow constitution 11.4.287(B) (scripts/lib/safe_rm.sh),
// restated in Go because the verifier deletes its own scratch directory:
//   - refuse empty, relative, the root, system directories, /home and its direct
//     children (user homes), $HOME and every ancestor of a home, protected direct
//     children of a home (the list below, from protected_paths.list), the temp root
//   - refuse fewer than 3 path components (a path strictly below the temp root
//     gets a 3-component bonus)
//   - the parent directory is resolved through symlinks; the last component is not.
//
// This port is STRICTER in one point: everything below the pure system trees
// (/usr, /etc, ...) is refused, not only the directory itself.
var (
	protectedExact = []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64", "/libx32", "/opt", "/proc", "/root",
		"/run", "/sbin", "/srv", "/sys", "/usr", "/var", "/mnt", "/media", "/snap", "/home", "/tmp", "/Users"}
	protectedTrees = []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64", "/libx32", "/proc", "/root", "/run", "/sbin", "/sys", "/usr", "/snap"}
	protectedHome  = []string{"scripts", "Projects", ".ssh", ".gnupg", ".claude*", ".config", ".local", "vasic", "ytdlp", "backups", ".cache"}
)

func isUnder(path, root string) bool {
	return path == root || strings.HasPrefix(path, strings.TrimSuffix(root, "/")+"/")
}

func resolveParent(p string) (string, error) {
	// Resolve through the deepest existing ancestor, then re-append the rest.
	rest := []string{}
	cur := p
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			for i := len(rest) - 1; i >= 0; i-- {
				r = filepath.Join(r, rest[i])
			}
			return r, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			return "", fmt.Errorf("cannot resolve %s", p)
		}
		rest = append(rest, filepath.Base(cur))
		cur = next
	}
}

func homeDirs() []string {
	var out []string
	add := func(h string) {
		if h == "" {
			return
		}
		out = append(out, filepath.Clean(h))
		if r, err := filepath.EvalSymlinks(h); err == nil {
			out = append(out, r)
		}
	}
	if h, err := os.UserHomeDir(); err == nil {
		add(h)
	}
	if u, err := user.Current(); err == nil {
		add(u.HomeDir)
	}
	for _, h := range strings.Split(os.Getenv("SAFE_RM_EXTRA_HOMES"), ":") {
		add(h)
	}
	return out
}

// checkRemovable returns nil only when dir may be deleted recursively.
func checkRemovable(dir string) error {
	refuse := func(why string) error { return fmt.Errorf("refusing to delete %q: %s", dir, why) }
	if dir == "" {
		return refuse("empty path")
	}
	clean := filepath.Clean(dir)
	if !filepath.IsAbs(clean) {
		return refuse("not absolute")
	}
	if clean == "/" {
		return refuse("filesystem root")
	}
	parent, err := resolveParent(filepath.Dir(clean))
	if err != nil {
		return refuse(err.Error())
	}
	cand := filepath.Join(parent, filepath.Base(clean))
	for _, c := range []string{clean, cand} {
		for _, p := range protectedExact {
			if c == p {
				return refuse("protected system path " + p)
			}
		}
		for _, t := range protectedTrees {
			if isUnder(c, t) {
				return refuse("inside protected system tree " + t)
			}
		}
		for _, h := range homeDirs() {
			if c == h {
				return refuse("home directory")
			}
			if isUnder(h, c) {
				return refuse("ancestor of a home directory")
			}
			if rel, ok := strings.CutPrefix(c, h+"/"); ok && !strings.Contains(rel, "/") {
				for _, pat := range protectedHome {
					if m, _ := filepath.Match(pat, rel); m {
						return refuse("protected home child ~/" + pat)
					}
				}
			}
		}
		for _, root := range []string{"/home", "/Users"} {
			if filepath.Dir(c) == root {
				return refuse("a user home directory")
			}
		}
	}
	tmp := os.Getenv("TMPDIR")
	if tmp == "" {
		tmp = "/tmp"
	}
	if r, err := filepath.EvalSymlinks(tmp); err == nil {
		tmp = r
	}
	if cand == tmp {
		return refuse("the temp root")
	}
	depth := len(strings.FieldsFunc(cand, func(r rune) bool { return r == '/' }))
	if isUnder(cand, tmp) && cand != tmp && !containsHome(tmp) {
		depth += 3
	}
	if depth < 3 {
		return refuse(fmt.Sprintf("only %d path components deep (minimum 3)", depth))
	}
	return nil
}

func containsHome(root string) bool {
	for _, h := range homeDirs() {
		if isUnder(h, root) {
			return true
		}
	}
	return false
}
