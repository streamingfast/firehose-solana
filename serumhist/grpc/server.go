package grpc

import (
	"fmt"
	"net"
	"time"

	"github.com/dfuse-io/dfuse-solana/serumhist/reader"

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
	server   *dgrpc.Server
	reader   *reader.Reader
}

func New(grpcAddr string, manager *reader.Reader) *Server {
	return &Server{
		Shutter:  shutter.New(),
		grpcAddr: grpcAddr,
		reader:   manager,
		server:   dgrpc.NewServer2(dgrpc.WithLogger(zlog)),
	}
}

func (s *Server) Serve() {
	s.server.RegisterService(func(gs *grpc.Server) {
		pbaccounthist.RegisterSerumHistoryServer(gs, s)
		pbhealth.RegisterHealthServer(gs, s)
	})

	zlog.Info("listening for serum history",
		zap.String("addr", s.grpcAddr),
	)

	s.OnTerminating(func(err error) {
		server.Shutdown(30 * time.Second)
	})

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
