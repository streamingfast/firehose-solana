package transform

import (
	"testing"

	pbtransforms "github.com/streamingfast/sf-solana/types/pb/sf/solana/transforms/v1"
	pbsolv2 "github.com/streamingfast/sf-solana/types/pb/sf/solana/type/v2"

	"github.com/streamingfast/bstream/transform"
	"github.com/streamingfast/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func programFilterTransform(programIds []solana.PublicKey, t *testing.T) *anypb.Any {
	transform := &pbtransforms.ProgramFilter{ProgramIds: []string{}}
	for _, pid := range programIds {
		transform.ProgramIds = append(transform.ProgramIds, pid.String())
	}
	a, err := anypb.New(transform)
	require.NoError(t, err)
	return a
}

func TestProgramFilter_Transform(t *testing.T) {
	t.Skipf("fix transformer test")
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
	//transformReg.Register(ProgramFilterFactory)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			transforms := []*anypb.Any{programFilterTransform(test.programIds, t)}

			preprocFunc, _, _, err := transformReg.BuildFromTransforms(transforms)
			require.NoError(t, err)

			blk := testBlock(t)

			output, err := preprocFunc(blk)
			if test.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				pbcodecBlock := output.(*pbsolv2.Block)
				assert.Equal(t, test.expectTrxLenght, len(pbcodecBlock.Transactions))
			}
		})
	}

}
