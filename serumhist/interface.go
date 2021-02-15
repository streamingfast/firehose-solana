package serumhist

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

type serumEvent interface {
}

type EventWriter interface {
	Write(e serumEvent)
}

type CheckpointResolver func(ctx context.Context) (*pbserumhist.Checkpoint, error)
