package grpc

import (
	"context"

	pbhealth "github.com/streamingfast/pbgo/grpc/health/v1"
)

func (s *Server) Check(ctx context.Context, request *pbhealth.HealthCheckRequest) (*pbhealth.HealthCheckResponse, error) {
	status := pbhealth.HealthCheckResponse_SERVING

	if s.IsTerminating() {
		status = pbhealth.HealthCheckResponse_NOT_SERVING
	}

	return &pbhealth.HealthCheckResponse{
		Status: status,
	}, nil

}
