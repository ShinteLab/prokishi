package usi

import (
	"bufio"
	"io"
	"testing"
	"time"
)

// These tests construct a Sender directly (bypassing NewSender, which spawns
// a real subprocess via exec.Command) by setting its unexported out/in
// fields to the two ends of io.Pipe()s. That lets us exercise
// monitorStdout, Send, and Terminate without ever starting a process.

// TestSenderMonitorStdout verifies that lines written to the (fake) stdout
// pipe are relayed, one per line and in order, onto OutCh.
func TestSenderMonitorStdout(t *testing.T) {
	pr, pw := io.Pipe()

	s := &Sender{
		out:   pr,
		OutCh: make(chan string),
	}
	go s.monitorStdout()

	lines := []string{"id name FakeEngine", "usiok", "readyok"}

	go func() {
		for _, l := range lines {
			_, _ = pw.Write([]byte(l + "\n"))
		}
		_ = pw.Close()
	}()

	for _, want := range lines {
		select {
		case got := <-s.OutCh:
			if got != want {
				t.Fatalf("OutCh delivered %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for line %q", want)
		}
	}
}

// TestSenderSend verifies that Send writes line + "\n" to the (fake) stdin
// pipe.
func TestSenderSend(t *testing.T) {
	pr, pw := io.Pipe()

	s := &Sender{in: pw}

	got := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(pr)
		if scanner.Scan() {
			got <- scanner.Text()
			return
		}
		got <- ""
	}()

	if err := s.Send("position startpos"); err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	select {
	case line := <-got:
		if line != "position startpos" {
			t.Fatalf("stdin pipe received %q, want %q", line, "position startpos")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Send() to reach the stdin pipe")
	}
}

// TestSenderTerminate verifies that Terminate closes the pipes and closes
// OutCh without panicking.
func TestSenderTerminate(t *testing.T) {
	outR, outW := io.Pipe()
	_, inW := io.Pipe()
	t.Cleanup(func() { _ = outW.Close() })

	s := &Sender{
		out:   outR,
		in:    inW,
		OutCh: make(chan string),
	}
	go s.monitorStdout()

	// Capture the channel handle before Terminate nils out s.OutCh, since
	// close() happens on the underlying channel object either way.
	ch := s.OutCh

	if err := s.Terminate(); err != nil {
		t.Fatalf("Terminate() error: %v", err)
	}

	select {
	case v, ok := <-ch:
		if ok {
			t.Fatalf("expected OutCh to be closed, got value %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for OutCh to close after Terminate")
	}
}

// TestSenderSendAfterTerminate verifies that calling Send after Terminate
// does not panic; it should simply surface an error from the now-closed
// stdin pipe.
func TestSenderSendAfterTerminate(t *testing.T) {
	outR, outW := io.Pipe()
	_, inW := io.Pipe()
	t.Cleanup(func() { _ = outW.Close() })

	s := &Sender{
		out:   outR,
		in:    inW,
		OutCh: make(chan string),
	}
	go s.monitorStdout()

	if err := s.Terminate(); err != nil {
		t.Fatalf("Terminate() error: %v", err)
	}

	err := s.Send("this must not panic")
	if err == nil {
		t.Fatal("expected an error sending on a terminated Sender, got nil")
	}
}
