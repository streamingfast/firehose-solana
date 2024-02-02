package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	bin "github.com/streamingfast/binary"
	pbbstream "github.com/streamingfast/bstream/pb/sf/bstream/v1"
	"github.com/streamingfast/derr"
	pbsol "github.com/streamingfast/firehose-solana/pb/sf/solana/type/v1"
	sfsol "github.com/streamingfast/solana-go"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//todo: find right value for maxSupportedTransactionVersion

var MaxSupportedTransactionVersion = uint64(0)
var GetBlockOpts = &rpc.GetBlockOpts{
	Commitment:                     rpc.CommitmentConfirmed,
	MaxSupportedTransactionVersion: &MaxSupportedTransactionVersion,
}

type fetchBlock func(ctx context.Context, requestedSlot uint64) (slot uint64, out *rpc.GetBlockResult, err error)

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
	f := &RPCFetcher{
		rpcClient:                rpcClient,
		fetchInterval:            fetchInterval,
		latestBlockRetryInterval: latestBlockRetryInterval,
		logger:                   logger,
	}
	return f
}

func (f *RPCFetcher) IsBlockAvailable(requestedSlot uint64) bool {
	f.logger.Info("checking if block is available", zap.Uint64("request_block_num", requestedSlot), zap.Uint64("latest_confirmed_slot", f.latestConfirmedSlot))
	return requestedSlot <= f.latestConfirmedSlot
}

func (f *RPCFetcher) Fetch(ctx context.Context, requestedSlot uint64) (out *pbbstream.Block, skip bool, err error) {
	f.logger.Info("fetching block", zap.Uint64("block_num", requestedSlot))

	//THIS IS A FKG Ugly hack!
	if requestedSlot >= 13334464 && requestedSlot <= 13334475 {
		return nil, true, nil
	}

	sleepDuration := time.Duration(0)
	for f.latestConfirmedSlot < requestedSlot {
		time.Sleep(sleepDuration)
		f.latestConfirmedSlot, err = f.rpcClient.GetSlot(ctx, rpc.CommitmentConfirmed)
		if err != nil {
			return nil, false, fmt.Errorf("fetching latestConfirmedSlot block num: %w", err)
		}

		f.logger.Info("got latest confirmed slot block", zap.Uint64("latest_confirmed_slot", f.latestConfirmedSlot), zap.Uint64("requested_block_num", requestedSlot))
		//
		if f.latestConfirmedSlot >= requestedSlot {
			break
		}
		sleepDuration = f.latestBlockRetryInterval
	}

	if f.latestFinalizedSlot < requestedSlot {
		f.latestFinalizedSlot, err = f.rpcClient.GetSlot(ctx, rpc.CommitmentFinalized)
		if err != nil {
			return nil, false, fmt.Errorf("fetching latest finalized Slot block num: %w", err)
		}
		f.logger.Info("got latest finalized slot block", zap.Uint64("latest_finalized_slot", f.latestFinalizedSlot), zap.Uint64("requested_block_num", requestedSlot))
	}

	f.logger.Info("fetching block", zap.Uint64("block_num", requestedSlot), zap.Uint64("latest_finalized_slot", f.latestFinalizedSlot), zap.Uint64("latest_confirmed_slot", f.latestConfirmedSlot))

	blockResult, skip, err := f.fetch(ctx, requestedSlot)
	if err != nil {
		return nil, false, fmt.Errorf("fetching block %d: %w", requestedSlot, err)
	}

	if skip {
		return nil, true, nil
	}

	if blockResult == nil {
		panic("blockResult is nil and skip is false. This should not happen.")
	}

	block, err := blockFromBlockResult(requestedSlot, f.latestFinalizedSlot, blockResult, f.logger)
	if err != nil {
		return nil, false, fmt.Errorf("decoding block %d: %w", requestedSlot, err)
	}

	f.logger.Info("fetched block", zap.Uint64("block_num", requestedSlot), zap.String("block_hash", blockResult.Blockhash.String()))
	return block, false, nil
}

