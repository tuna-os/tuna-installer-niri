package main

// Unit tests for offline.go — offline-install and sandbox plumbing.
// Contract: ../../INSTALLER-FRONTENDS.md §3 (privileges) and §4 (offline).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFishermanCommandOutsideFlatpak(t *testing.T) {
	if inFlatpak() {
		t.Skip("running inside a Flatpak sandbox")
	}
	got := fishermanCommand()
	want := []string{"sudo", "/usr/local/bin/fisherman"}
	if len(got) != len(want) {
		t.Fatalf("fishermanCommand() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fishermanCommand() = %v, want %v", got, want)
		}
	}
}

func TestHostCommandPassthroughOutsideFlatpak(t *testing.T) {
	if inFlatpak() {
		t.Skip("running inside a Flatpak sandbox")
	}
	got := hostCommand("bootc", "status")
	want := []string{"bootc", "status"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("hostCommand() = %v, want %v", got, want)
	}
}

func TestOfflineStoresEnvParsingDedupAndDirFilter(t *testing.T) {
	base := t.TempDir()
	dirA := filepath.Join(base, "store-a")
	dirB := filepath.Join(base, "store-b")
	plain := filepath.Join(base, "plain-file")
	for _, d := range []string{dirA, dirB} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TUNA_OFFLINE_STORES", strings.Join([]string{dirA, dirB, plain, dirA}, ":"))

	got := offlineStores()
	// dirA and dirB present exactly once, in order; the plain file and the
	// default store (not a directory in the test environment) are filtered.
	if len(got) != 2 {
		t.Fatalf("offlineStores() = %v, want exactly [dirA, dirB]", got)
	}
	if got[0] != dirA || got[1] != dirB {
		t.Fatalf("offlineStores() = %v, want [%s, %s]", got, dirA, dirB)
	}
}

func TestOfflineImagesEmptyWithoutStores(t *testing.T) {
	if got := offlineImages(nil); len(got) != 0 {
		t.Fatalf("offlineImages(nil) = %v, want empty", got)
	}
	// A store root that cannot exist yields nothing rather than an error.
	if got := offlineImages([]string{filepath.Join(t.TempDir(), "missing-store")}); len(got) != 0 {
		t.Fatalf("offlineImages(missing store) = %v, want empty", got)
	}
}

func TestWriteRecipe(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	path, err := writeRecipe([]byte(`{"image":"localhost/foo:1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, filepath.Join(dir, "tuna-installer")) {
		t.Fatalf("recipe path %s not under runtime dir %s", path, dir)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"image":"localhost/foo:1"}` {
		t.Errorf("recipe content = %q", data)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// The recipe may hold a LUKS passphrase / user password — never readable
	// by other users.
	if info.Mode().Perm() != 0o600 {
		t.Errorf("recipe mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestWriteRecipeFallsBackToTmp(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	path, err := writeRecipe([]byte("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Errorf("recipe content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("recipe mode = %v, want 0600", info.Mode().Perm())
	}
}
