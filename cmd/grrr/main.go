package main

import (
	"fmt"

	"github.com/streamingfast/dbin"

	pbbstream "github.com/streamingfast/pbgo/sf/bstream/v1"
	"google.golang.org/protobuf/proto"
)

func main() {
	//data, err := os.ReadFile("/Users/cbillett/devel/sf/0154667200.dbin")
	//if err != nil {
	//	panic(err)
	//}

	fr, err := dbin.NewFileReader("/Users/cbillett/devel/sf/0154667200.dbin")
	if err != nil {
		panic(err)
	}

	contentType, version, err := fr.ReadHeader()
	if err != nil {
		panic(err)
	}
	fmt.Println("type:", contentType, "version:", version)

	for {
		data, err := fr.ReadMessage()
		if err != nil {
			panic(err)
		}
		b := &pbbstream.Block{}
		err = proto.Unmarshal(data, b)
		if err != nil {
			panic(err)
		}

		fmt.Println(b.Number)
	}
}
