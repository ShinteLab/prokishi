package registry

import (
	"sync"
	"time"
)

type Direction int8

const (
	DirSend Direction = iota // client → engine
	DirRecv                  // engine → client
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Dir       Direction `json:"dir"`
	Message   string    `json:"message"`
}

type Connection struct {
	ID          string    `json:"id"`
	EngineID    string    `json:"engineId"`
	EngineName  string    `json:"engineName"`
	EnginePath  string    `json:"enginePath"`
	ConnectedAt time.Time `json:"connectedAt"`
}

type USILogEvent struct {
	ConnID string   `json:"connId"`
	Entry  LogEntry `json:"entry"`
}

type EmitFunc func(event string, data any)

type Registry struct {
	mu          sync.RWMutex
	connections map[string]Connection
	logs        map[string][]LogEntry
	emit        EmitFunc
}

func New() *Registry {
	return &Registry{
		connections: make(map[string]Connection),
		logs:        make(map[string][]LogEntry),
	}
}

func (r *Registry) SetEmit(fn EmitFunc) {
	r.mu.Lock()
	r.emit = fn
	r.mu.Unlock()
}

func (r *Registry) Add(conn Connection) {
	r.mu.Lock()
	r.connections[conn.ID] = conn
	r.logs[conn.ID] = nil
	emit := r.emit
	r.mu.Unlock()
	if emit != nil {
		emit("conn-added", conn)
	}
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	delete(r.connections, id)
	delete(r.logs, id)
	emit := r.emit
	r.mu.Unlock()
	if emit != nil {
		emit("conn-removed", id)
	}
}

func (r *Registry) AppendLog(connID string, dir Direction, msg string) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Dir:       dir,
		Message:   msg,
	}
	r.mu.Lock()
	r.logs[connID] = append(r.logs[connID], entry)
	emit := r.emit
	r.mu.Unlock()
	if emit != nil {
		emit("usi-log", USILogEvent{ConnID: connID, Entry: entry})
	}
}

func (r *Registry) ListConnections() []Connection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Connection, 0, len(r.connections))
	for _, c := range r.connections {
		result = append(result, c)
	}
	return result
}

func (r *Registry) GetLogs(id string) []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	src := r.logs[id]
	result := make([]LogEntry, len(src))
	copy(result, src)
	return result
}
