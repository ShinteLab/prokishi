// Package testfakeengine provides a minimal, buildable "fake engine"
// program used to integration-test code that spawns and talks to a shogi
// engine subprocess over stdin/stdout (usi.Sender today, potentially the
// server package later) without depending on a real engine binary.
package testfakeengine

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// fakeEngineSource is a trivial stdin-echo program: it reads lines from
// stdin and writes each one back to stdout prefixed with "echo: ", so a
// caller can verify a round trip through a real subprocess. It exits
// cleanly on EOF or on receiving the line "quit".
const fakeEngineSource = `package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		if line == "quit" {
			return
		}
		fmt.Fprintf(writer, "echo: %s\n", line)
		writer.Flush()
	}
}
`

// Build writes the embedded fake-engine source to a temp file, compiles it
// into a real executable with "go build", and returns the path to the
// resulting binary. It fails the test (via tb.Fatalf) if the build fails.
func Build(tb testing.TB) string {
	tb.Helper()

	dir := tb.TempDir()

	srcPath := filepath.Join(dir, "fakeengine.go")
	if err := os.WriteFile(srcPath, []byte(fakeEngineSource), 0o644); err != nil {
		tb.Fatalf("testfakeengine: write source: %v", err)
	}

	binName := "fakeengine"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)

	cmd := exec.Command("go", "build", "-o", binPath, srcPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("testfakeengine: go build failed: %v\n%s", err, out)
	}

	return binPath
}
