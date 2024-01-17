package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	bin "github.com/streamingfast/binary"
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

	//todo : if err is a type skipped block error here, requestedSlot will be requestSlot + 1 while it's returning no skipped error
	var blockResult *rpc.GetBlockResult

	for {
		blockResult, err = f.rpcClient.GetBlockWithOpts(ctx, requestedSlot, GetBlockOpts)
		if err != nil {
			rpcErr := err.(*jsonrpc.RPCError)
			if rpcErr != nil && rpcErr.Code == -32009 {
				requestedSlot += 1
				continue
			}
			return nil, fmt.Errorf("fetching block %d: %w", requestedSlot, err)
		}
		break
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

	fixedPreviousBlockHash := fixPreviousBlockHash(result)

	transactions, err := toPbTransactions(result.Transactions)
	if err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}
	block := &pbsol.Block{
		PreviousBlockhash: result.PreviousBlockhash.String(),
		Blockhash:         fixedPreviousBlockHash,
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
		ParentId:  fixedPreviousBlockHash,
		Timestamp: timestamppb.New(result.BlockTime.Time()),
		LibNum:    libNum,
		ParentNum: result.ParentSlot,
		Payload:   payload,
	}

	return pbBlock, nil

}

func fixPreviousBlockHash(blockResult *rpc.GetBlockResult) (previousFixedBlockHash string) {
	switch blockResult.Blockhash.String() {
	case "Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v":
		//zlogger.Warn("applying horrible tweak to block Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v")
		if blockResult.PreviousBlockhash.String() == "11111111111111111111111111111111" {
			previousFixedBlockHash = "HQEr9qcbUVBt2okfu755FdJvJrPYTSpzzmmyeWTj5oau"
			return previousFixedBlockHash
		}
	case "6UFQveZ94DUKGbcLFoyayn1QwthVfD3ZqvrM2916pHCR":
		//zlogger.Warn("applying horrible tweak to block 63,072,071")
		if blockResult.PreviousBlockhash.String() == "11111111111111111111111111111111" {
			previousFixedBlockHash = "7cLQx2cZvyKbGoMuutXEZ3peg3D21D5qbX19T5V1XEiK"
			return previousFixedBlockHash
		}
	case "Fqbm7QvCTYnToXWcCw6nbkWhMmXx2Nv91LsXBrKraB43":
		//zlogger.Warn("applying horrible tweak to block 53,135,959")
		if previousFixedBlockHash == "11111111111111111111111111111111" {
			previousFixedBlockHash = "RfXUrekgajPSb1R4CGFJWNaHTnB6p53Tzert4gouj2u"
			return previousFixedBlockHash
		}
	case "ABp9G2NaPzM6kQbeyZYCYgdzL8JN9AxSSbCQG2X1K9UF":
		//zlogger.Warn("applying horrible tweak to block 46,223,993")
		if previousFixedBlockHash == "11111111111111111111111111111111" {
			previousFixedBlockHash = "9F2C7TGqUpFu6krd8vQbUv64BskrneBSgY7U2QfrGx96"
			return previousFixedBlockHash
		}
	case "ByUxmGuaT7iQS9qGS8on5xHRjiHXcGxvwPPaTGZXQyz7":
		//zlogger.Warn("applying horrible tweak to block 61,328,766")
		if previousFixedBlockHash == "11111111111111111111111111111111" {
			previousFixedBlockHash = "J6rRToKMK5DQDzVLqo7ibL3snwBYtqkYnRnQ7vXoUSEc"
			return previousFixedBlockHash
		}
	}

	return blockResult.PreviousBlockhash.String()
}

func toPbTransactions(transactions []rpc.TransactionWithMeta) (out []*pbsol.ConfirmedTransaction, err error) {
	for _, transaction := range transactions {
		meta, err := toPbTransactionMeta(transaction.Meta)
		if err != nil {
			return nil, fmt.Errorf(`decoding transaction meta: %w`, err)
		}
		if err != nil {
			return nil, fmt.Errorf(`decoding transaction: %w`, err)
		}
		out = append(out, &pbsol.ConfirmedTransaction{
			Transaction: toPbTransaction(transaction.MustGetTransaction()),
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
			StackHeight:    &compiledInstruction.StackHeight, //not return by the rpc endpoint getBlockCall
		})
	}
	return
}

func toPbTransactionError(e interface{}) (*pbsol.TransactionError, error) {
	if e == nil {
		return nil, nil
	}

	txErr := MustNewTransactionError(e)
	buf := bytes.NewBuffer(nil)
	encoder := bin.NewEncoder(buf)
	err := txErr.Encode(encoder)
	if err != nil {
		return nil, err
	}
	return &pbsol.TransactionError{
		Err: buf.Bytes(),
	}, nil
}

func toPbTransaction(transaction *solana.Transaction) *pbsol.Transaction {
	return &pbsol.Transaction{
		Signatures: toPbSignatures(transaction.Signatures),
		Message:    toPbMessage(transaction.Message),
	}

}

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
