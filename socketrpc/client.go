package socketrpc

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type RPCClient struct {
	mu   *sync.Mutex
	conn net.Conn
}

func NewRPCClient() (*RPCClient, error) {
	conn, err := net.Dial("unix", socketAddr)
	if err != nil {
		return nil, err
	}
	c := RPCClient{
		mu:   &sync.Mutex{},
		conn: conn,
	}
	return &c, nil
}

func (c *RPCClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.Close()
	c.conn = nil
}

func (c *RPCClient) SendMessage(cmd byte, jsonMsg string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		log.Fatal("RPC connection does not exist anymore.")
	}
	c.conn.Write([]byte{cmd})
	if jsonMsg != "" {
		c.conn.Write([]byte(jsonMsg))
	}

	c.conn.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1<<14)
	n, err := c.conn.Read(buf)
	if err != nil || n == 0 {
		log.Fatal(err)
	}
	if buf[0] != ErrOk {
		e := fmt.Sprintf("Call returned error code %x\n", buf[0])
		log.Println(e)
		return "", errors.New(e)
	}
	return string(buf[1:n]), nil
}
