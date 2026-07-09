//go:build integration

package usi_test

import (
	"testing"
	"time"

	"prokishi/internal/testfakeengine"
	"prokishi/usi"
)

// TestSenderIntegrationRoundTrip spawns a real (built) fake-engine
// subprocess through usi.NewSender and verifies a line sent via Send()
// round-trips back through OutCh, then verifies the Sender shuts down
// cleanly when the engine process exits.
//
// NOTE on shutdown: we deliberately do NOT call s.Terminate() ourselves
// here. NewSender starts an internal goroutine that calls cmd.Wait() and
// then always calls s.Terminate() once the process exits. Terminate()
// unconditionally does `close(s.OutCh)` after nil-ing the field out, with
// no guard against being invoked twice. If a caller also invokes
// Terminate() explicitly, the explicit close (via s.in.Close()) makes the
// fake engine see EOF on stdin and exit, which makes that internal
// goroutine invoke Terminate() a second time -> close of a nil channel ->
// panic in a goroutine the test cannot recover from. Since the task here is
// to test the existing Sender, not to change its behavior, we exercise the
// single, automatic termination path (engine exits -> internal goroutine
// terminates the Sender exactly once) instead of double-terminating it.
func TestSenderIntegrationRoundTrip(t *testing.T) {
	binPath := testfakeengine.Build(t)

	s, err := usi.NewSender(binPath)
	if err != nil {
		t.Fatalf("NewSender() error: %v", err)
	}

	// Capture the channel handle up front: Terminate() (called internally
	// once the engine process exits) nils out the exported OutCh field, but
	// the underlying channel object referenced by our local copy is
	// unaffected and still gets closed.
	ch := s.OutCh

	if err := s.Send("position startpos"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	select {
	case got := <-ch:
		want := "echo: position startpos"
		if got != want {
			t.Fatalf("OutCh delivered %q, want %q", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for round-tripped output from fake engine")
	}

	// Ask the fake engine to exit. This closes its process, which makes
	// NewSender's internal cmd.Wait() goroutine call Sender.Terminate()
	// automatically.
	if err := s.Send("quit"); err != nil {
		t.Fatalf("Send(quit) error: %v", err)
	}

	select {
	case v, ok := <-ch:
		if ok {
			t.Fatalf("expected OutCh to be closed after engine exit, got value %q", v)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Sender to auto-terminate after engine exit")
	}
}
