package client

import (
	"fmt"
	"go.uber.org/zap"
	"net"
	"sync"
)

type ConnectionHandler struct {
	log          *zap.Logger
	connectionId string
	target       string
	conn         net.Conn
	in, out      chan []byte
	done         chan struct{}
}

func NewConnectionHandler(log *zap.Logger, connectionId, target string, in, out chan []byte) *ConnectionHandler {
	return &ConnectionHandler{log: log, connectionId: connectionId, target: target, in: in, out: out}
}

func (h *ConnectionHandler) Run() error {
	h.done = make(chan struct{})
	conn, err := net.Dial("tcp", h.target)
	if err != nil {
		return fmt.Errorf("cannot connect to target: %w", err)
	}
	h.log.Debug("connected to target")
	h.conn = conn
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				h.log.Error("cannot read from connection", zap.Error(err))
				break
			}
			h.log.Info("read from connection", zap.String("connectionId", h.connectionId), zap.ByteString("data", buf[:n]))
			h.out <- buf[:n]
		}
		wg.Done()
	}()
	go func() {
		for {
			select {
			case <-h.done:
				wg.Done()
				return
			case data := <-h.in:
				_, err := conn.Write(data)
				if err != nil {
					h.log.Error("cannot write to connection", zap.Error(err))
					wg.Done()
					return
				}
			}
		}
	}()
	wg.Wait()
	return nil
}

func (h *ConnectionHandler) Close() error {
	close(h.done)
	return h.conn.Close()
}
