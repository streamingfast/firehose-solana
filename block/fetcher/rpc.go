package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	sfsol "github.com/streamingfast/solana-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//todo: find right value for maxSupportedTransactionVersion

var MaxSupportedTransactionVersion = uint64(0)
var GetBlockOpts = &rpc.GetBlockOpts{
	Commitment:                     rpc.CommitmentConfirmed,
	MaxSupportedTransactionVersion: &MaxSupportedTransactionVersion,
}

type RPCFetcher struct {
	rpcClient                *rpc.Client
	latestConfirmedSlot      uint64
	latestFinalizedSlot      uint64
	latestBlockRetryInterval time.Duration
	fetchInterval            time.Duration
	lastFetchAt              time.Time
	logger                   *zap.Logger
}

func NewRPC(rpcClient *rpc.Client, fetchInterval time.Duration, latestBlockRetryInterval time.Duration, logger *zap.Logger) *RPCFetcher {
	return &RPCFetcher{
		rpcClient:                rpcClient,
		fetchInterval:            fetchInterval,
		latestBlockRetryInterval: latestBlockRetryInterval,
		logger:                   logger,
	}
}

func (f *RPCFetcher) Fetch(ctx context.Context, requestedSlot uint64) (out *pbbstream.Block, err error) {
	f.logger.Info("fetching block", zap.Uint64("block_num", requestedSlot))

	for f.latestConfirmedSlot < requestedSlot {
		f.latestConfirmedSlot, err = f.rpcClient.GetSlot(ctx, rpc.CommitmentConfirmed)
		if err != nil {
			return nil, fmt.Errorf("fetching latestConfirmedSlot block num: %w", err)
		}

		f.logger.Info("got latest confirmed slot block", zap.Uint64("latest_confirmed_slot", f.latestConfirmedSlot), zap.Uint64("block_num", requestedSlot))
		//
		if f.latestConfirmedSlot < requestedSlot {
			time.Sleep(f.latestBlockRetryInterval)
			continue
		}
		break
	}
	for f.latestFinalizedSlot < requestedSlot {
		f.latestFinalizedSlot, err = f.rpcClient.GetSlot(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return nil, fmt.Errorf("fetching latest finalized Slot block num: %w", err)
		}

		f.logger.Info("got latest finalized slot block", zap.Uint64("latest_finalized_slot", f.latestFinalizedSlot), zap.Uint64("block_num", requestedSlot))
		//
		if f.latestFinalizedSlot < requestedSlot {
			time.Sleep(f.latestBlockRetryInterval)
			continue
		}
		break
	}

	blockResult, err := f.rpcClient.GetBlockWithOpts(ctx, requestedSlot, GetBlockOpts)
	if err != nil {
		return nil, fmt.Errorf("fetching block %d: %w", requestedSlot, err)
	}
	block, err := blockFromBlockResult(requestedSlot, f.latestConfirmedSlot, f.latestFinalizedSlot, blockResult)
	if err != nil {
		return nil, fmt.Errorf("decoding block %d: %w", requestedSlot, err)
	}
	return block, nil
}

