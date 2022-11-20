package tunnel

import (
	"context"
	"github.com/costap/tunnelv2/internal/pkg/proto/tunnel/v1"
	"go.uber.org/zap"
)

type frame struct {
	id   string
	data []byte
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
		log: log,
	}
}

func (s *Service) TunnelConnection(ctx context.Context, conn connection) {
	s.connections[conn.id] = conn
	go func() {
		for {
			select {
			case data := <-conn.output:
				s.output <- frame{id: conn.id, data: data}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) Tunnel(stream tunnelv1.TunnelService_TunnelServer) error {
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				s.log.Error("Failed to read from stream", zap.Error(err))
				return
			}
			s.connections[msg.ConnectionId].input <- msg.Data
		}
	}()
	go func() {
		for {
			frame := <-s.output
			err := stream.Send(&tunnelv1.TunnelResponse{ConnectionId: frame.id, Data: frame.data})
			if err != nil {
				s.log.Error("Failed to write to stream", zap.Error(err))
				return
			}
		}
	}()
	return nil
}
