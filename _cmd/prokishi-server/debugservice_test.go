package main

import (
	"testing"
	"time"

	"shinte/prokishi/registry"
)

func TestDebugService_ListConnections(t *testing.T) {
	reg := registry.New()
	connectedAt := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	reg.Add(registry.Connection{
		ID:          "conn-1",
		EngineID:    "engine-1",
		EngineName:  "Apery",
		EnginePath:  `C:\engines\apery.exe`,
		ConnectedAt: connectedAt,
	})

	svc := &DebugService{registry: reg}
	got := svc.ListConnections()

	if len(got) != 1 {
		t.Fatalf("len(ListConnections()) = %d, want 1", len(got))
	}

	want := ConnectionInfo{
		ID:          "conn-1",
		EngineID:    "engine-1",
		EngineName:  "Apery",
		EnginePath:  `C:\engines\apery.exe`,
		ConnectedAt: connectedAt.Format(time.RFC3339),
	}
	if got[0] != want {
		t.Errorf("ListConnections()[0] = %+v, want %+v", got[0], want)
	}
}

func TestDebugService_ListConnections_Empty(t *testing.T) {
	svc := &DebugService{registry: registry.New()}

	got := svc.ListConnections()
	if len(got) != 0 {
		t.Errorf("len(ListConnections()) = %d, want 0", len(got))
	}
}

func TestDebugService_GetLogs(t *testing.T) {
	reg := registry.New()
	reg.Add(registry.Connection{ID: "conn-1"})
	reg.AppendLog("conn-1", registry.DirSend, "usinewgame")
	reg.AppendLog("conn-1", registry.DirRecv, "readyok")

	svc := &DebugService{registry: reg}
	got := svc.GetLogs("conn-1")

	if len(got) != 2 {
		t.Fatalf("len(GetLogs()) = %d, want 2", len(got))
	}

	if got[0].Dir != int8(registry.DirSend) {
		t.Errorf("got[0].Dir = %d, want %d (DirSend)", got[0].Dir, registry.DirSend)
	}
	if got[0].Message != "usinewgame" {
		t.Errorf("got[0].Message = %q, want %q", got[0].Message, "usinewgame")
	}
	if got[1].Dir != int8(registry.DirRecv) {
		t.Errorf("got[1].Dir = %d, want %d (DirRecv)", got[1].Dir, registry.DirRecv)
	}
	if got[1].Message != "readyok" {
		t.Errorf("got[1].Message = %q, want %q", got[1].Message, "readyok")
	}

	if _, err := time.Parse(time.RFC3339Nano, got[0].Timestamp); err != nil {
		t.Errorf("got[0].Timestamp = %q is not RFC3339Nano: %v", got[0].Timestamp, err)
	}
}

func TestDebugService_GetLogs_UnknownID(t *testing.T) {
	svc := &DebugService{registry: registry.New()}

	got := svc.GetLogs("no-such-connection")
	if len(got) != 0 {
		t.Errorf("len(GetLogs()) = %d, want 0", len(got))
	}
}
