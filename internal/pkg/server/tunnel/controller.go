package tunnel

import (
	"context"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net"
	"sync"
)

type Tunnel struct {
	input, output chan []byte
}

type Controller struct {
	log     *zap.Logger
	s       *Service
	tunnels map[string]*Tunnel
}

func NewController(log *zap.Logger, service *Service) *Controller {
	return &Controller{log: log, s: service, tunnels: make(map[string]*Tunnel)}
}

func (c *Controller) StartTunnel(input, output chan []byte) (string, error) {
	id := uuid.New().String()
	c.tunnels[id] = &Tunnel{input: input, output: output}
	return id, nil
}

func (c *Controller) StopTunnel() error {
	return nil
}

func (c *Controller) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tConn := connection{id: uuid.New().String(), input: make(chan []byte), output: make(chan []byte)}
	c.s.TunnelConnection(ctx, tConn)

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			tConn.input <- buf[:n]
		}
		wg.Done()
	}()
	go func() {
		for {
			data := <-tConn.output
			_, err := conn.Write(data)
			if err != nil {
				return
			}
		}
		wg.Done()
	}()
	wg.Wait()
}
