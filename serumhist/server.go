package serumhist

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dfuse-io/solana-go"

	"github.com/dfuse-io/logging"

	pbaccounthist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
	"github.com/dfuse-io/dgrpc"
	pbhealth "github.com/dfuse-io/pbgo/grpc/health/v1"
	"github.com/dfuse-io/shutter"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type InjectorServer struct {
	*shutter.Shutter

	manager  *OrderManager
	grpcAddr string
	server   *grpc.Server
}

func New(grpcAddr string, manager *OrderManager) *InjectorServer {
	return &InjectorServer{
		Shutter:  shutter.New(),
		grpcAddr: grpcAddr,
		manager:  manager,
		server:   dgrpc.NewServer(dgrpc.WithLogger(zlog)),
	}
}

func (s *InjectorServer) Serve() {
	pbaccounthist.RegisterSerumOrderTrackerServer(s.server, s)
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

func (s *InjectorServer) Terminate(err error) {
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

func (s *InjectorServer) TrackOrder(r *pbaccounthist.TrackOrderRequest, stream pbaccounthist.SerumOrderTracker_TrackOrderServer) error {
	ctx := stream.Context()
	logger := logging.Logger(ctx, zlog)
	logger.Debug("tracking order", zap.Reflect("request", r))

	market, err := solana.PublicKeyFromBase58(r.Market)
	if err != nil {
		return fmt.Errorf("unable to decode market key")
	}

	subscription, err := s.manager.subscribe(r.OrderId, market, logger)
	if err != nil {
		return fmt.Errorf("unable to create subscription: %w", err)
	}
	defer s.manager.unsubscribe(ctx, subscription)

	for {
		select {
		case <-ctx.Done():
			return nil
		case resp, opened := <-subscription.conn:
			if !opened {
				// we've been shutdown somehow, simply close the current connection.
				// we'll have logged at the source\
				return nil
			}
			logger.Debug("sending order transition",
				zap.Stringer("current_state", resp.CurrentState),
				zap.Stringer("previous_state", resp.PreviousState),
				zap.Stringer("transition", resp.Transition),
			)

			err := stream.Send(resp)
			if err != nil {
				logger.Info("failed writing to socket, shutting down subscription", zap.Error(err))
				return err
			}
		}
	}
}

func (s *InjectorServer) Check(ctx context.Context, request *pbhealth.HealthCheckRequest) (*pbhealth.HealthCheckResponse, error) {
	panic("implement me")
}
