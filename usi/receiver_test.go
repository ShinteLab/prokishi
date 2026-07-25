package usi_test

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ShinteLab/prokishi/usi"
)

// TestReceiverMultiLineInput verifies that feeding multi-line input through
// the reader delivers one message per line, in order, on InCh, and that the
// internal read-loop goroutine does not keep delivering (or hang) once the
// reader hits EOF.
func TestReceiverMultiLineInput(t *testing.T) {
	pr, pw := io.Pipe()
	var out bytes.Buffer

	r := usi.NewReceiver(&out, pr)

	lines := []string{"usi", "isready", "usinewgame", "go"}

	go func() {
		for _, l := range lines {
			_, _ = pw.Write([]byte(l + "\n"))
		}
		_ = pw.Close()
	}()

	for _, want := range lines {
		select {
		case got := <-r.InCh:
			if got != want {
				t.Fatalf("InCh delivered %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for line %q", want)
		}
	}

	// After EOF, the read-loop goroutine should stop scanning and stop
	// sending; verify no further (unexpected) value shows up on InCh.
	select {
	case got := <-r.InCh:
		t.Fatalf("unexpected extra value on InCh after EOF: %q", got)
	case <-time.After(200 * time.Millisecond):
		// expected: no more data, no hang.
	}
}

// TestReceiverSend verifies that Send writes the command followed by a
// trailing newline to the underlying writer.
func TestReceiverSend(t *testing.T) {
	var out bytes.Buffer

	// Reader side is never used by Send; give it an already-exhausted
	// reader so the read-loop goroutine returns immediately.
	r := usi.NewReceiver(&out, strings.NewReader(""))

	if err := r.Send("readyok"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	want := "readyok\n"
	if got := out.String(); got != want {
		t.Fatalf("writer got %q, want %q", got, want)
	}
}
