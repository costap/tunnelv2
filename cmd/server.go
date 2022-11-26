package cmd

import (
	"crypto/tls"
	"fmt"
	tunnelv1 "github.com/costap/tunnelv2/internal/pkg/proto/tunnel/v1"
	"github.com/costap/tunnelv2/internal/pkg/server/tcp"
	tunnel2 "github.com/costap/tunnelv2/internal/pkg/server/tunnel"
	"github.com/jzelinskie/cobrautil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	tcpPort    = 8080
	grpcPort   = 9000
	listenHost = ""
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Starts the tunnel server component",
	Long: `Exposes two endpoints, a public one and a private. Any connections on the public 
are proxy to any clients.`,
	Run: serveRun,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().IntVar(&tcpPort, "tcp-port", tcpPort, "public port to listen to")
	serverCmd.Flags().IntVar(&grpcPort, "grpc-port", grpcPort, "private port to listen to")
}

type server struct {
	logger        *zap.Logger
	tunnelService *tunnel2.Service
	controller    *tunnel2.Controller

	grpcServer *grpc.Server
	tcpServer  *tcp.Server
}

func (s *server) startGRPC(address string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		s.logger.Info("Stopped gRPC server")
	}()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		s.logger.Fatal("Failed to start gRPC server", zap.Error(err))
		return
	}

	s.grpcServer = grpc.NewServer()
	tunnelv1.RegisterTunnelServiceServer(s.grpcServer, s.tunnelService)

	s.logger.Info("Starting gRPC server", zap.String("address", listener.Addr().String()))
	if err := s.grpcServer.Serve(listener); err != nil {
		s.logger.Fatal("Failed to serve gRPC server", zap.Error(err))
		return
	}
}

func (s *server) startTcp(address string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		s.logger.Warn("Stopped HTTP server")
	}()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		s.logger.Fatal("Failed to create TCP server", zap.Error(err))
	}
	s.tcpServer = tcp.NewServer()
	s.logger.Info("Server is running...", zap.String("address", listener.Addr().String()))
	if err := s.tcpServer.Serve(listener, s.controller); err != nil {
		s.logger.Fatal("Failed to start TCP server", zap.Error(err))
	}
}

func (s *server) run(tcpAddr, grpcAddr string) {
	var wg sync.WaitGroup
	wg.Add(2)
	go s.startTcp(tcpAddr, &wg)
	go s.startGRPC(grpcAddr, &wg)
	wg.Wait()
}

func (s *server) stop() {
	s.grpcServer.Stop()
	s.logger.Info("GRPC server stopped")
	s.logger.Info("TCP server stopped", zap.Error(s.tcpServer.Close()))
}

func serveRun(cmd *cobra.Command, args []string) {
	logger := newZapLogger(cobrautil.MustGetBool(cmd, "debug"))
	logger.Info("Server is starting...", zap.String("Version", GetVersion(false)))

	ts := tunnel2.NewService(logger)
	s := server{logger: logger, tunnelService: ts, controller: tunnel2.NewController(logger, ts)}

	tcpPort := cobrautil.MustGetInt(cmd, "tcp-port")
	grpcPort := cobrautil.MustGetInt(cmd, "grpc-port")

	go s.run(fmt.Sprintf(":%v", tcpPort), fmt.Sprintf(":%v", grpcPort))
	defer s.stop()

	// Wait for the process to be shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

func newZapLogger(debug bool) *zap.Logger {
	zap.NewProductionConfig()
	encoderConfig := zapcore.EncoderConfig{
		LevelKey:       "level",
		MessageKey:     "msg",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "file",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(os.Stdout),
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			if debug {
				return true
			}
			return lvl > zapcore.DebugLevel
		}))
	return zap.New(core)
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load server's certificate and private key
	serverCert, err := tls.LoadX509KeyPair("cert/server-cert.pem", "cert/server-key.pem")
	if err != nil {
		return nil, err
	}

	// Create the credentials and return it
	config := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.NoClientCert,
	}

	return credentials.NewTLS(config), nil
}
