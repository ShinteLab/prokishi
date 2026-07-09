package usi

import (
	"io"
	"testing"
)

type nopReadCloser struct{ io.Reader }

func (nopReadCloser) Close() error { return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestSenderTerminateIdempotent(t *testing.T) {
	pr, pw := io.Pipe()
	s := &Sender{
		out:   nopReadCloser{pr},
		in:    nopWriteCloser{pw},
		OutCh: make(chan string),
	}

	if err := s.Terminate(); err != nil {
		t.Fatalf("first Terminate() error: %v", err)
	}

	if err := s.Terminate(); err != nil {
		t.Fatalf("second Terminate() error: %v", err)
	}
}
