package main

import "testing"

func TestGreetService_Greet(t *testing.T) {
	g := &GreetService{}

	got := g.Greet("World")
	want := "Hello World!"
	if got != want {
		t.Errorf("Greet(%q) = %q, want %q", "World", got, want)
	}
}