func blockFromBlockResult(requestedSlot uint64, confirmedSlot uint64, finalizedSlot uint64, result *rpc.GetBlockResult) (*pbbstream.Block, error) {

	libNum := finalizedSlot

	if finalizedSlot > requestedSlot {
		libNum = result.ParentSlot
	}

	//todo:  //horrible tweaks
	//	switch blk.Blockhash {
	//	case "Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v":
	//		zlogger.Warn("applying horrible tweak to block Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v")
	//		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
	//			blk.PreviousBlockhash = "HQEr9qcbUVBt2okfu755FdJvJrPYTSpzzmmyeWTj5oau"
	//		}
	//	case "6UFQveZ94DUKGbcLFoyayn1QwthVfD3ZqvrM2916pHCR":
	//		zlogger.Warn("applying horrible tweak to block 63,072,071")
	//		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
	//			blk.PreviousBlockhash = "7cLQx2cZvyKbGoMuutXEZ3peg3D21D5qbX19T5V1XEiK"
	//		}
	//	case "Fqbm7QvCTYnToXWcCw6nbkWhMmXx2Nv91LsXBrKraB43":
	//		zlogger.Warn("applying horrible tweak to block 53,135,959")
	//		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
	//			blk.PreviousBlockhash = "RfXUrekgajPSb1R4CGFJWNaHTnB6p53Tzert4gouj2u"
	//		}
	//	case "ABp9G2NaPzM6kQbeyZYCYgdzL8JN9AxSSbCQG2X1K9UF":
	//		zlogger.Warn("applying horrible tweak to block 46,223,993")
	//		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
	//			blk.PreviousBlockhash = "9F2C7TGqUpFu6krd8vQbUv64BskrneBSgY7U2QfrGx96"
	//		}
	//	case "ByUxmGuaT7iQS9qGS8on5xHRjiHXcGxvwPPaTGZXQyz7":
	//		zlogger.Warn("applying horrible tweak to block 61,328,766")
	//		if blk.PreviousBlockhash == "11111111111111111111111111111111" {
	//			blk.PreviousBlockhash = "J6rRToKMK5DQDzVLqo7ibL3snwBYtqkYnRnQ7vXoUSEc"
	//		}
	//	}

	//todo: horrible tweaks validate parent slot number

	transactions, err := toPbTransactions(result.Transactions)
	if err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}
	block := &pbsol.Block{
		PreviousBlockhash: result.PreviousBlockhash.String(),
		Blockhash:         result.Blockhash.String(),
		ParentSlot:        result.ParentSlot,
		Transactions:      transactions,
		Rewards:           toPBReward(result.Rewards),
		BlockTime:         pbsol.NewUnixTimestamp(result.BlockTime.Time()),
		BlockHeight: &pbsol.BlockHeight{
			BlockHeight: *result.BlockHeight,
		},
		Slot: requestedSlot,
	}

	payload, err := anypb.New(block)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal block: %w", err)
	}

	pbBlock := &pbbstream.Block{
		Number:    requestedSlot,
		Id:        result.Blockhash.String(),
		ParentId:  result.PreviousBlockhash.String(),
		Timestamp: timestamppb.New(result.BlockTime.Time()),
		LibNum:    libNum,
		ParentNum: result.ParentSlot,
		Payload:   payload,
	}

	return pbBlock, nil

}

func toPbTransactions(transactions []rpc.TransactionWithMeta) (out []*pbsol.ConfirmedTransaction, err error) {
	for _, transaction := range transactions {
		meta, err := toPbTransactionMeta(transaction.Meta)
		if err != nil {
			return nil, fmt.Errorf(`decoding transaction meta: %w`, err)
		}
		solanaTrx := &solana.Transaction{}
		transaction.Transaction.GetRawJSON()
		err = json.Unmarshal(transaction.Transaction.GetRawJSON(), solanaTrx)
		if err != nil {
			return nil, fmt.Errorf(`decoding transaction: %w`, err)
		}
		out = append(out, &pbsol.ConfirmedTransaction{
			Transaction: toPbTransaction(solanaTrx),
			Meta:        meta,
		})
	}
	return
}

func toPbTransactionMeta(meta *rpc.TransactionMeta) (*pbsol.TransactionStatusMeta, error) {
	returnData, err := toPbReturnData(meta.ReturnData)
	if err != nil {
		return nil, fmt.Errorf("decoding return data: %w", err)
	}
	trxErr, err := toPbTransactionError(meta.Err)
	return &pbsol.TransactionStatusMeta{
		Err:                     trxErr,
		Fee:                     meta.Fee,
		PreBalances:             meta.PreBalances,
		PostBalances:            meta.PostBalances,
		InnerInstructions:       toPbInnerInstructions(meta.InnerInstructions),
		InnerInstructionsNone:   false, //todo: should we remove?
		LogMessages:             meta.LogMessages,
		LogMessagesNone:         false, //todo: should we remove?
		PreTokenBalances:        toPbTokenBalances(meta.PreTokenBalances),
		PostTokenBalances:       toPbTokenBalances(meta.PostTokenBalances),
		Rewards:                 toPBReward(meta.Rewards),
		LoadedWritableAddresses: toPbWritableAddresses(meta.LoadedAddresses.Writable),
		LoadedReadonlyAddresses: toPbReadonlyAddresses(meta.LoadedAddresses.ReadOnly),
		ReturnData:              returnData,
		ReturnDataNone:          false, //todo: should we remove?
		ComputeUnitsConsumed:    meta.ComputeUnitsConsumed,
	}, nil
}

