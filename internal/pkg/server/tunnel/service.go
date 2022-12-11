package tunnel

import (
	"context"
	"github.com/costap/tunnelv2/internal/pkg/proto/tunnel/v1"
	"go.uber.org/zap"
	"sync"
)

type action int

const (
	action_open action = iota
	action_close
	action_data
)

type frame struct {
	id     string
	data   []byte
	action action
}
type connection struct {
	id            string
	input, output chan []byte
}
type Service struct {
	log         *zap.Logger
	connections map[string]connection
	output      chan frame
}

func NewService(log *zap.Logger) *Service {
	return &Service{
		log:         log,
		connections: make(map[string]connection),
		output:      make(chan frame),
	}
}

func (s *Service) TunnelConnection(ctx context.Context, conn connection) {
	s.connections[conn.id] = conn
	s.output <- frame{id: conn.id, action: action_open}
	go func() {
		for {
			select {
			case data := <-conn.input:
				s.output <- frame{id: conn.id, data: data, action: action_data}
			case <-ctx.Done():
				s.output <- frame{id: conn.id, action: action_close}
				return
			}
		}
	}()
}

func (s *Service) Tunnel(stream tunnelv1.TunnelService_TunnelServer) error {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				s.log.Error("Failed to read from stream", zap.Error(err))
				break
			}
			s.connections[msg.ConnectionId].output <- msg.Data
		}
		wg.Done()
	}()
	go func() {
		for {
			frame := <-s.output
			var rt tunnelv1.ResponseType
			switch frame.action {
			case action_open:
				rt = tunnelv1.ResponseType_OPEN_CONNECTION
			case action_close:
				rt = tunnelv1.ResponseType_CLOSE_CONNECTION
			case action_data:
				rt = tunnelv1.ResponseType_DATA_RECEIVE
			}
			err := stream.Send(&tunnelv1.TunnelResponse{ConnectionId: frame.id, Data: frame.data, Type: rt})
			if err != nil {
				s.log.Error("Failed to write to stream", zap.Error(err))
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()
	return nil
}
