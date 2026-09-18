package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallReadsRecipeFromStdin guards the gh-22 fix: the recipe (which can
// carry a LUKS passphrase) must reach the backend over stdin, never over argv,
// because /proc/PID/cmdline is world-readable on Linux. runInstall must be
// invoked with the recipe that came from stdin.
func TestRunInstallRejectsEmptyRecipe(t *testing.T) {
	// runInstall is the backend's stdin entry point. An empty stdin must be a
	// hard error, not a silent pass-through (which would let a truncated
	// recipe reach fisherman).
	if os.Getenv("GO_TEST_RUN_INSTALL") == "1" {
		runInstall("")
		// If we get here the empty recipe was accepted — fail loudly via the
		// test process itself.
		os.Exit(0)
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestRunInstallRejectsEmptyRecipe")
	cmd.Env = append(os.Environ(), "GO_TEST_RUN_INSTALL=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("runInstall(\"\") accepted an empty recipe; stdout=%s", out)
	}
	if !strings.Contains(string(out), "invalid recipe: empty") {
		t.Fatalf("expected 'invalid recipe: empty' error, got: %s", out)
	}
}

// TestRunInstallSubprocessHelper is the re-exec target for the scenarios
// below: they each need runInstall's os.Exit calls to end a child process,
// not the test binary, so they invoke this test by name with the recipe and
// a trigger flag passed through the environment. On its own (no flag) it is
// a no-op.
func TestRunInstallSubprocessHelper(t *testing.T) {
	if os.Getenv("GO_TEST_RUN_INSTALL") != "1" {
		return
	}
	runInstall(os.Getenv("GO_TEST_RUN_INSTALL_RECIPE"))
	os.Exit(0)
}

// envWithout returns the current environment with the named variables
// removed, for tests that need e.g. no $HOME rather than merely an empty one.
func envWithout(keys ...string) []string {
	drop := map[string]bool{}
	for _, k := range keys {
		drop[k] = true
	}
	var out []string
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i >= 0 && drop[kv[:i]] {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// runInstallSubprocess runs runInstall(recipeJSON) in a child process with
// the given environment, returning what it printed and how it exited.
func runInstallSubprocess(t *testing.T, recipeJSON string, env []string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunInstallSubprocessHelper$")
	cmd.Env = append(env, "GO_TEST_RUN_INSTALL=1", "GO_TEST_RUN_INSTALL_RECIPE="+recipeJSON)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running subprocess: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), code
}

func TestRunInstallRejectsInvalidJSON(t *testing.T) {
	_, stderr, code := runInstallSubprocess(t, "{not json", os.Environ())
	if code == 0 {
		t.Fatalf("runInstall accepted invalid JSON; want failure")
	}
	if !strings.Contains(stderr, "invalid recipe:") {
		t.Fatalf("expected 'invalid recipe:' error, got: %s", stderr)
	}
}

// TestRunInstallRejectsMissingImageOutsideLiveISO guards spec §4: a recipe
// with no image is only valid when liveISOImage() reports a booted image to
// fall back on. An empty PATH means the bootc probe liveISOImage() runs
// necessarily fails, so this deterministically exercises the live-ISO-less
// branch regardless of what the host machine happens to have installed.
func TestRunInstallRejectsMissingImageOutsideLiveISO(t *testing.T) {
	noBootc := t.TempDir()
	env := append(os.Environ(), pathWith(noBootc))
	_, stderr, code := runInstallSubprocess(t, `{"disk":"/dev/sda","filesystem":"btrfs"}`, env)
	if code == 0 {
		t.Fatalf("runInstall accepted a missing image outside live-ISO mode; want failure")
	}
	if !strings.Contains(stderr, "invalid recipe: image is required outside live-ISO mode") {
		t.Fatalf("expected the live-ISO-less image error, got: %s", stderr)
	}
}

const sudoStub = `#!/bin/sh
cat "$2" 1>&2
echo STUB_SUDO_STDERR 1>&2
exit "${STUB_SUDO_EXIT:-7}"
`

// TestRunInstallRunsFishermanAndPersistsLog covers the full happy-ish path:
// a recipe with an image passes validation, gets its DistroID default filled
// in, is written to a private recipe file, and is handed to the fisherman
// command (stubbed here so the test never needs real privileges) with both
// stdout/stderr teed into the persistent install log alongside the pipe the
// QML frontend reads.
func TestRunInstallRunsFishermanAndPersistsLog(t *testing.T) {
	bin := t.TempDir()
	writeStub(t, bin, "sudo", sudoStub)

	runtimeDir := t.TempDir()
	stateHome := t.TempDir()

	env := append(envWithout("XDG_RUNTIME_DIR", "XDG_STATE_HOME"),
		pathWith(bin),
		"XDG_RUNTIME_DIR="+runtimeDir,
		"XDG_STATE_HOME="+stateHome,
	)

	recipe := `{"disk":"/dev/sda","filesystem":"btrfs","image":"ghcr.io/tuna-os/skipjack:niri"}`
	_, stderr, code := runInstallSubprocess(t, recipe, env)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stub fisherman exits non-zero); stderr=%s", code, stderr)
	}
	if !strings.Contains(stderr, "fisherman failed:") {
		t.Fatalf("expected 'fisherman failed:' error, got: %s", stderr)
	}
	if !strings.Contains(stderr, `"distroID": "tunaos"`) {
		t.Fatalf("expected the default distroID in the recipe fisherman received, got: %s", stderr)
	}
	if !strings.Contains(stderr, "STUB_SUDO_STDERR") {
		t.Fatalf("expected the stub fisherman's own stderr to be teed through, got: %s", stderr)
	}

	logPath := filepath.Join(stateHome, "tuna-installer", "install.log")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("install log was not written: %v", err)
	}
	if !strings.Contains(string(logData), "=== install started") {
		t.Errorf("install log missing start marker: %s", logData)
	}
	if !strings.Contains(string(logData), "=== install exited") {
		t.Errorf("install log missing exit marker: %s", logData)
	}
	if !strings.Contains(string(logData), "STUB_SUDO_STDERR") {
		t.Errorf("install log missing fisherman's teed output: %s", logData)
	}
}