func toPbReturnData(data rpc.ReturnData) (*pbsol.ReturnData, error) {
	if len(data.Data) == 0 {
		return nil, nil
	}
	d, err := base64.StdEncoding.DecodeString(data.Data[0])
	if err != nil {
		return nil, fmt.Errorf("decoding return data: %w", err)
	}
	pId, err := sfsol.PublicKeyFromBase58(data.ProgramId)

	if err != nil {
		return nil, fmt.Errorf("decoding program id: %w", err)
	}
	return &pbsol.ReturnData{
		ProgramId: pId.ToSlice(),
		Data:      d,
	}, nil
}

func toPbReadonlyAddresses(readonlyAddresses solana.PublicKeySlice) [][]byte {
	var out [][]byte
	for _, readonlyAddresse := range readonlyAddresses {
		out = append(out, readonlyAddresse[:])
	}
	return out
}

func toPbWritableAddresses(writableAddresses solana.PublicKeySlice) [][]byte {
	var out [][]byte
	for _, writableAddresse := range writableAddresses {
		out = append(out, writableAddresse[:])
	}
	return out
}

func toPbTokenBalances(balances []rpc.TokenBalance) []*pbsol.TokenBalance {
	var out []*pbsol.TokenBalance
	for _, balance := range balances {
		out = append(out, &pbsol.TokenBalance{
			AccountIndex:  uint32(balance.AccountIndex),
			Mint:          balance.Mint.String(),
			UiTokenAmount: toPbUiTokenAmount(balance.UiTokenAmount),
			Owner:         balance.Owner.String(),
			ProgramId:     balance.ProgramId.String(),
		})
	}
	return out
}

func toPbUiTokenAmount(amount *rpc.UiTokenAmount) *pbsol.UiTokenAmount {
	if amount == nil {
		return nil
	}
	uiAmount := float64(0)
	if amount.UiAmount != nil {
		uiAmount = *amount.UiAmount
	}
	return &pbsol.UiTokenAmount{
		UiAmount:       uiAmount,
		Decimals:       uint32(amount.Decimals),
		Amount:         amount.Amount,
		UiAmountString: amount.UiAmountString,
	}
}

func toPbInnerInstructions(instructions []rpc.InnerInstruction) []*pbsol.InnerInstructions {
	var out []*pbsol.InnerInstructions
	for _, instruction := range instructions {
		out = append(out, &pbsol.InnerInstructions{
			Index:        uint32(instruction.Index),
			Instructions: compileInstructionsToPbInnerInstructionArray(instruction.Instructions),
		})
	}
	return out
}

func compileInstructionsToPbInnerInstructionArray(instructions []solana.CompiledInstruction) (out []*pbsol.InnerInstruction) {
	for _, compiledInstruction := range instructions {

		var accounts []byte
		for _, account := range compiledInstruction.Accounts {
			if account > math.MaxUint8 {
				panic("received instruction with account index greater then 256")
			}
			accounts = append(accounts, byte(account))
		}

		out = append(out, &pbsol.InnerInstruction{
			ProgramIdIndex: uint32(compiledInstruction.ProgramIDIndex),
			Accounts:       accounts,
			Data:           compiledInstruction.Data,
			StackHeight:    nil, //not return by the rpc endpoint getBlockCall //todo: check if it part of bigtable data
		})
	}
	return
}

type TransactionError struct {
	Type    string `json:"err"`
	Details string
}

