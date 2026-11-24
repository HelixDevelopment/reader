package mutation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Constitution 11.4.287: removeScratch may delete only a scratch directory
// this tool created (a descendant of ".helix-verify" carrying the marker, or an
// empty one). Every victim below is a REAL directory on disk; the test asserts
// it survives.

func mkVictim(t *testing.T, rel string) string {
	t.Helper()
	d := filepath.Join(t.TempDir(), rel)
	if err := os.MkdirAll(filepath.Join(d, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "sub", "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return d
}

func survives(t *testing.T, d string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(d, "sub", "keep.txt")); err != nil {
		t.Fatalf("victim %s was deleted: %v", d, err)
	}
}

func TestRemoveScratchRefusesNonScratch(t *testing.T) {
	deep := mkVictim(t, "a/b/Projects/helix") // depth >= 3, not under .helix-verify
	if err := removeScratch(deep); err == nil {
		t.Fatalf("removeScratch accepted a non-scratch directory")
	}
	survives(t, deep)

	// a perfectly marked directory that is NOT under .helix-verify is still refused
	marked := mkVictim(t, "x/y/Projects/marked")
	if err := os.WriteFile(filepath.Join(marked, scratchMarker), []byte(scratchMagic), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeScratch(marked); err == nil {
		t.Fatalf("removeScratch accepted a marked directory outside .helix-verify")
	}
	survives(t, marked)

	// under .helix-verify but WITHOUT the marker and non-empty: not ours
	unmarked := mkVictim(t, ".helix-verify/other")
	if err := removeScratch(unmarked); err == nil {
		t.Fatalf("removeScratch accepted an unmarked non-empty directory under .helix-verify")
	}
	survives(t, unmarked)

	// .helix-verify itself is not a descendant
	hv := mkVictim(t, ".helix-verify")
	if err := removeScratch(hv); err == nil {
		t.Fatalf("removeScratch accepted .helix-verify itself")
	}
	survives(t, hv)

	for _, p := range []string{"", "/", "/home", "/usr", "/tmp", os.Getenv("HOME"), filepath.Dir(os.Getenv("HOME")), "relative/path"} {
		if err := removeScratch(p); err == nil {
			t.Fatalf("removeScratch accepted %q", p)
		}
	}
}

func TestRemoveScratchRefusesSymlink(t *testing.T) {
	victim := mkVictim(t, "real/Projects")
	base := filepath.Join(t.TempDir(), ".helix-verify")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "lnk")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}
	if err := removeScratch(link); err == nil {
		t.Fatalf("removeScratch followed a symlink")
	}
	survives(t, victim)
	// a symlinked parent that resolves outside .helix-verify
	outer := filepath.Join(t.TempDir(), ".helix-verify")
	if err := os.Symlink(filepath.Dir(victim), outer); err != nil {
		t.Fatal(err)
	}
	if err := removeScratch(filepath.Join(outer, "Projects")); err == nil {
		t.Fatalf("removeScratch accepted a path whose resolved parent is not under .helix-verify")
	}
	survives(t, victim)
}

func TestRemoveScratchAcceptsOwnMarkedScratch(t *testing.T) {
	d := filepath.Join(t.TempDir(), ".helix-verify", "go-mutation")
	if err := prepareScratch(d); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "case.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := removeScratch(d); err != nil {
		t.Fatalf("own scratch refused: %v", err)
	}
	if _, err := os.Stat(d); !os.IsNotExist(err) {
		t.Fatalf("own scratch not removed")
	}
	// absent path is fine
	if err := removeScratch(d); err != nil {
		t.Fatalf("absent scratch: %v", err)
	}
}

func TestRunValidatesWorkBeforeAnyDelete(t *testing.T) {
	root := t.TempDir()
	victim := mkVictim(t, "Projects/helix")
	_, err := Run(Config{Root: root, Work: victim, Evidence: filepath.Join(root, "ev")}, []Mutant{{ID: "x", Module: "m", File: "f", Old: "a", New: "b", Package: "./", Test: "T", Expect: "e"}})
	if err == nil {
		t.Fatalf("Run accepted a non-scratch --work")
	}
	survives(t, victim)
}

func TestScratchPathRefusalReasons(t *testing.T) {
	for _, p := range []string{"/", "/home", "/usr", "/etc"} {
		_, err := scratchPath(p)
		if err == nil || !strings.Contains(err.Error(), "system directory") {
			t.Fatalf("%q: want the system-directory refusal, got %v", p, err)
		}
	}
	// HOME-relative rules, with HOME inside a ".helix-verify" tree so only the HOME rules can refuse
	base := filepath.Join(t.TempDir(), ".helix-verify", "a")
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(filepath.Join(home, "Projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	for _, p := range []string{home, base, filepath.Join(home, "Projects")} {
		_, err := scratchPath(p)
		if err == nil || !strings.Contains(err.Error(), "HOME") {
			t.Fatalf("%q: want a HOME refusal, got %v", p, err)
		}
	}
	if _, err := scratchPath(filepath.Join(home, ".helix-verify", "ok")); err != nil {
		t.Fatalf("a legitimate scratch below HOME refused: %v", err)
	}
}

// shutil.copytree(symlinks=False) copies the CONTENTS of a symlinked directory.
func TestCopyTreeFollowsSymlinkedDirectories(t *testing.T) {
	src := t.TempDir()
	real := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "f.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(src, "lnk")); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "out")
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "lnk", "f.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("symlinked directory contents not copied: %q %v", b, err)
	}
}
