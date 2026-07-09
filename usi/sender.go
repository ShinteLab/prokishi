package usi

import (
	"bufio"
	"io"
	"log"
	"os/exec"
	"sync"

	"golang.org/x/xerrors"
)

// サーバ上で動作し、エンジンの起動、監視を行う
// 実際のエンジンにデータを渡す(Send)
// また標準出力を監視して、チャンネルに渡す
type Sender struct {
	cmd          *exec.Cmd
	out          io.ReadCloser
	in           io.WriteCloser
	terminated   bool
	terminateMtx sync.Mutex
	OutCh        chan string
}

// エンジンを起動し、監視を開始
func NewSender(e string) (*Sender, error) {

	cmd := exec.Command(e)

	s := new(Sender)
	s.OutCh = make(chan string)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, xerrors.Errorf("StdoutPipe() error: %w")
	}

	s.out = out
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, xerrors.Errorf("StdinPipe() error: %w")
	}

	s.in = in
	s.cmd = cmd
	err = cmd.Start()
	if err != nil {
		return nil, xerrors.Errorf("exec.Start() error: %w", err)
	}

	go s.monitorStdout()

	go func() {
		//プロセス終了を監視
		err := cmd.Wait()
		if err != nil {
			log.Println(err)
		}
		s.Terminate()
	}()

	return s, nil
}

// エンジンの標準入力に送信
func (s *Sender) Send(line string) error {
	_, err := s.in.Write([]byte(line + "\n"))
	if err != nil {
		return xerrors.Errorf("exec.Start() error: %w", err)
	}
	return nil
}

// エンジンが出力した出力をチャンネルに書き込む
func (s *Sender) monitorStdout() {

	scanner := bufio.NewScanner(s.out)

	for scanner.Scan() {
		if s.OutCh != nil {
			s.OutCh <- scanner.Text()
		}
	}
}

func (s *Sender) Pid() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}
	return 0
}

// 終了処理
func (s *Sender) Terminate() error {

	s.terminateMtx.Lock()
	defer s.terminateMtx.Unlock()

	if s.terminated {
		return nil
	}
	s.terminated = true

	//基本的に閉じててエラーが出るがとりあえず
	s.out.Close()
	s.in.Close()

	ch := s.OutCh
	s.OutCh = nil
	close(ch)

	return nil
}
