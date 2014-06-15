package hoop

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/nyushi/traproxy"
)

// Protocol definition
const (
	TCP = iota
	UDP = iota
)

type closer interface {
	Close() error
}

type Hoop struct {
	ListenPort int
	Proto      int
	Remote     string
	conn       io.Closer
	running    bool
}

func NewHoop(port, proto int, remote string) *Hoop {
	h := &Hoop{}
	h.running = true
	h.Proto = proto
	h.ListenPort = port
	h.Remote = remote
	return h
}

func ProtoString(p int) string {
	switch p {
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	}
	return ""
}

func (h *Hoop) Start() error {
	switch h.Proto {
	case TCP:
		return h.startTCP()
	case UDP:
		return h.startUDP()
	}
	return errors.New("unknown protocol")
}

func (h *Hoop) startTCP() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", h.ListenPort))
	h.conn = l
	if err != nil {
		return err
	}
	go func() {
		for h.running {
			l := h.conn.(net.Listener)
			c, err := l.Accept()
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

func (h *Hoop) startUDP() error {
	laddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", h.ListenPort))
	l, err := net.ListenUDP("udp", laddr)
	h.conn = l
	if err != nil {
		return err
	}

	r, err := net.Dial("udp", h.Remote)
	if err != nil {
		return err
	}
	go func() {
		for h.running {
			buf := make([]byte, 65507)
			rsize, err := l.Read(buf)
			if err != nil {
				continue
			}
			r.Write(buf[0:rsize])
		}
	}()
	return nil
}
func (h *Hoop) Stop() {
	h.running = false
	h.conn.Close()
}
