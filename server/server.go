package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"prokishi/api"
	"prokishi/db"
	"prokishi/registry"
	"prokishi/usi"
	"strconv"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
)

func Run(ctx context.Context, host string, port int, opts ...Option) error {

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return xerrors.Errorf("net.Listen() error: %w", err)
	}

	fmt.Println("IP Address ====================")
	printIPs()
	fmt.Println("===============================")
	fmt.Println("Listener Address:", listener.Addr())

	s := grpc.NewServer()

	var reg *registry.Registry
	for _, opt := range opts {
		cfg := &Config{}
		opt(cfg)
		if cfg.Registry != nil {
			reg = cfg.Registry
		}
	}
	RegisterServiceServer(s, reg)

	go func() {
		err := s.Serve(listener)
		if err != nil {
			log.Printf("Serve() error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	select {
	case <-ctx.Done():
	case <-quit:
	}

	s.GracefulStop()
	return nil
}

func printIPs() {
	ift, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, ifi := range ift {
		addrs, err := ifi.Addrs()
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, addr := range addrs {
			ip := getIP(addr)
			if !ip.IsLoopback() {
				fmt.Printf("%v\n", ip)
			}
		}
	}
}

func getIP(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	return ip
}

type Server struct {
	api.ConnectionServiceServer
	api.USISendServiceServer
	api.USIReceiveServiceServer

	engines  *sync.Map
	registry *registry.Registry
}

// GRPCサービスを登録
func RegisterServiceServer(r grpc.ServiceRegistrar, reg *registry.Registry) *Server {

	serv := &Server{
		engines:  &sync.Map{},
		registry: reg,
	}

	api.RegisterConnectionServiceServer(r, serv)
	api.RegisterUSISendServiceServer(r, serv)
	api.RegisterUSIReceiveServiceServer(r, serv)

	return serv
}

// 認証
func (s *Server) verifyAuthentication(code string) bool {

	ctx := context.Background()
	cnt, err := db.CountCodes(ctx)
	if err != nil {
		slog.Error(err.Error())
		return false
	}

	if cnt == 0 {
		return true
	}

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

// コネクションIDでエンジンを実行し登録する。connID と enginePath を返す。
func (s *Server) startEngine(engineID string) (connID string, enginePath string, err error) {

	if engineID == "" {
		return "", "", fmt.Errorf("EngineId required.")
	}

	e, dbErr := db.SelectEngine(context.Background(), engineID)
	if dbErr != nil {
		return "", "", xerrors.Errorf("db.SelectEngine() error: %w", dbErr)
	}
	if e == nil {
		return "", "", xerrors.Errorf("Engine is Not Found:[%s]", engineID)
	}

	engine, newErr := usi.NewSender(e.Path)
	if newErr != nil {
		return "", "", xerrors.Errorf("usi.NewSender() error: %w", newErr)
	}

	uid := uuid.New()
	connID = uid.String()
	s.engines.Store(connID, engine)
	return connID, e.Path, nil
}

