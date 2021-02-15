package bigq

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

func (b *Bigq) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	panic("implement me")
}
