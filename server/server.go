package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"github.com/ShinteLab/prokishi/api"
	"github.com/ShinteLab/prokishi/db"
	"github.com/ShinteLab/prokishi/registry"
	"github.com/ShinteLab/prokishi/usi"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
)

// Listen は host:port を bind してリスナを返す。ポート使用中などの
// bind エラーをここで確定させ、呼び出し側が同期的に受け取れるようにする。
func Listen(host string, port int) (net.Listener, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, xerrors.Errorf("net.Listen() error: %w", err)
	}
	return listener, nil
}

func Run(ctx context.Context, host string, port int, opts ...Option) error {
	listener, err := Listen(host, port)
	if err != nil {
		return err
	}
	return Serve(ctx, listener, opts...)
}

// Serve は Listen 済みのリスナ上で gRPC サーバを動かす。ctx の完了か
// SIGINT で GracefulStop する。リスナは戻る際にクローズされる。
func Serve(ctx context.Context, listener net.Listener, opts ...Option) error {

	slog.Info("server listening", "addr", listener.Addr())

	s := grpc.NewServer()

	merged := &Config{}
	for _, opt := range opts {
		opt(merged)
	}
	RegisterServiceServer(s, merged.Registry, merged.UseAuth)

	serveErr := make(chan error, 1)
	go func() {
		err := s.Serve(listener)
		if err != nil {
			log.Printf("Serve() error: %v", err)
		}
		serveErr <- err
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	defer signal.Stop(quit)
	select {
	case <-ctx.Done():
	case <-quit:
	case err := <-serveErr:
		// Serve が自発的に落ちた場合はその原因を返す。
		if err != nil {
			return xerrors.Errorf("Serve() error: %w", err)
		}
		return nil
	}

	s.GracefulStop()
	return nil
}


type Server struct {
	api.ConnectionServiceServer
	api.USISendServiceServer
	api.USIReceiveServiceServer

	engines  *sync.Map
	registry *registry.Registry
	useAuth  bool
}

// GRPCサービスを登録
func RegisterServiceServer(r grpc.ServiceRegistrar, reg *registry.Registry, useAuth bool) *Server {

	serv := &Server{
		engines:  &sync.Map{},
		registry: reg,
		useAuth:  useAuth,
	}

	api.RegisterConnectionServiceServer(r, serv)
	api.RegisterUSISendServiceServer(r, serv)
	api.RegisterUSIReceiveServiceServer(r, serv)

	return serv
}

// 認証
func (s *Server) verifyAuthentication(code string) bool {

	if !s.useAuth {
		return true
	}

	ctx := context.Background()
	c, err := db.SelectCode(ctx, code)
	if err != nil {
		slog.Error(err.Error())
		return false
	}

	if c == nil {
		slog.Warn(fmt.Sprintf("code not found:[%s]", code))
		return false
	}
	return true
}

// コネクションIDでエンジンを実行し登録する。
func (s *Server) startEngine(engineID string) (connID string, engineName string, enginePath string, pid int, err error) {

	if engineID == "" {
		return "", "", "", 0, fmt.Errorf("EngineId required.")
	}

	e, dbErr := db.SelectEngine(context.Background(), engineID)
	if dbErr != nil {
		return "", "", "", 0, xerrors.Errorf("db.SelectEngine() error: %w", dbErr)
	}
	if e == nil {
		return "", "", "", 0, xerrors.Errorf("Engine is Not Found:[%s]", engineID)
	}

	engine, newErr := usi.NewSender(e.Path)
	if newErr != nil {
		return "", "", "", 0, xerrors.Errorf("usi.NewSender() error: %w", newErr)
	}

	uid := uuid.New()
	connID = uid.String()
	s.engines.Store(connID, engine)
	return connID, e.Name, e.Path, engine.Pid(), nil
}

