package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/costap/tunnelv2/internal/pkg/client"
	tunnelv1 "github.com/costap/tunnelv2/internal/pkg/proto/tunnel/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	serverAddress = "localhost:9000"
	targetAddress = "jsa-admin.thewindgod.com:80"
)

// clientCmd represents the client command
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Start a client deamon",
	RunE:  clientRun,
}

func init() {
	rootCmd.AddCommand(clientCmd)

	clientCmd.Flags().StringVarP(&serverAddress, "server", "s", serverAddress, "server address")
	clientCmd.Flags().StringVarP(&targetAddress, "target", "t", targetAddress, "server address")
}

func clientRun(cmd *cobra.Command, args []string) error {
	log := newZapLogger(debug)
	tlsCredentials, err := loadClientTLSCredentials()
	if err != nil {
		log.Fatal("cannot load TLS credentials: ", zap.Error(err))
	}

	cc1, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(tlsCredentials))
	defer cc1.Close()
	if err != nil {
		log.Fatal("cannot dial server: ", zap.Error(err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tc := tunnelv1.NewTunnelServiceClient(cc1)
	r := client.NewRouter(log, tc, targetAddress)
	if err := r.Start(ctx); err != nil {
		return fmt.Errorf("cannot start router: %w", err)
	}
	return nil
}

func loadClientTLSCredentials() (credentials.TransportCredentials, error) {
	// Load certificate of the CA who signed server's certificate
	pemServerCA, err := os.ReadFile("cert/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to add server CA's certificate")
	}

	// Create the credentials and return it
	config := &tls.Config{
		RootCAs: certPool,
	}

	return credentials.NewTLS(config), nil
}
