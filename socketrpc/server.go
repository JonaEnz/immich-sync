package socketrpc

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type RPCServer struct {
	mu        *sync.RWMutex
	exit      chan interface{}
	callbacks map[byte]func(string) (byte, string)
}

func NewRPCServer() RPCServer {
	s := RPCServer{
		mu:        &sync.RWMutex{},
		exit:      nil,
		callbacks: make(map[byte]func(string) (byte, string)),
	}
	s.callbacks[CmdScanAll] = nil
	s.callbacks[CmdAddDir] = nil
	return s
}

func (s *RPCServer) Start() {
	s.mu.Lock()
	s.exit = make(chan interface{})
	s.mu.Unlock()

	if err := os.RemoveAll(socketAddr); err != nil {
		log.Fatal(err)
	}
	socket, err := net.Listen("unix", socketAddr)
	os.Chmod(socketAddr, 0o777)
	if err != nil {
		log.Fatal(err)
	}
	go func(socket net.Listener) {
		for {
			conn, _ := socket.Accept()
			go func(conn net.Conn) {
				for {
					defer conn.Close()
					buf := make([]byte, 1<<14)
					n, err := conn.Read(buf)
					if err == io.EOF {
						return
					}
					if err != nil || n == 0 {
						conn.Write([]byte{ErrGeneric})
						log.Fatal(err)
					}
					cmd := buf[0]
					message := string(buf[1:n])
					callbackFunc, ok := s.callbacks[cmd]
					if !ok {
						conn.Write([]byte{ErrUnknownCmd})
						return
					}
					if callbackFunc == nil {
						conn.Write([]byte{ErrUnsupportedCmd})
						return
					}
					result, resultString := callbackFunc(message)
					conn.Write([]byte{result})
					conn.Write([]byte(resultString))
				}
			}(conn)
			select {
			case <-s.exit:
				return
			default:
			}
		}
	}(socket)
	log.Printf("Started RPC server on socket '%s'", socketAddr)
}

func (s *RPCServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exit != nil {
		close(s.exit)
	}
}

func (s *RPCServer) RegisterCallback(cmd byte, f func(string) (byte, string)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks[cmd] = f
	return nil
}

func (s *RPCServer) WaitForExit() {
	<-s.exit
}
