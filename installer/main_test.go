package main

import (
	"os"
	"os/exec"
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
