package transform

import (
	"bytes"
	"fmt"
	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/transform"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	pbtransforms "github.com/streamingfast/sf-solana/pb/sf/solana/transforms/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var ProgramFilterMessageName = proto.MessageName(&pbtransforms.ProgramFilter{})

var ProgramFilterFactory = &transform.Factory{
	Obj: &pbtransforms.ProgramFilter{},
	NewFunc: func(message *anypb.Any) (transform.Transform, error) {
		mname := message.MessageName()
		if mname != ProgramFilterMessageName {
			return nil, fmt.Errorf("expected type url %q, recevied %q ", ProgramFilterMessageName, message.TypeUrl)
		}

		filter := &pbtransforms.ProgramFilter{}
		err := proto.Unmarshal(message.Value, filter)
		if err != nil {
			return nil, fmt.Errorf("unexpected unmarshall error: %w", err)
		}

		return &ProgramFilter{
			filteredProgramId: filter.ProgramIds,
		}, nil
	},
}

type ProgramFilter struct {
	filteredProgramId [][]byte
}

func (p *ProgramFilter) matches(programId []byte) bool {
	for _, pid := range p.filteredProgramId {
		if bytes.Equal(pid, programId) {
			return true
		}
	}
	return false
}
func (p *ProgramFilter) Transform(readOnlyBlk *bstream.Block, in transform.Input) (transform.Output, error) {
	solBlock := readOnlyBlk.ToProtocol().(*pbcodec.Block)
	filteredTransactions := []*pbcodec.Transaction{}
	for _, transaction := range solBlock.Transactions {
		match := false
		for _, instruction := range transaction.Instructions {
			if p.matches(instruction.ProgramId) {
				match = true
			}
		}
		if match {
			filteredTransactions = append(filteredTransactions, transaction)
		}
	}
	solBlock.Transactions = filteredTransactions
	solBlock.TransactionCount = uint32(len(filteredTransactions))
	return solBlock, nil
}
