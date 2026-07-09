package registry

import (
	"sync"
	"testing"
	"time"
)

func TestAddAndList(t *testing.T) {
	r := New()
	conn := Connection{
		ID:         "conn-1",
		EngineID:   "eng-1",
		EngineName: "Apery",
		PID:        1234,
		ConnectedAt: time.Now(),
	}

	r.Add(conn)

	conns := r.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("ListConnections() = %d, want 1", len(conns))
	}
	if conns[0].ID != "conn-1" {
		t.Errorf("ID = %q, want %q", conns[0].ID, "conn-1")
	}
}

func TestRemove(t *testing.T) {
	r := New()
	r.Add(Connection{ID: "conn-1", ConnectedAt: time.Now()})
	r.Add(Connection{ID: "conn-2", ConnectedAt: time.Now()})

	r.Remove("conn-1")

	conns := r.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("ListConnections() = %d, want 1", len(conns))
	}
	if conns[0].ID != "conn-2" {
		t.Errorf("remaining ID = %q, want %q", conns[0].ID, "conn-2")
	}
}

func TestAppendLogAndGetLogs(t *testing.T) {
	r := New()
	r.Add(Connection{ID: "conn-1", ConnectedAt: time.Now()})

	r.AppendLog("conn-1", DirSend, "position startpos")
	r.AppendLog("conn-1", DirRecv, "bestmove 7g7f")

	logs := r.GetLogs("conn-1")
	if len(logs) != 2 {
		t.Fatalf("GetLogs() = %d, want 2", len(logs))
	}
	if logs[0].Dir != DirSend {
		t.Errorf("logs[0].Dir = %d, want DirSend(%d)", logs[0].Dir, DirSend)
	}
	if logs[1].Message != "bestmove 7g7f" {
		t.Errorf("logs[1].Message = %q, want %q", logs[1].Message, "bestmove 7g7f")
	}
}

func TestGetLogs_ReturnsCopy(t *testing.T) {
	r := New()
	r.Add(Connection{ID: "conn-1", ConnectedAt: time.Now()})
	r.AppendLog("conn-1", DirSend, "usi")

	logs := r.GetLogs("conn-1")
	logs[0].Message = "modified"

	original := r.GetLogs("conn-1")
	if original[0].Message != "usi" {
		t.Errorf("GetLogs returned a reference, not a copy")
	}
}

func TestEmitCallbacks(t *testing.T) {
	r := New()

	var events []string
	var mu sync.Mutex
	r.SetEmit(func(event string, data any) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	})

	r.Add(Connection{ID: "conn-1", ConnectedAt: time.Now()})
	r.AppendLog("conn-1", DirSend, "usi")
	r.Remove("conn-1")

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 3 {
		t.Fatalf("events = %v, want 3 events", events)
	}
	if events[0] != "conn-added" {
		t.Errorf("events[0] = %q, want %q", events[0], "conn-added")
	}
	if events[1] != "usi-log" {
		t.Errorf("events[1] = %q, want %q", events[1], "usi-log")
	}
	if events[2] != "conn-removed" {
		t.Errorf("events[2] = %q, want %q", events[2], "conn-removed")
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := New()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			connID := "conn-" + string(rune('A'+id%26))
			r.Add(Connection{ID: connID, ConnectedAt: time.Now()})
			r.AppendLog(connID, DirSend, "usi")
			r.ListConnections()
			r.GetLogs(connID)
			r.Remove(connID)
		}(i)
	}

	wg.Wait()
}
