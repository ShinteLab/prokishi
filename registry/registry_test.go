package registry_test

import (
	"fmt"
	"shinte/prokishi/registry"
	"sync"
	"testing"
	"time"
)

// emitSpy is a simple concurrency-safe closure-based spy for registry.EmitFunc.
type emitSpy struct {
	mu    sync.Mutex
	calls []emitCall
}

type emitCall struct {
	event string
	data  any
}

func (s *emitSpy) fn(event string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, emitCall{event: event, data: data})
}

func (s *emitSpy) all() []emitCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]emitCall, len(s.calls))
	copy(out, s.calls)
	return out
}

func (s *emitSpy) last() (emitCall, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.calls) == 0 {
		return emitCall{}, false
	}
	return s.calls[len(s.calls)-1], true
}

func newConn(id string) registry.Connection {
	return registry.Connection{
		ID:          id,
		EngineID:    "engine-" + id,
		EngineName:  "Engine " + id,
		EnginePath:  "/path/to/" + id,
		ConnectedAt: time.Now(),
	}
}

func TestNew(t *testing.T) {
	r := registry.New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	conns := r.ListConnections()
	if len(conns) != 0 {
		t.Fatalf("expected empty registry, got %d connections", len(conns))
	}
	logs := r.GetLogs("nonexistent")
	if len(logs) != 0 {
		t.Fatalf("expected no logs for unknown id, got %d", len(logs))
	}
}

func TestAdd(t *testing.T) {
	r := registry.New()
	spy := &emitSpy{}
	r.SetEmit(spy.fn)

	conn := newConn("conn-1")
	r.Add(conn)

	conns := r.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0] != conn {
		t.Fatalf("stored connection %+v does not match added connection %+v", conns[0], conn)
	}

	call, ok := spy.last()
	if !ok {
		t.Fatal("expected an emit call after Add, got none")
	}
	if call.event != "conn-added" {
		t.Fatalf("expected event %q, got %q", "conn-added", call.event)
	}
	gotConn, ok := call.data.(registry.Connection)
	if !ok {
		t.Fatalf("expected emit payload of type registry.Connection, got %T", call.data)
	}
	if gotConn != conn {
		t.Fatalf("emit payload %+v does not match added connection %+v", gotConn, conn)
	}
}

func TestNilEmitDoesNotPanic(t *testing.T) {
	r := registry.New()

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("operations without EmitFunc panicked: %v", rec)
		}
	}()

	conn := newConn("conn-1")
	r.Add(conn)
	r.AppendLog(conn.ID, registry.DirSend, "hello")
	r.ListConnections()
	r.GetLogs(conn.ID)
	r.Remove(conn.ID)
}

func TestRemove(t *testing.T) {
	r := registry.New()
	spy := &emitSpy{}
	r.SetEmit(spy.fn)

	conn := newConn("conn-1")
	r.Add(conn)
	r.AppendLog(conn.ID, registry.DirSend, "usi")

	if logs := r.GetLogs(conn.ID); len(logs) != 1 {
		t.Fatalf("expected 1 log before removal, got %d", len(logs))
	}

	r.Remove(conn.ID)

	conns := r.ListConnections()
	if len(conns) != 0 {
		t.Fatalf("expected connection to be removed, still have %d", len(conns))
	}
	if logs := r.GetLogs(conn.ID); len(logs) != 0 {
		t.Fatalf("expected logs to be removed, still have %d", len(logs))
	}

	call, ok := spy.last()
	if !ok {
		t.Fatal("expected an emit call after Remove, got none")
	}
	if call.event != "conn-removed" {
		t.Fatalf("expected event %q, got %q", "conn-removed", call.event)
	}
	gotID, ok := call.data.(string)
	if !ok {
		t.Fatalf("expected emit payload of type string, got %T", call.data)
	}
	if gotID != conn.ID {
		t.Fatalf("expected emit payload %q, got %q", conn.ID, gotID)
	}
}

