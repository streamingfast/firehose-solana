package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/dfuse-io/dfuse-solana/serumhist"

	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dgrpc"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	*shutter.Shutter

	grpcAddr string
	server   *grpc.Server
	manager  *serumhist.Manager
}

func New(grpcAddr string, manager *serumhist.Manager) *Server {
	return &Server{
		Shutter:  shutter.New(),
		grpcAddr: grpcAddr,
		manager:  manager,
		server:   dgrpc.NewServer(dgrpc.WithLogger(zlog)),
	}
}

func (s *Server) Serve() {
	pbaccounthist.RegisterSerumHistoryServer(s.server, s)
	pbhealth.RegisterHealthServer(s.server, s)

	zlog.Info("listening for serum history",
		zap.String("addr", s.grpcAddr),
	)

	lis, err := net.Listen("tcp", s.grpcAddr)
	if err != nil {
		s.Shutdown(fmt.Errorf("failed listening grpc %q: %w", s.grpcAddr, err))
		return
	}

	if err := s.server.Serve(lis); err != nil {
		s.Shutdown(fmt.Errorf("error on grpcServer.Serve: %w", err))
		return
	}
}

func (s *Server) Terminate(err error) {
	if s.server == nil {
		return
	}

	stopped := make(chan bool)

	// Stop the server gracefully
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	// And don't wait more than 60 seconds for graceful stop to happen
	select {
	case <-time.After(30 * time.Second):
		zlog.Info("gRPC server did not terminate gracefully within allowed time, forcing shutdown")
		s.server.Stop()
	case <-stopped:
		zlog.Info("gRPC server teminated gracefully")
	}
}