func toPbTransactionError(err interface{}) (*pbsol.TransactionError, error) {
	if err == nil {
		return nil, nil
	}

	if mapErr, ok := err.(map[string]interface{}); ok {
		for key, value := range mapErr {
			detail, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("decoding transaction error: %w", err)
			}
			trxErr := &TransactionError{
				Type:    key,
				Details: string(detail),
			}
			fmt.Println(trxErr)
			return nil, nil
		}
	}

	//8 0 0 0 3 25 0 0 0 113 23 0 0

	//	"InstructionError": [
	//	  3,
	//	    {
	//	       "Custom": 6001
	//	    }
	//  ]

	//8 0 0 0 -> TransactionError.InstructionError
	//3 -> instruction index
	//25 0 0 0 -> InstructionError.Custom
	//113 23 0 0 -> u32 error code
	panic("not implemented") //todo : implement when test with a failed transaction
}

func toPbTransaction(transaction *solana.Transaction) *pbsol.Transaction {
	return &pbsol.Transaction{
		Signatures: toPbSignatures(transaction.Signatures),
		Message:    toPbMessage(transaction.Message),
	}

}

//todo: review message implementation with Charles

func toPbMessage(message solana.Message) *pbsol.Message {
	return &pbsol.Message{
		Header:              toPbMessageHeader(message.Header),
		AccountKeys:         toPbAccountKeys(message.AccountKeys),
		RecentBlockhash:     message.RecentBlockhash[:],
		Instructions:        toPbInstructions(message.Instructions),
		Versioned:           message.IsVersioned(),
		AddressTableLookups: toPbAddressTableLookups(message.AddressTableLookups),
	}
}

func toPbInstructions(instructions []solana.CompiledInstruction) []*pbsol.CompiledInstruction {
	var out []*pbsol.CompiledInstruction
	for _, instruction := range instructions {
		var accounts []byte
		for _, account := range instruction.Accounts {
			if account > math.MaxUint8 {
				panic("received instruction with account index greater then 256")
			}
			accounts = append(accounts, byte(account))
		}
		out = append(out, &pbsol.CompiledInstruction{
			ProgramIdIndex: uint32(instruction.ProgramIDIndex),
			Accounts:       accounts,
			Data:           instruction.Data,
		})
	}
	return out
}

func toPbAddressTableLookups(addressTableLookups solana.MessageAddressTableLookupSlice) (out []*pbsol.MessageAddressTableLookup) {
	for _, addressTableLookup := range addressTableLookups {
		out = append(out, &pbsol.MessageAddressTableLookup{
			AccountKey:      addressTableLookup.AccountKey[:],
			WritableIndexes: addressTableLookup.WritableIndexes,
			ReadonlyIndexes: addressTableLookup.ReadonlyIndexes,
		})
	}
	return
}

func toPbAccountKeys(accountKeys []solana.PublicKey) (out [][]byte) {
	for _, accountKey := range accountKeys {
		out = append(out, accountKey[:])
	}
	return
}

func toPbMessageHeader(header solana.MessageHeader) *pbsol.MessageHeader {
	return &pbsol.MessageHeader{
		NumRequiredSignatures:       uint32(header.NumRequiredSignatures),
		NumReadonlySignedAccounts:   uint32(header.NumReadonlySignedAccounts),
		NumReadonlyUnsignedAccounts: uint32(header.NumReadonlyUnsignedAccounts),
	}
}

func toPbSignatures(signatures []solana.Signature) (out [][]byte) {
	for _, signature := range signatures {
		out = append(out, signature[:])
	}
	return
}

func toPBReward(rewards []rpc.BlockReward) (out []*pbsol.Reward) {
	for _, reward := range rewards {
		out = append(out, &pbsol.Reward{
			Pubkey:      reward.Pubkey.String(),
			Lamports:    reward.Lamports,
			PostBalance: reward.PostBalance,
			RewardType:  toPBRewardType(reward.RewardType),
		})
	}
	return
}

func toPBRewardType(rewardType rpc.RewardType) pbsol.RewardType {
	switch rewardType {
	case rpc.RewardTypeFee:
		return pbsol.RewardType_Fee
	case rpc.RewardTypeRent:
		return pbsol.RewardType_Rent
	case rpc.RewardTypeVoting:
		return pbsol.RewardType_Voting
	case rpc.RewardTypeStaking:
		return pbsol.RewardType_Staking
	default:
		panic(fmt.Errorf("unsupported reward type %q", rewardType))
	}
}
