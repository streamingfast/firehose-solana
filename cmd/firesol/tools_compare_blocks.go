package main

import (
	"cmp"
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/cli"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-core/cmd/tools/compare"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
)

// The RPC poller and the geyser plugin don't write a block's rewards the same way: the poller sorts
// them while geyser keeps the validator's order (RPC endpoints don't agree on one), and older poller
// versions write a zero commission as "0" where geyser leaves it empty. Comparing blocks from the two
// extractors flags every block with rewards unless those are ignored.
var ignoreRewardsOrder = boolEnv("FIRESOL_COMPARE_IGNORE_REWARDS_ORDER")
var ignoreZeroCommissionFormat = boolEnv("FIRESOL_COMPARE_IGNORE_ZERO_COMMISSION_FORMAT")

// boolEnv returns true if the given environment variable is set to "true"
func boolEnv(name string) bool {
	return os.Getenv(name) == "true"
}

func NewToolsCompareBlocksCmd(logger *zap.Logger) *cobra.Command {
	chain := &firecore.Chain[*pbsol.Block]{
		ShortName: "sol",
		Tools: &firecore.ToolsConfig[*pbsol.Block]{
			SanitizeBlockForCompare: sanitizeSolanaBlockForCompare,
		},
	}

	cmd := compare.NewToolsCompareBlocksCmd(chain, logger)
	cmd.Long += "\n\n" + cli.Dedent(`
		Set FIRESOL_COMPARE_IGNORE_REWARDS_ORDER=true to compare a block's rewards ignoring their order,
		and FIRESOL_COMPARE_IGNORE_ZERO_COMMISSION_FORMAT=true to treat a zero commission written as "0"
		the same as an empty one. Both are needed to compare RPC poller blocks with geyser plugin ones.
	`)

	// firecore registers those on its 'tools' command, which firesol doesn't configure
	flags := cmd.PersistentFlags()
	flags.String("bytes-encoding", "hex", "Encoding for bytes fields when printing diffs, either 'hex', 'base58' or 'base64'")
	flags.Uint64("merged-blocks-bundle-size", 100, "Number of blocks per merged-blocks file in the stores being read")
	flags.StringSlice("proto-paths", []string{""}, "Paths to proto files to use for dynamic decoding of blocks")

	return cmd
}

func sanitizeSolanaBlockForCompare(block *pbbstream.Block) *pbbstream.Block {
	if !ignoreRewardsOrder && !ignoreZeroCommissionFormat {
		return block
	}

	solBlock := &pbsol.Block{}
	if err := block.Payload.UnmarshalTo(solBlock); err != nil {
		panic(fmt.Errorf("unexpected block message %s, unable to unmarshal to pbsol.Block: %w", block.Payload.TypeUrl, err))
	}

	if ignoreZeroCommissionFormat {
		for _, r := range solBlock.Rewards {
			if r.Commission == "0" {
				r.Commission = ""
			}
		}
	}
	if ignoreRewardsOrder {
		slices.SortFunc(solBlock.Rewards, func(a, b *pbsol.Reward) int {
			if c := cmp.Compare(a.Pubkey, b.Pubkey); c != 0 {
				return c
			}
			if c := cmp.Compare(a.RewardType, b.RewardType); c != 0 {
				return c
			}
			return cmp.Compare(a.Lamports, b.Lamports)
		})
	}

	payload, err := anypb.New(solBlock)
	if err != nil {
		panic(fmt.Errorf("unable to marshal sanitized block %d: %w", block.Number, err))
	}
	block.Payload = payload

	return block
}