func (f *RPCFetcher) fetch(ctx context.Context, requestedSlot uint64) (*rpc.GetBlockResult, bool, error) {
	currentSlot := requestedSlot
	var out *rpc.GetBlockResult
	skipped := false
	//f.logger.Info("getting block", zap.Uint64("block_num", currentSlot))
	err := derr.Retry(math.MaxUint64, func(ctx context.Context) error {
		var innerErr error
		out, innerErr = f.rpcClient.GetBlockWithOpts(ctx, requestedSlot, GetBlockOpts)

		if innerErr != nil {
			var rpcErr *jsonrpc.RPCError
			if errors.As(innerErr, &rpcErr) {
				if rpcErr.Code == -32004 {
					f.logger.Warn("block not available. Retrying same block", zap.Uint64("block_num", currentSlot))
					return innerErr
				}
				if rpcErr.Code == -32009 || rpcErr.Code == -32007 {
					f.logger.Info("block was skipped", zap.Uint64("block_num", currentSlot))
					currentSlot += 1
					skipped = true
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("after retrying fetch block %d: %w", requestedSlot, err)
	}
	return out, skipped, err
}

func blockFromBlockResult(slot uint64, finalizedSlot uint64, result *rpc.GetBlockResult, logger *zap.Logger) (*pbbstream.Block, error) {
	libNum := finalizedSlot

	if finalizedSlot > slot {
		libNum = result.ParentSlot
	}

	fixedPreviousBlockHash := fixPreviousBlockHash(result, logger)

	transactions, err := toPbTransactions(result.Transactions)
	if err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}

	var blockTime *pbsol.UnixTimestamp
	if result.BlockTime != nil {
		blockTime = pbsol.NewUnixTimestamp(result.BlockTime.Time())
	}

	var blockHeight *pbsol.BlockHeight
	if result.BlockHeight != nil {
		blockHeight = &pbsol.BlockHeight{
			BlockHeight: *result.BlockHeight,
		}
	}
	block := &pbsol.Block{
		PreviousBlockhash: fixedPreviousBlockHash,
		Blockhash:         result.Blockhash.String(),
		ParentSlot:        result.ParentSlot,
		Transactions:      transactions,
		Rewards:           toPBReward(result.Rewards),
		BlockTime:         blockTime,
		BlockHeight:       blockHeight,
		Slot:              slot,
	}

	payload, err := anypb.New(block)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal block: %w", err)
	}

	var timeStamp *timestamppb.Timestamp
	if result.BlockTime != nil {
		timeStamp = timestamppb.New(result.BlockTime.Time())
	}
	pbBlock := &pbbstream.Block{
		Number:    slot,
		Id:        result.Blockhash.String(),
		ParentId:  fixedPreviousBlockHash,
		Timestamp: timeStamp,
		LibNum:    libNum,
		ParentNum: result.ParentSlot,
		Payload:   payload,
	}

	return pbBlock, nil

}

var blockToPatch = map[string]string{
	"2TLDT6Z3WJ5h5958BjdzMwmNGnVo3e4qcHyGBVgBPDm9": "FCgBdK9Fufcsdc9RGu5SwMwbCFiw4SxNnJzCpZTdNpDq",
	"AACrVjKzuvTLxmjC7Ktn5St1GkFHd9smvNHDf2dCPF5o": "4hkGhNnuuCFqD3w35KXJJvDyfkbSQHZgrUn7SKc7xy76",
	"Dp1h2oCTMisFbT2EVmpd8thzEZA59sHy5cVEWpnAW9nK": "AkKwmqftQux3tBPZMkHUFEG1ShRj86my3PtLcsXiPEhC",
	"3fDddQ4CMS1v2AQRhzTKoUV7bhUj7FnrrhgCR27sxbwL": "2hcGqX4YBiD5jcj3AvWCk2DNjN7gL8trnGVdBJ1bmHLw",
	"HSrRb3iwKJafacyaDGY3U1UTEcW8fnR6RKJeXbcnjjJL": "bFjEKnrEytyyfgfvPn46tXYMjqyERfq5yQW8wETMj5b",
	"4mjTfuxGqczLL2hHDsJc86Ni3GuiVHyRZ76xWzYEp1rd": "EWL4JSYcGNdgqEKc7LceGbSkLgLRvJPLxruXHaE5LBXz",
	"FexbnjDrAshJSGUETr8DoB8hUrVbmEe9frxZuZE3YLj6": "5HP9qtrjKkYQgqijkkRJJUYY3EunSYWi7vH8RRoqVE5B",
	"4kM9y9ucKjfoTmjt41Qn83xpZbBUgLDAbVfySKbyUcRD": "CAHggyj6n8ytmb5peziHDTaPyQGAjDuNRdBZhzNAPYUC",
	"GHpc3nirTs9fj95andWPw4QX4jwi3Uu4NLJEKbWJ5YmX": "A8p44eo3n9nEyaCv5Mfa77DfJky2m5i6XRdZwBAqADM8",
	"DhomkvG22nCYqwhoghtpsaoUsNwE3Sd5rP3uWDKk3dcd": "8CZuKdcphp4vs5emnNyPeguLLCtdCwyh4hmEy6XtZ9uo",
	"Hde6FztxXayXcySpHdtNGK9HJGoNGzmDPLreqv1ocQJr": "AC27cduJoqJu86ETpajJmrwktZko5epUr1ezfLhx1Vzt",
	"8VJJtvfTo5ixbEA4YyvuLiUQdi1x3fgeY78hhLohD9Dq": "77Wa7nyGcDJnY8wWhVWBAjzpxr3ovqH265ZpVjX6N4LA",
	"GnjK9dG91VpLkWPNFZwqv6Dyp6FSVtgTTgcZp1Fc8gbU": "AGuB4sQ2xBQ7qLxHAE1jEDbEQun76LRVPpbsU7TA2Ceu",
	"2p5J7RpEAcv7S1rFdubBd2Jsxk5A9gsPZPYEeZ3AFmPb": "37duR4zVdkmBDQf7nRbWwabzWdCTNY4YxrT7BXQCRnSz",
	"GyRKbxESPzwQgzY9HGKCahzyCoYE5LPcAeybL5JA43Cv": "6A7Thgk1sWmX5RhygsqxsVKzXyRJTtUVrYCz5fjZEb6x",
	"BXnB4SLEKHJiUeM6CShcQhG5kC5mXY6xmnzPYnU64sWH": "FyUTMMDb8u7jcQeuouoZ1JS72dbuqxeBvf8PFEZeXwBn",
	"BTFfa2oTTsCecqmj8JVn5gdpLazAh58mqJFHCkiLMKHF": "CQzukbEBmT9Kf2VCeW8oV7ASvRitZwcB4tVdwdm3VmGN",
	"7wwDCCNA9EfLiZQzzBZTZKPsJisWxVHNUbKkSWjmnaHg": "6UDRkQfuAHwtMyZVBi8ACoqk2HeumjhVBtGiLGfkwTMD",
	"C8qCiSUrvjAcGizHDDUHwCfYYZnqDb6tnzsg7XYBKCZ3": "GnN1RorTCy3DCzCJxZjPjSQZaX2JvFbF1g7dMCMMmCiB",
	"FBnxhciRnEKEtpwX7VRQLv55xCEmTHtRy7fEvjid2W8S": "A85UR6HfVfdNezrSBc7fQiNtbfZrYWXtEbjMuW3Mnqt7",
	"9WgaJZbYTD4WpTQnQtia87xbb9y3iqsdWJNVzcLG7rX9": "9YWWR5h3tPHKgm6ZpGAHisdeYGyjtNbwXiyLUvKzqP3H",
	"3BzDxbNwCgtCDdg74KC6LFnDJVAyJeqtSwKuFL4cufx9": "9d8LkjdgGxVJfB5PUnhFUr2r7w8hMmKeoEQg7Kg2jFiJ",
	"CKL5Pd6f85jtdrLibnbVtnY8VatS9rjBsX8ztD6wAfdj": "7uBfGie2UTW7sUfnZvvYbXi1y5BTUdUWTYhX2WxhYTSf",
	"7x3cd4zzTMn9ixa4unPN2UGi4ctU8qEpqVAwyC9dwvZe": "4VoUxo2RrJ1reriLMUzuA3VaKuGKqMszXY7KDt6i4cpP",
	"Goi3t9JjgDkyULZbM2TzE5QqHP1fPeMcHNaXNFBCBv1v": "HQEr9qcbUVBt2okfu755FdJvJrPYTSpzzmmyeWTj5oau",
	"6UFQveZ94DUKGbcLFoyayn1QwthVfD3ZqvrM2916pHCR": "7cLQx2cZvyKbGoMuutXEZ3peg3D21D5qbX19T5V1XEiK",
	"Fqbm7QvCTYnToXWcCw6nbkWhMmXx2Nv91LsXBrKraB43": "RfXUrekgajPSb1R4CGFJWNaHTnB6p53Tzert4gouj2u",
	"ABp9G2NaPzM6kQbeyZYCYgdzL8JN9AxSSbCQG2X1K9UF": "9F2C7TGqUpFu6krd8vQbUv64BskrneBSgY7U2QfrGx96",
	"ByUxmGuaT7iQS9qGS8on5xHRjiHXcGxvwPPaTGZXQyz7": "J6rRToKMK5DQDzVLqo7ibL3snwBYtqkYnRnQ7vXoUSEc",
}

func fixPreviousBlockHash(blockResult *rpc.GetBlockResult, logger *zap.Logger) (previousFixedBlockHash string) {
	if prev, ok := blockToPatch[blockResult.Blockhash.String()]; ok {
		logger.Info("patching previous block hash", zap.String("block_hash", blockResult.Blockhash.String()), zap.String("previous_block_hash", prev))
		return prev

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
	if meta == nil {
		return &pbsol.TransactionStatusMeta{}, nil
	}
	returnData, err := toPbReturnData(meta.ReturnData)
	if err != nil {
		return nil, fmt.Errorf("decoding return data: %w", err)
	}

	innerInstructions := toPbInnerInstructions(meta.InnerInstructions)

	trxErr, err := toPbTransactionError(meta.Err)
	return &pbsol.TransactionStatusMeta{
		Err:                     trxErr,
		Fee:                     meta.Fee,
		PreBalances:             meta.PreBalances,
		PostBalances:            meta.PostBalances,
		InnerInstructions:       innerInstructions,
		LogMessages:             meta.LogMessages,
		PreTokenBalances:        toPbTokenBalances(meta.PreTokenBalances),
		PostTokenBalances:       toPbTokenBalances(meta.PostTokenBalances),
		Rewards:                 toPBReward(meta.Rewards),
		LoadedWritableAddresses: toPbWritableAddresses(meta.LoadedAddresses.Writable),
		LoadedReadonlyAddresses: toPbReadonlyAddresses(meta.LoadedAddresses.ReadOnly),
		ReturnData:              returnData,
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
		o := make([]byte, len(readonlyAddresse))
		copy(o, readonlyAddresse[:])
		out = append(out, o)
	}
	return out
}

func toPbWritableAddresses(writableAddresses solana.PublicKeySlice) [][]byte {
	var out [][]byte
	for _, writableAddresse := range writableAddresses {
		o := make([]byte, len(writableAddresse))
		copy(o, writableAddresse[:])
		out = append(out, o)
	}
	return out
}

func toPbTokenBalances(balances []rpc.TokenBalance) []*pbsol.TokenBalance {
	var out []*pbsol.TokenBalance

	for _, balance := range balances {
		var owner string
		if balance.Owner != nil {
			owner = balance.Owner.String()
		}

		var programId string

		if balance.ProgramId.String() != "11111111111111111111111111111111" {
			programId = balance.ProgramId.String()
		}

		out = append(out, &pbsol.TokenBalance{
			AccountIndex:  uint32(balance.AccountIndex),
			Mint:          balance.Mint.String(),
			UiTokenAmount: toPbUiTokenAmount(balance.UiTokenAmount),
			Owner:         owner,
			ProgramId:     programId,
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
			StackHeight:    toStackHeight(compiledInstruction.StackHeight),
		})
	}
	return
}

func toStackHeight(stackHeight uint32) *uint32 {
	if stackHeight == 0 {
		return nil
	}
	s := stackHeight
	return &s
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
		o := make([]byte, len(addressTableLookup.AccountKey))
		copy(o, addressTableLookup.AccountKey[:])

		out = append(out, &pbsol.MessageAddressTableLookup{
			AccountKey:      o,
			WritableIndexes: addressTableLookup.WritableIndexes,
			ReadonlyIndexes: addressTableLookup.ReadonlyIndexes,
		})
	}
	return
}

func toPbAccountKeys(accountKeys []solana.PublicKey) (out [][]byte) {
	for _, accountKey := range accountKeys {
		a := make([]byte, len(accountKey))
		copy(a, accountKey[:])
		out = append(out, a)
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
		s := make([]byte, len(signature))
		copy(s, signature[:])

		out = append(out, s)
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

	slices.SortFunc(out, func(a, b *pbsol.Reward) bool {
		return a.Lamports > b.Lamports
	})

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
