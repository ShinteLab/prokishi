package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"shinte/prokishi/api"
	"shinte/prokishi/db"
	"shinte/prokishi/registry"
	"shinte/prokishi/usi"

	"golang.org/x/xerrors"
)

// コネクションIDを発行
func (s *Server) Connection(ctx context.Context, r *api.ConnectionRequest) (*api.ConnectionResponse, error) {

	if !s.verifyAuthentication(r.Code) {
		return nil, fmt.Errorf("can not be verified")
	}

	if r.Code != "" {
		if err := db.UpdateCode(r.Code); err != nil {
			slog.Warn("UpdateCode failed", "err", err)
		}
	}

	connID, engineName, enginePath, pid, err := s.startEngine(r.EngineId)
	if err != nil {
		return nil, xerrors.Errorf("startEngine() error: %w", err)
	}

	slog.Info(fmt.Sprintf("register Engine map:%s", connID))

	if s.registry != nil {
		s.registry.Add(registry.Connection{
			ID:          connID,
			EngineID:    r.EngineId,
			EngineName:  engineName,
			EnginePath:  enginePath,
			PID:         pid,
			ConnectedAt: time.Now(),
		})
	}

	return &api.ConnectionResponse{ConnectionId: connID}, nil
}

// エンジンに送信
func (s *Server) Send(ctx context.Context, r *api.SendRequest) (*api.SendResponse, error) {

	e, err := s.getEngine(r.Code, r.ConnectionId)
	if err != nil {
		return nil, xerrors.Errorf("getEngine() error: %w", err)
	}

	quit := r.Cmd == "quit"

	slog.Debug(fmt.Sprintf("USI(I):%s", r.Cmd))
	if err = e.Send(r.Cmd); err != nil {
		return nil, xerrors.Errorf("Send() error: %w", err)
	}

	if s.registry != nil {
		s.registry.AppendLog(r.ConnectionId, registry.DirSend, r.Cmd)
	}

	if quit {
		slog.Info(fmt.Sprintf("remove Engine map:%s", r.ConnectionId))
		s.engines.Delete(r.ConnectionId)
		if s.registry != nil {
			s.registry.Remove(r.ConnectionId)
		}
	}

	return &api.SendResponse{}, nil
}

// エンジンの出力をクライアントに送信
func (s *Server) Receive(r *api.ReceiveRequest, stream api.USIReceiveService_ReceiveServer) error {

	e, err := s.getEngine(r.Code, r.ConnectionId)
	if err != nil {
		return xerrors.Errorf("getEngine() error: %w", err)
	}

	for v := range e.OutCh {
		slog.Debug(fmt.Sprintf("USI(O):%s", v))
		if s.registry != nil {
			s.registry.AppendLog(r.ConnectionId, registry.DirRecv, v)
		}
		stream.Send(&api.ReceiveResponse{Cmd: v})
	}

	return nil
}

// エンジン取得
func (s *Server) getEngine(code string, id string) (*usi.Sender, error) {
	if !s.verifyAuthentication(code) {
		return nil, fmt.Errorf("can not be verified")
	}

	e, ok := s.engines.Load(id)
	if !ok {
		return nil, fmt.Errorf("connectionId is failed")
	}
	return e.(*usi.Sender), nil
}
