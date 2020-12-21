package resolvers

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func graphqlErrorFromGRPC(_ context.Context, err error) error {
	code := status.Code(err)
	if code == codes.Canceled {
		return fmt.Errorf("backend error: request cancelled")
	}

	// FIXME: Could we be able to retrieve which top-level operation had the problem exactly? From the `ctx` maybe?
	// At this point, we deal with errors from our side most probably, so log them as error
	zlog.Error("backend error", zap.Error(err))
	if code == codes.DeadlineExceeded {
		return fmt.Errorf("backend error: request timeout")
	}

	return err
}
