package transform

import (
	"github.com/streamingfast/bstream/transform"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	pbtransforms "github.com/streamingfast/sf-solana/pb/sf/solana/transforms/v1"
	"github.com/streamingfast/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"testing"
)

func programFilterTransform(programIds []solana.PublicKey, t *testing.T) *anypb.Any {
	transform := &pbtransforms.ProgramFilter{ProgramIds: [][]byte{}}
	for _, pid := range programIds {
		transform.ProgramIds = append(transform.GetProgramIds(), pid.ToSlice())
	}
	a, err := anypb.New(transform)
	require.NoError(t, err)
	return a
}

func TestProgramFilter_Transform(t *testing.T) {
	tests := []struct {
		name            string
		programIds      []solana.PublicKey
		expectError     bool
		expectTrxLenght int
	}{
		{
			name:            "single matching program filter",
			programIds:      []solana.PublicKey{solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111")},
			expectTrxLenght: 1,
		},
		{
			name:            "single none program filter",
			programIds:      []solana.PublicKey{solana.MustPublicKeyFromBase58("11111111111111111111111111111111")},
			expectTrxLenght: 1,
		},
		{
			name: "one matching one that does not",
			programIds: []solana.PublicKey{
				solana.MustPublicKeyFromBase58("Vote111111111111111111111111111111111111111"),
				solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
			},
			expectTrxLenght: 2,
		},
	}

	transformReg := transform.NewRegistry()
	transformReg.Register(ProgramFilterFactory)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transforms := []*anypb.Any{programFilterTransform(test.programIds, t)}

			preprocFunc, err := transformReg.BuildFromTransforms(transforms)
			require.NoError(t, err)

			blk := testBlock(t)

			output, err := preprocFunc(blk)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				pbcodecBlock := output.(*pbcodec.Block)
				assert.Equal(t, test.expectTrxLenght, len(pbcodecBlock.Transactions))
			}
		})
	}

}
