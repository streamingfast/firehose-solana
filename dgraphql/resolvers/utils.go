package resolvers

import (
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/graph-gophers/graphql-go"
)

func toTime(timestamp *timestamp.Timestamp) graphql.Time {
	t, err := ptypes.Timestamp(timestamp)
	if err != nil {
		panic(fmt.Errorf("toTime: %s", err))
	}

	return graphql.Time{Time: t}
}
