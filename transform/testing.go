package transform

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/sf-solana/codec"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	"github.com/stretchr/testify/require"
	"testing"
)

func testBlock(t *testing.T) *bstream.Block {
	blockCnt := `{"id":"YUuA+kkh37zwbZnFHCqPZSr5zToZ68adXvI7VXRCNWw=","number":1,"version":1,"previous_id":"tIdIlLWz0FXu0xbqPczAVPnjQjyGhdWTlBUvG5n5buM=","genesis_unix_timestamp":1634747398,"clock_unix_timestamp":1634747398,"transactions":[{"id":"aXRooAlh0Qcc2iCuv3et6aCerk8H0TpAk0zTI3q2EdJoPjKL77Xo/CTBcuVgC1uPbzGeBsI7iifBkoMs+6NzCA==","header":{"num_required_signatures":1,"num_readonly_unsigned_accounts":3},"account_keys":["lnu9P98huhUMZuA1HU/6J/XUizR9yG0oWQqNe6fmWHc=","IJ5QUyZZH8YHlTnMx5XWaqSMzJ5um4tq3fRHzm8AWXM=","BqfVFxkvCq/G8mXj+3fMetqCxSnQvjsTbi0AVSAAAAA=","BqfVFxjHdMkoVmOYaR1etoteuKObS21cc1VbIQAAAAA=","B2FIHTV0dLt8TXYk69O9s9g1XnPREEP8DaNTgAAAAAA="],"recent_blockhash":"zNM8mdqpnNyqWHQrYHYfFB14t/6wZb8nS5jBGz4qcp0=","instructions":[{"program_id":"B2FIHTV0dLt8TXYk69O9s9g1XnPREEP8DaNTgAAAAAA=","account_keys":["IJ5QUyZZH8YHlTnMx5XWaqSMzJ5um4tq3fRHzm8AWXM=","BqfVFxkvCq/G8mXj+3fMetqCxSnQvjsTbi0AVSAAAAA=","BqfVFxjHdMkoVmOYaR1etoteuKObS21cc1VbIQAAAAA=","lnu9P98huhUMZuA1HU/6J/XUizR9yG0oWQqNe6fmWHc="],"data":"AgAAAAEAAAAAAAAAAAAAAAAAAAC0h0iUtbPQVe7TFuo9zMBU+eNCPIaF1ZOUFS8bmflu4wHAop9hAAAAAA==","ordinal":1,"depth":1,"balance_changes":[{"pubkey":"B2FIHTV0dLt8TXYk69O9s9g1XnPREEP8DaNTgAAAAAA=","prev_lamports":26858640,"new_lamports":26858640},{"pubkey":"BqfVFxh19ynHPZNAjyFhIAZ+2Ix24Iwof8GUYAAAAAA=","prev_lamports":1,"new_lamports":1},{"pubkey":"BqfVFxh19ynHPZNAjyFhIAZ+2Ix24Iwof8GUYAAAAAA=","prev_lamports":1,"new_lamports":1},{"pubkey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","prev_lamports":500000000000,"new_lamports":500000000000}]}]},{"id":"oLWk7rOnxt0YU446jkIfCVOh6FLeH4gnflT7555QpP7XcDZ3tNHy0SE6IwF5oKsQd9olPb/X9FIr+Zd1XuaMBw==","index":1,"header":{"num_required_signatures":1,"num_readonly_unsigned_accounts":1},"account_keys":["k/al+DWeemhGBaJ5J93UAxwcemXB6gR7OmgAhRSfNgQ=","6NxmV86hPpRROWZ6QDZ6Fqc96yk4lAoOk99KOuT7Ei0=","AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="],"recent_blockhash":"zNM8mdqpnNyqWHQrYHYfFB14t/6wZb8nS5jBGz4qcp0=","instructions":[{"program_id":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","account_keys":["k/al+DWeemhGBaJ5J93UAxwcemXB6gR7OmgAhRSfNgQ=","6NxmV86hPpRROWZ6QDZ6Fqc96yk4lAoOk99KOuT7Ei0="],"data":"AgAAAADodkgXAAAA","ordinal":1,"depth":1,"balance_changes":[{"pubkey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","prev_lamports":500000000000000000,"new_lamports":499999900000000000},{"pubkey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","new_lamports":100000000000}]}]}],"transaction_count":2}`
	b := &pbcodec.Block{}

	err := json.Unmarshal([]byte(blockCnt), b)
	require.NoError(t, err)

	blk := &bstream.Block{
		Id:             string(b.Id),
		Number:         b.Number,
		PreviousId:     string(b.PreviousId),
		LibNum:         1,
		PayloadKind:    codec.Protocol_SOL,
		PayloadVersion: 1,
	}

	cnt, err := proto.Marshal(b)
	require.NoError(t, err)

	blk, err = bstream.GetBlockPayloadSetter(blk, cnt)
	require.NoError(t, err)
	return blk
}
