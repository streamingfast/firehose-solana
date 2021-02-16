package bqloader

import (
	"context"

	pbserumhist "github.com/dfuse-io/dfuse-solana/pb/dfuse/solana/serumhist/v1"
)

func (bq *BQLoader) GetCheckpoint(ctx context.Context) (*pbserumhist.Checkpoint, error) {
	// TODO: we need to walk the file on google storage and figure out where to start...
	panic("implement me!")
}

//gs://dfuseio-global-billing-us/billable-events/2019-06-18-14-17-05-2071456468184800893.avro
// gs://dfuseio-global-..../meta/sol-mainnet/serum-fills/<SLOT_NUM_START>-<SLOT_NUM_END>-<SLOT_ID_START>-<SLOT_ID_END>-timestamp?.avro -> 28
// gs://dfuseio-global-..../meta/sol-mainnet/serum-orders/<SLOT_NUM_START>-<SLOT_NUM_END>-<SLOT_ID_START>-<SLOT_ID_END>-timestamp?.avro -> 35
// gs://dfuseio-global-..../meta/sol-mainnet/serum-traders/<SLOT_NUM_START>-<SLOT_NUM_END>-<SLOT_ID_START>-<SLOT_ID_END>-timestamp?.avro -> 22
