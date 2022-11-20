package tcp

import (
	"context"
	"net"
)

type Server struct {
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Serve(l net.Listener, handler Handler) error {

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go handler.Handle(context.Background(), conn)
		}
	}()
	return nil
}

func (s *Server) Close() error {
	return nil
}