// TestRunInstallSucceedsWhenFishermanSucceeds covers the runErr == nil path:
// no "fisherman failed" error and a clean exit once the stub reports success.
func TestRunInstallSucceedsWhenFishermanSucceeds(t *testing.T) {
	bin := t.TempDir()
	writeStub(t, bin, "sudo", sudoStub)

	env := append(envWithout("XDG_RUNTIME_DIR", "XDG_STATE_HOME"),
		pathWith(bin),
		"XDG_RUNTIME_DIR="+t.TempDir(),
		"XDG_STATE_HOME="+t.TempDir(),
		"STUB_SUDO_EXIT=0",
	)

	recipe := `{"disk":"/dev/sda","filesystem":"btrfs","image":"ghcr.io/tuna-os/skipjack:niri"}`
	_, stderr, code := runInstallSubprocess(t, recipe, env)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%s", code, stderr)
	}
	if strings.Contains(stderr, "fisherman failed:") {
		t.Fatalf("did not expect a 'fisherman failed:' error, got: %s", stderr)
	}
}

// TestRunInstallFallsBackToStderrWhenLogUnavailable covers the logErr != nil
// branch: with no $HOME and no $XDG_STATE_HOME, installLogDir() can't resolve
// anywhere to write, so runInstall must still run fisherman — straight to
// os.Stdout/os.Stderr, no tee — rather than aborting the install over a
// logging failure.
func TestRunInstallFallsBackToStderrWhenLogUnavailable(t *testing.T) {
	bin := t.TempDir()
	writeStub(t, bin, "sudo", sudoStub)

	env := append(envWithout("HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"),
		pathWith(bin),
		"XDG_RUNTIME_DIR="+t.TempDir(),
	)

	recipe := `{"disk":"/dev/sda","filesystem":"btrfs","image":"ghcr.io/tuna-os/skipjack:niri"}`
	_, stderr, code := runInstallSubprocess(t, recipe, env)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr)
	}
	if !strings.Contains(stderr, "install log:") {
		t.Fatalf("expected the install-log-unavailable warning, got: %s", stderr)
	}
	if !strings.Contains(stderr, "STUB_SUDO_STDERR") {
		t.Fatalf("expected the stub fisherman's stderr straight on stderr, got: %s", stderr)
	}
}

func TestParseLSBLKOutput(t *testing.T) {
	input := []byte(`{
		"blockdevices": [
			{"name": "nvme0n1", "size": "500G", "type": "disk", "tran": "nvme"},
			{"name": "loop0", "size": "2G", "type": "loop"},
			{"name": "sda", "size": "1T", "type": "disk", "tran": "sata"}
		]
	}`)

	disks, err := parseLSBLKOutput(input)
	if err != nil {
		t.Fatalf("parseLSBLKOutput failed: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if disks[0].Name != "nvme0n1" || disks[0].Transport != "nvme" {
		t.Errorf("unexpected disk 0: %+v", disks[0])
	}
	if disks[1].Name != "sda" || disks[1].Transport != "sata" {
		t.Errorf("unexpected disk 1: %+v", disks[1])
	}
}

func TestRecipeUnmarshal(t *testing.T) {
	input := []byte(`{
		"disk": "/dev/sda",
		"filesystem": "btrfs",
		"btrfsSubvolumes": true,
		"encryption": {
			"type": "luks-passphrase",
			"passphrase": "secret-passphrase"
		},
		"image": "ghcr.io/tuna-os/skipjack:niri",
		"targetImgref": "ghcr.io/tuna-os/skipjack:stable",
		"bootloader": "systemd",
		"composeFsBackend": true,
		"flatpaks": ["org.mozilla.firefox"],
		"additionalImageStores": ["/run/media/store"],
		"distroID": "tunaos",
		"selinuxDisabled": true,
		"hostname": "niri-host"
	}`)

	var r Recipe
	if err := json.Unmarshal(input, &r); err != nil {
		t.Fatalf("json.Unmarshal(Recipe) failed: %v", err)
	}

	if r.Disk != "/dev/sda" || r.Filesystem != "btrfs" || !r.BtrfsSubvolumes {
		t.Errorf("unexpected storage fields: %+v", r)
	}
	if r.Encryption.Type != "luks-passphrase" || r.Encryption.Passphrase != "secret-passphrase" {
		t.Errorf("unexpected encryption: %+v", r.Encryption)
	}
	if r.Image != "ghcr.io/tuna-os/skipjack:niri" || r.TargetImgref != "ghcr.io/tuna-os/skipjack:stable" {
		t.Errorf("unexpected image fields: %+v", r)
	}
	if r.Bootloader != "systemd" || !r.ComposeFsBackend {
		t.Errorf("unexpected boot/composefs config: %+v", r)
	}
	if len(r.Flatpaks) != 1 || r.Flatpaks[0] != "org.mozilla.firefox" {
		t.Errorf("unexpected flatpaks: %+v", r.Flatpaks)
	}
	if len(r.AdditionalImageStores) != 1 || r.AdditionalImageStores[0] != "/run/media/store" {
		t.Errorf("unexpected additional image stores: %+v", r.AdditionalImageStores)
	}
	if r.Hostname != "niri-host" || r.DistroID != "tunaos" || !r.SelinuxDisabled {
		t.Errorf("unexpected system metadata: %+v", r)
	}
}
