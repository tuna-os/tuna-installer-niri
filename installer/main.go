// TunaOS Niri Installer — Go backend
//
// Follows DankMaterialShell's architecture: Go backend services
// exposed to QML via DBus or stdout-pipe for the installer.
// This backend wraps fisherman and provides disk discovery.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// DiskInfo represents a block device from lsblk
type DiskInfo struct {
	Name      string `json:"name"`
	Size      string `json:"size"`
	Type      string `json:"type"`
	Transport string `json:"tran,omitempty"`
}

// Recipe is the fisherman install recipe
type Recipe struct {
	Disk            string     `json:"disk"`
	Filesystem      string     `json:"filesystem"`
	BtrfsSubvolumes bool       `json:"btrfsSubvolumes"`
	Encryption      Encryption `json:"encryption"`
	Image           string     `json:"image"`
	TargetImgref    string     `json:"targetImgref"`
	SelinuxDisabled bool       `json:"selinuxDisabled"`
	Hostname        string     `json:"hostname"`
}

type Encryption struct {
	Type       string `json:"type"`
	Passphrase string `json:"passphrase"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: tuna-installer-niri <command> [args...]")
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  discover-disks     List available block devices as JSON")
		fmt.Fprintln(os.Stderr, "  install <recipe>   Run fisherman with the given recipe JSON")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "discover-disks":
		discoverDisks()
	case "install":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: tuna-installer-niri install <recipe-json>")
			os.Exit(1)
		}
		runInstall(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func discoverDisks() {
	cmd := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,TYPE,TRAN")
	output, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "lsblk failed: %v\n", err)
		os.Exit(1)
	}

	var result struct {
		Blockdevices []json.RawMessage `json:"blockdevices"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		fmt.Fprintf(os.Stderr, "parse lsblk output: %v\n", err)
		os.Exit(1)
	}

	var disks []DiskInfo
	for _, raw := range result.Blockdevices {
		var d DiskInfo
		if err := json.Unmarshal(raw, &d); err != nil {
			continue
		}
		if d.Type == "disk" {
			disks = append(disks, d)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(disks); err != nil {
		fmt.Fprintf(os.Stderr, "encode output: %v\n", err)
		os.Exit(1)
	}
}

func runInstall(recipeJSON string) {
	var recipe Recipe
	if err := json.Unmarshal([]byte(recipeJSON), &recipe); err != nil {
		fmt.Fprintf(os.Stderr, "invalid recipe: %v\n", err)
		os.Exit(1)
	}

	// Write recipe to temp file
	tmpFile, err := os.CreateTemp("", "fisherman-recipe-*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())

	enc := json.NewEncoder(tmpFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(recipe); err != nil {
		fmt.Fprintf(os.Stderr, "write recipe: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	// Run fisherman
	cmd := exec.Command("fisherman", tmpFile.Name())
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Stream output line by line
	// (In production, use a pipe and report progress via DBus or IPC)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "fisherman failed: %v\n", err)
		fmt.Fprint(os.Stdout, stderr.String())
		os.Exit(1)
	}

	fmt.Print(stdout.String())
	if stderr.Len() > 0 {
		fmt.Fprint(os.Stdout, stderr.String())
	}
}

// For QML integration via Quickshell, expose methods via a DBus service.
// The QML frontend calls the Go backend through a simple process-pipe
// (stdout JSON protocol) or DBus interface.
//
// Example QML import:
//
//	import org.tunaos.installer 1.0
//
//	InstallerBackend {
//	    function discoverDiskins() { ... }
//	    function startInstall(disk, hostname) { ... }
//	    signal outputChanged(string log)
//	    signal installFinished(bool success)
//	}
//
// The DBus service name: org.tunaos.Installer
// Object path: /org/tunaos/Installer
var _ = sync.Mutex{} // ensure sync import is used for future thread-safe additions
