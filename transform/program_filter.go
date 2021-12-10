package transform

import (
	"bytes"
	"fmt"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/bstream/transform"
	pbcodec "github.com/streamingfast/sf-solana/pb/sf/solana/codec/v1"
	pbtransforms "github.com/streamingfast/sf-solana/pb/sf/solana/transforms/v1"
	"google.golang.org/protobuf/proto"
)

var NewProgramFilterFactory = func(message proto.Message) (transform.Transform, error) {
	obj, ok := message.(*pbtransforms.ProgramFilter)
	if !ok {
		return nil, fmt.Errorf("invalid proto message type expected 'ProgramFilter'")
	}

	return &ProgramFilter{
		filteredProgramId: obj.ProgramIds,
	}, nil
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
func (p *ProgramFilter) Transform(blk *bstream.Block, in transform.Input) (out transform.Output) {
	slot := blk.ToNative().(*pbcodec.Block)
	filteredTransactions := []*pbcodec.Transaction{}
	for _, transaction := range slot.Transactions {
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
	slot.Transactions = filteredTransactions
	slot.TransactionCount = uint32(len(filteredTransactions))
	return slot
}

func (p ProgramFilter) Doc() string {
	return "program filter documenation"
}
