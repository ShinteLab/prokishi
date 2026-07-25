package main

import (
	"shinte/prokishi/registry"
	"time"
)

type ConnectionInfo struct {
	ID          string `json:"id"`
	EngineID    string `json:"engineId"`
	EngineName  string `json:"engineName"`
	EnginePath  string `json:"enginePath"`
	ConnectedAt string `json:"connectedAt"`
}

type LogEntryItem struct {
	Timestamp string `json:"timestamp"`
	Dir       int8   `json:"dir"`
	Message   string `json:"message"`
}

type DebugService struct {
	registry *registry.Registry
}

func (s *DebugService) ListConnections() []ConnectionInfo {
	conns := s.registry.ListConnections()
	result := make([]ConnectionInfo, len(conns))
	for i, c := range conns {
		result[i] = ConnectionInfo{
			ID:          c.ID,
			EngineID:    c.EngineID,
			EngineName:  c.EngineName,
			EnginePath:  c.EnginePath,
			ConnectedAt: c.ConnectedAt.Format(time.RFC3339),
		}
	}
	return result
}

func (s *DebugService) GetLogs(id string) []LogEntryItem {
	logs := s.registry.GetLogs(id)
	result := make([]LogEntryItem, len(logs))
	for i, l := range logs {
		result[i] = LogEntryItem{
			Timestamp: l.Timestamp.Format(time.RFC3339Nano),
			Dir:       int8(l.Dir),
			Message:   l.Message,
		}
	}
	return result
}
