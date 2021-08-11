package grpc

import (
	"time"

	"github.com/dfuse-io/dfuse-solana/serumhist/reader"

	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/streamingfast/dgrpc"
	pbhealth "github.com/streamingfast/pbgo/grpc/health/v1"
	"github.com/streamingfast/shutter"
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

	s.OnTerminating(func(_ error) {
		s.server.Shutdown(30 * time.Second)
	})

	go s.server.Launch(s.grpcAddr)
}
