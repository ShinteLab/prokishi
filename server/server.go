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
	RegisterServiceServer(s)

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

	engines *sync.Map
}

// GRPCサービスを登録
func RegisterServiceServer(r grpc.ServiceRegistrar) *Server {

	var serv Server

	api.RegisterConnectionServiceServer(r, &serv)
	api.RegisterUSISendServiceServer(r, &serv)
	api.RegisterUSIReceiveServiceServer(r, &serv)

	serv.engines = &sync.Map{}
	return &serv
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

// コネクションIDでエンジンを実行し登録する
func (s *Server) startEngine(id string) (string, error) {

	if id == "" {
		return "", fmt.Errorf("EngineId required.")
	}

	e, err := db.SelectEngine(context.Background(), id)
	if err != nil {
		return "", xerrors.Errorf("db.SelectEngine() error: %w", err)
	}
	if e == nil {
		return "", xerrors.Errorf("Engine is Not Found:[%s]", id)
	}

	engine, err := usi.NewSender(e.Path)
	if err != nil {
		return "", xerrors.Errorf("usi.NewSender() error: %w", err)
	}

	//コネクションIDを生成
	uid := uuid.New()
	rtn := uid.String()
	s.engines.Store(rtn, engine)
	return rtn, nil
}

