package server_test

import (
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"shinte/prokishi/api"
	"shinte/prokishi/db"
	"shinte/prokishi/internal/testfakeengine"
	"shinte/prokishi/registry"
	"shinte/prokishi/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// chdirTemp creates a fresh temporary directory, changes the process's
// working directory into it, and restores the original working directory
// (plus closes the DB handle) once the test finishes. This mirrors the
// pattern in db/testhelper_test.go; that helper is unexported to package db
// and can't be imported here (and this file is package server_test, so it
// can't reuse server/server_internal_test.go's copy either), so it's
// replicated locally.
func chdirTemp(t *testing.T) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir() error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("os.Chdir(restore) error: %v", err)
		}
	})
}

func setupTestDB(t *testing.T) {
	t.Helper()

	chdirTemp(t)

	if err := db.Init(true); err != nil {
		t.Fatalf("db.Init() error: %v", err)
	}
	if err := db.Open(true); err != nil {
		t.Fatalf("db.Open() error: %v", err)
	}
}

// testServer wires a server.Server up to an in-process bufconn listener and
// dials a client connection against it, returning the three generated
// service clients plus the registry the server was constructed with and a
// cleanup func (also registered via t.Cleanup).
type testServer struct {
	conn *grpc.ClientConn
	reg  *registry.Registry

	connClient    api.ConnectionServiceClient
	sendClient    api.USISendServiceClient
	receiveClient api.USIReceiveServiceClient
}

func newTestServer(t *testing.T, useAuth bool) *testServer {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()

	reg := registry.New()
	server.RegisterServiceServer(grpcServer, reg, useAuth)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(grpcServer.Stop)

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return lis.DialContext(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &testServer{
		conn:          conn,
		reg:           reg,
		connClient:    api.NewConnectionServiceClient(conn),
		sendClient:    api.NewUSISendServiceClient(conn),
		receiveClient: api.NewUSIReceiveServiceClient(conn),
	}
}

// registerEngine inserts an auth code and an engine (pointed at a real,
// spawnable fake-engine binary) into the temp DB and returns their IDs.
func registerFixtures(t *testing.T) (code string, engineID string) {
	t.Helper()

	const authCode = "test-auth-code"
	if err := db.InsertCode(authCode, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	binPath := testfakeengine.Build(t)
	const id = "test-engine-id"
	if err := db.InsertEngine(id, binPath, "Test Engine"); err != nil {
		t.Fatalf("db.InsertEngine() error: %v", err)
	}

	return authCode, id
}

func TestHandler_ConnectionAuthFailure(t *testing.T) {
	setupTestDB(t)
	_, engineID := registerFixtures(t)

	ts := newTestServer(t, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ts.connClient.Connection(ctx, &api.ConnectionRequest{
		Code:     "wrong-code",
		EngineId: engineID,
	})
	if err == nil {
		t.Fatalf("Connection() with wrong code: error = nil, want error")
	}
	if !strings.Contains(err.Error(), "can not be verified") {
		t.Fatalf("Connection() with wrong code: error = %v, want message containing %q", err, "can not be verified")
	}

	if got := len(ts.reg.ListConnections()); got != 0 {
		t.Fatalf("registry has %d connections after a failed auth, want 0", got)
	}
}

// TestHandler_FullFlow drives the full Connection -> Send -> Receive ->
// quit cleanup flow through real gRPC calls (via bufconn) against a real
// spawned fake-engine subprocess, exercising the same path a real USI GUI
// <-> engine round trip would take.
func TestHandler_FullFlow(t *testing.T) {
	setupTestDB(t)
	code, engineID := registerFixtures(t)

	ts := newTestServer(t, true)

	// --- Connection ---
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelConnect()

	connResp, err := ts.connClient.Connection(connectCtx, &api.ConnectionRequest{
		Code:     code,
		EngineId: engineID,
	})
	if err != nil {
		t.Fatalf("Connection() error: %v", err)
	}
	connID := connResp.ConnectionId
	if connID == "" {
		t.Fatalf("Connection() returned empty ConnectionId")
	}

	conns := ts.reg.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("registry has %d connections after Connection(), want 1", len(conns))
	}
	if conns[0].ID != connID {
		t.Fatalf("registry connection ID = %q, want %q", conns[0].ID, connID)
	}
	if conns[0].EngineID != engineID {
		t.Fatalf("registry connection EngineID = %q, want %q", conns[0].EngineID, engineID)
	}

	// --- Receive stream (opened before sending, so we don't race the
	// engine's output against subscribing to it) ---
	streamCtx, cancelStream := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStream()

	stream, err := ts.receiveClient.Receive(streamCtx, &api.ReceiveRequest{
		Code:         code,
		ConnectionId: connID,
	})
	if err != nil {
		t.Fatalf("Receive() error: %v", err)
	}

	type recvResult struct {
		resp *api.ReceiveResponse
		err  error
	}
	recvCh := make(chan recvResult, 8)
	go func() {
		for {
			resp, err := stream.Recv()
			recvCh <- recvResult{resp, err}
			if err != nil {
				return
			}
		}
	}()

	// --- Send a command, verify it round-trips through the fake engine ---
	sendCtx, cancelSend := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelSend()

	if _, err := ts.sendClient.Send(sendCtx, &api.SendRequest{
		Code:         code,
		ConnectionId: connID,
		Cmd:          "position startpos",
	}); err != nil {
		t.Fatalf("Send(position startpos) error: %v", err)
	}

	select {
	case r := <-recvCh:
		if r.err != nil {
			t.Fatalf("stream.Recv() error: %v", r.err)
		}
		want := "echo: position startpos"
		if r.resp.Cmd != want {
			t.Fatalf("stream.Recv() Cmd = %q, want %q", r.resp.Cmd, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for echoed response on Receive stream")
	}

	// --- quit: the handler special-cases this to remove the connection
	// from the engine map and registry after forwarding it to the engine ---
	quitCtx, cancelQuit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelQuit()

	if _, err := ts.sendClient.Send(quitCtx, &api.SendRequest{
		Code:         code,
		ConnectionId: connID,
		Cmd:          "quit",
	}); err != nil {
		t.Fatalf("Send(quit) error: %v", err)
	}

	// The fake engine exits on "quit" without echoing it, which closes its
	// stdout pipe; usi.Sender then closes OutCh, which ends the server's
	// Receive loop (range over a closed channel) and the stream itself.
	select {
	case r := <-recvCh:
		if r.err != io.EOF {
			t.Fatalf("stream.Recv() after quit = (%v, %v), want io.EOF", r.resp, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Receive stream to close after quit")
	}

	// Cleanup on quit should have removed the connection from the registry.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if len(ts.reg.ListConnections()) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("registry still has %d connections after quit, want 0", len(ts.reg.ListConnections()))
		}
		time.Sleep(10 * time.Millisecond)
	}

	// A subsequent Send on the now-removed connection should fail.
	postQuitCtx, cancelPostQuit := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelPostQuit()

	if _, err := ts.sendClient.Send(postQuitCtx, &api.SendRequest{
		Code:         code,
		ConnectionId: connID,
		Cmd:          "isready",
	}); err == nil {
		t.Fatalf("Send() after quit: error = nil, want error (connection should be gone)")
	} else if !strings.Contains(err.Error(), "connectionId is failed") {
		t.Fatalf("Send() after quit: error = %v, want message containing %q", err, "connectionId is failed")
	}
}
