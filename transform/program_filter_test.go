package transform

import (
	"fmt"
	pbtransform "github.com/streamingfast/sf-solana/pb/sf/solana/transforms/v1"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestProgramFilter_Doc(t *testing.T) {
	foo := &pbtransform.ProgramFilter{}
	fmt.Println(proto.MessageName(foo))
}
