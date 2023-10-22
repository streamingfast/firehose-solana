package main

import (
	"cloud.google.com/go/bigtable"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/mr-tron/base58/base58"
	"github.com/spf13/cobra"
	"github.com/streamingfast/cli/sflags"
	firecore "github.com/streamingfast/firehose-core"
	"github.com/streamingfast/firehose-solana/bt"
	pbsolv1 "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	"github.com/streamingfast/logging"
	"github.com/streamingfast/solana-go/programs/addresstablelookup"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"strconv"
)

func newToolsPrintDataCmd(logger *zap.Logger, tracer logging.Tracer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "print-data <account_address> <blockNum>",
		Short:   "print all instruction data for an account address at given block number",
		Args:    cobra.ExactArgs(2),
		RunE:    printDataE(logger, tracer),
		Example: "firesol tools bt print-data 4zAnosJWXmzwYAjgBxYSm8Rjtf5CJv75kyE24d5nQvQb 163360384",
	}

	cmd.Flags().Bool("firehose-enabled", false, "When enable the blocks read will output Firehose formatted logs 'FIRE <block_num> <block_payload_in_hex>'")
	cmd.Flags().Bool("compact", false, "When printing in JSON it will print compact instead of pretty-printed output")
	cmd.Flags().Bool("linkable", false, "Ensure that no block is skipped they are linkable")
	cmd.Flags().String("trx-hash", "", "Search only the data for a specific transaction")
	return cmd
}

func printDataE(logger *zap.Logger, tracer logging.Tracer) firecore.CommandExecutor {
	return func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		accountAddress := args[0]
		blockNumStr := args[1]
		trxHashFlag := sflags.MustGetString(cmd, "trx-hash")

		firehoseEnabled := sflags.MustGetBool(cmd, "firehose-enabled")
		compact := sflags.MustGetBool(cmd, "compact")
		linkable := sflags.MustGetBool(cmd, "linkable")
		btProject := sflags.MustGetString(cmd, "bt-project")
		btInstance := sflags.MustGetString(cmd, "bt-instance")

		logger.Info("retrieving from bigtable",
			zap.Bool("firehose_enabled", firehoseEnabled),
			zap.Bool("compact", compact),
			zap.Bool("linkable", linkable),
			zap.String("block_num", blockNumStr),
			zap.String("bt_project", btProject),
			zap.String("bt_instance", btInstance),
			zap.String("account_address", accountAddress),
			zap.String("trx_hash", trxHashFlag),
		)
		client, err := bigtable.NewClient(ctx, btProject, btInstance)
		if err != nil {
			return err
		}
		startBlockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to parse start block number %s: %w", blockNumStr, err)
		}

		btClient := bt.New(client, 10, logger, tracer)

		return btClient.ReadBlocks(ctx, startBlockNum, startBlockNum+1, linkable, func(block *pbsolv1.Block) error {
			if firehoseEnabled {
				cnt, err := proto.Marshal(block)
				if err != nil {
					return fmt.Errorf("failed to proto  marshal pb sol block: %w", err)
				}
				b64Cnt := base64.StdEncoding.EncodeToString(cnt)
				lineCnt := fmt.Sprintf("FIRE BLOCK %d %s", block.Slot, b64Cnt)
				if _, err := fmt.Println(lineCnt); err != nil {
					return fmt.Errorf("failed to write log line (char lenght %d): %w", len(lineCnt), err)
				}
				return nil
			}

			blockNum := block.BlockHeight.BlockHeight

			for _, trx := range block.Transactions {
				if trx.Meta.Err != nil {
					continue
				}
				accountKeys := trx.Transaction.Message.AccountKeys
				var data []*Data
				trxHash := base58.Encode(trx.Transaction.Signatures[0])
				logger.Debug("processing trx", zap.String("hash", trxHash))

				if trxHashFlag != "" && trxHashFlag != trxHash {
					continue
				}

				for instIndex, inst := range trx.Transaction.Message.Instructions {
					d, err := processInstruction(blockNum, trxHash, accountKeys, accountAddress, inst)
					if err != nil {
						return fmt.Errorf("processing compiled instruction: %w", err)
					}
					if d != nil {
						data = append(data, d)
					}

					if instIndex+1 > len(trx.Meta.InnerInstructions) {
						continue
					}

					inner := trx.Meta.InnerInstructions[instIndex]
					for _, inst := range inner.Instructions {
						d, err = processInstruction(blockNum, trxHash, accountKeys, accountAddress, inst)
						if err != nil {
							return fmt.Errorf("processing inner instruction: %w", err)
						}
						if d != nil {
							data = append(data, d)
						}
					}
				}

				if data == nil {
					continue
				}
				cnt, err := json.MarshalIndent(data, "", "    ")
				if err != nil {
					return fmt.Errorf("unable to json marshall block: %w", err)
				}
				fmt.Println(string(cnt))
			}

			return nil
		})
	}
}

func processInstruction(
	blockNumber uint64,
	trxHash string,
	accountKeys [][]byte,
	accountAddress string,
	instructionable pbsolv1.Instructionable,
) (*Data, error) {
	var data *Data

	inst := instructionable.ToInstruction()

	if len(inst.Accounts) == 0 {
		return nil, nil
	}

	if base58.Encode(accountKeys[inst.Accounts[0]]) == accountAddress {
		decodedInstruction, err := addresstablelookup.DecodeInstruction(inst.Data)
		if err != nil {
			return nil, fmt.Errorf("decoding address table lookup instruction: %w", err)
		}

		switch val := decodedInstruction.Impl.(type) {
		case *addresstablelookup.ExtendLookupTable:
			accountsFromDataInString := make([]string, len(val.Addresses))
			for i := range val.Addresses {
				accountsFromDataInString[i] = base58.Encode(val.Addresses[i][:])
			}
			data = &Data{
				BlockNum: blockNumber,
				TrxHash:  trxHash,
				Val:      accountsFromDataInString,
			}
		default:
			return nil, nil // only interested in extend lookup table instruction
		}
	}

	return data, nil
}

type Data struct {
	BlockNum uint64   `json:"block_num"`
	TrxHash  string   `json:"trx_hash"`
	Val      []string `json:"val"`
}
