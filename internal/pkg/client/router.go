package client

import (
	"context"
	tunnelv1 "github.com/costap/tunnelv2/internal/pkg/proto/tunnel/v1"
	"go.uber.org/zap"
	"io"
)

type Router struct {
	log         *zap.Logger
	client      tunnelv1.TunnelServiceClient
	target      string
	connections map[string]*ConnectionHandler
}

func NewRouter(log *zap.Logger, client tunnelv1.TunnelServiceClient, target string) *Router {
	return &Router{log: log, client: client, target: target, connections: make(map[string]*ConnectionHandler)}
}

func (r *Router) Start(ctx context.Context) error {
	stream, err := r.client.Tunnel(ctx)
	if err != nil {
		r.log.Fatal("cannot create tunnel: ", zap.Error(err))
	}
	waitc := make(chan struct{})
	go func() {
		for {
			r.log.Debug("waiting for message")
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return
			}
			switch in.Type {
			case tunnelv1.ResponseType_OPEN_CONNECTION:
				r.log.Debug("received open connection")
				if _, ok := r.connections[in.ConnectionId]; !ok {
					out := make(chan []byte)
					r.connections[in.ConnectionId] = NewConnectionHandler(r.log, in.ConnectionId, r.target, make(chan []byte), out)
					go r.connections[in.ConnectionId].Run()
					go func() {
						for {
							select {
							case <-ctx.Done():
								return
							case <-r.connections[in.ConnectionId].done:
								return
							case data := <-out:
								err := stream.Send(&tunnelv1.TunnelRequest{
									ConnectionId: in.ConnectionId,
									Data:         data,
								})
								if err != nil {
									r.log.Fatal("Failed to send data", zap.Error(err))
								}
							}
						}
					}()
				}
				if in.Data != nil && len(in.Data) > 0 {
					r.connections[in.ConnectionId].in <- in.Data
				}
			case tunnelv1.ResponseType_DATA_RECEIVE:
				r.log.Debug("received data")
				if c, ok := r.connections[in.ConnectionId]; ok {
					c.in <- in.Data
				}
			case tunnelv1.ResponseType_CLOSE_CONNECTION:
				r.log.Debug("received close connection")
				if c, ok := r.connections[in.ConnectionId]; ok {
					c.Close()
					delete(r.connections, in.ConnectionId)
				}
			}
			if err != nil {
				r.log.Fatal("Failed to receive a note", zap.Error(err))
			}
			r.log.Info("received", zap.String("connectionId", in.ConnectionId), zap.ByteString("data", in.Data))
		}
	}()
	<-waitc
	return nil
}