func TestAppendLog(t *testing.T) {
	r := registry.New()
	spy := &emitSpy{}
	r.SetEmit(spy.fn)

	conn := newConn("conn-1")
	r.Add(conn)

	r.AppendLog(conn.ID, registry.DirSend, "position startpos")
	time.Sleep(time.Millisecond) // ensure the clock advances between entries
	r.AppendLog(conn.ID, registry.DirRecv, "bestmove 7g7f")

	logs := r.GetLogs(conn.ID)
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}

	first, second := logs[0], logs[1]
	if first.Dir != registry.DirSend || first.Message != "position startpos" {
		t.Fatalf("unexpected first entry: %+v", first)
	}
	if second.Dir != registry.DirRecv || second.Message != "bestmove 7g7f" {
		t.Fatalf("unexpected second entry: %+v", second)
	}
	if first.Timestamp.IsZero() || second.Timestamp.IsZero() {
		t.Fatalf("expected non-zero timestamps, got %+v and %+v", first, second)
	}
	if second.Timestamp.Before(first.Timestamp) {
		t.Fatalf("expected timestamps to advance: first=%v second=%v", first.Timestamp, second.Timestamp)
	}

	calls := spy.all()
	// One "conn-added" call plus two "usi-log" calls.
	if len(calls) != 3 {
		t.Fatalf("expected 3 emit calls, got %d", len(calls))
	}
	for i, want := range []struct {
		dir registry.Direction
		msg string
	}{
		{registry.DirSend, "position startpos"},
		{registry.DirRecv, "bestmove 7g7f"},
	} {
		call := calls[i+1]
		if call.event != "usi-log" {
			t.Fatalf("call %d: expected event %q, got %q", i, "usi-log", call.event)
		}
		evt, ok := call.data.(registry.USILogEvent)
		if !ok {
			t.Fatalf("call %d: expected payload of type registry.USILogEvent, got %T", i, call.data)
		}
		if evt.ConnID != conn.ID {
			t.Fatalf("call %d: expected ConnID %q, got %q", i, conn.ID, evt.ConnID)
		}
		if evt.Entry.Dir != want.dir || evt.Entry.Message != want.msg {
			t.Fatalf("call %d: unexpected entry %+v", i, evt.Entry)
		}
	}
}

func TestGetLogsReturnsDefensiveCopy(t *testing.T) {
	r := registry.New()
	conn := newConn("conn-1")
	r.Add(conn)
	r.AppendLog(conn.ID, registry.DirSend, "original")

	logs := r.GetLogs(conn.ID)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	// Mutate the returned slice/entry.
	logs[0].Message = "mutated"
	logs = append(logs, registry.LogEntry{Message: "injected"})

	again := r.GetLogs(conn.ID)
	if len(again) != 1 {
		t.Fatalf("expected internal state to still have 1 log entry, got %d", len(again))
	}
	if again[0].Message != "original" {
		t.Fatalf("expected internal log message to remain %q, got %q", "original", again[0].Message)
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := registry.New()
	spy := &emitSpy{}
	r.SetEmit(spy.fn)

	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			id := fmt.Sprintf("conn-%d", g)
			conn := newConn(id)
			r.Add(conn)
			for i := 0; i < iterations; i++ {
				r.AppendLog(id, registry.DirSend, fmt.Sprintf("msg-%d", i))
				_ = r.ListConnections()
				_ = r.GetLogs(id)
			}
		}(g)
	}
	wg.Wait()

	conns := r.ListConnections()
	if len(conns) != goroutines {
		t.Fatalf("expected %d connections, got %d", goroutines, len(conns))
	}
	for g := 0; g < goroutines; g++ {
		id := fmt.Sprintf("conn-%d", g)
		logs := r.GetLogs(id)
		if len(logs) != iterations {
			t.Fatalf("connection %s: expected %d log entries, got %d", id, iterations, len(logs))
		}
	}
}
