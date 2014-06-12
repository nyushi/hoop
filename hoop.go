package hoop

import (
	"fmt"
	"log"
	"net"

	"github.com/nyushi/traproxy"
)

type Hoop struct {
	ListenPort int
	Remote     string
	Listener   net.Listener
	running    bool
}

func NewHoop(port int, remote string) *Hoop {
	h := &Hoop{}
	h.running = true
	h.ListenPort = port
	h.Remote = remote
	return h
}

func (h *Hoop) Start() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", h.ListenPort))
	h.Listener = l
	if err != nil {
		return err
	}
	go func() {
		for h.running {
			c, err := h.Listener.Accept()
			if err != nil {
				continue
			}
			r, err := net.Dial("tcp", h.Remote)
			if err != nil {
				c.Close()
				continue
			}
			a, ok := c.(*net.TCPConn)
			if !ok {
				c.Close()
				r.Close()
			}
			b, ok := r.(*net.TCPConn)
			if !ok {
				c.Close()
				r.Close()
			}
			log.Printf("new connection created %d -> %s", h.ListenPort, h.Remote)
			go traproxy.Pipe(a, b, nil)
			go traproxy.Pipe(b, a, nil)
		}
	}()
	return nil
}

func (h *Hoop) Stop() {
	h.running = false
	h.Listener.Close()
}
