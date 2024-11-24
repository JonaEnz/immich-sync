package socketrpc

import (
	"log"
	"net"
	"os"
	"sync"
)

type RPCServer struct {
	mu        *sync.RWMutex
	exit      chan interface{}
	callbacks map[byte]func(string) byte
}

func NewRPCServer() RPCServer {
	s := RPCServer{
		mu:        &sync.RWMutex{},
		exit:      nil,
		callbacks: make(map[byte]func(string) byte),
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
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Started RPC server on socket '%s'", socketAddr)
	go func(socket net.Listener) {
		for {
			conn, _ := socket.Accept()
			go func(conn net.Conn) {
				defer conn.Close()
				buf := make([]byte, 4096)
				n, err := conn.Read(buf)
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
				result := callbackFunc(message)
				conn.Write([]byte{result})
			}(conn)
			select {
			case <-s.exit:
				return
			default:
			}
		}
	}(socket)
}

func (s *RPCServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exit != nil {
		s.exit <- 1
	}
}

func (s *RPCServer) RegisterCallback(cmd byte, f func(string) byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks[cmd] = f
	return nil
}
