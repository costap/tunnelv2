package client

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
	"sync"
)

// ConnectionHandler is a handler for a single connection
type ConnectionHandler struct {
	log          *zap.Logger
	connectionId string
	target       string
	in, out      chan []byte
	done         chan struct{}
	running      uint32
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(log *zap.Logger, connectionId, target string, in, out chan []byte) *ConnectionHandler {
	return &ConnectionHandler{log: log, connectionId: connectionId, target: target, in: in, out: out, running: 0}
}

// Run starts the connection handler
func (h *ConnectionHandler) Run() error {
	h.running += 1
	h.log.Debug("starting connection handler", zap.String("connectionId", h.connectionId))
	h.done = make(chan struct{})
	conn, err := net.Dial("tcp", h.target)
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("cannot connect to target: %w", err)
	}
	h.log.Debug("connected to target")
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err == io.EOF {
				h.log.Debug("connection closed by remote")
				break
			}
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
	h.log.Debug("connection handler stopped", zap.String("connectionId", h.connectionId))
	h.running -= 1
	return nil
}

func (h *ConnectionHandler) Close() {
	close(h.done)
}
