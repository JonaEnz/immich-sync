package socketrpc

import (
	"log"
	"net"
	"time"
)

func SendMessage(cmd byte, jsonMsg string) {
	c, err := net.Dial("unix", socketAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte{cmd})
	if jsonMsg != "" {
		c.Write([]byte(jsonMsg))
	}

	c.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 1)
	n, err := c.Read(buf)
	if err != nil || n == 0 {
		log.Fatal(err)
	}
	if buf[0] != ErrOk {
		log.Printf("Call returned error code %x\n", buf[0])
	}
}
