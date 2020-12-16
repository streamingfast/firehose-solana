package dgraphql

import (
	"encoding/json"
	"fmt"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/dfuse-io/dgraphql/static"
)

//go:generate rice embed-go

func GraphqlExamples() []*static.GraphqlExample {
	box := rice.MustFindBox("examples")

	return []*static.GraphqlExample{
		{
			Label:    "Get All Registered Tokens",
			Document: graphqlDocument(box, "get_all_registered_tokens.graphql"),
		},
		{
			Label:    "Get All Tokens",
			Document: graphqlDocument(box, "get_all_tokens.graphql"),
		},
		{
			Label:    "Get Registered Token",
			Document: graphqlDocument(box, "get_registered_token.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
				"mainnet": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
			},
		},
		{
			Label:    "Get Token",
			Document: graphqlDocument(box, "get_token.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
				"mainnet": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
			},
		},
		{
			Label:    "Stream Serum Instructions",
			Document: graphqlDocument(box, "stream_serum_instructions.graphql"),
			Variables: static.GraphqlVariablesByNetwork{
				"generic": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
				"mainnet": r(`{"account": "YOUR-ACCOUNT-HERE"}`),
			},
		},
	}
}

func graphqlDocument(box *rice.Box, name string) static.GraphqlDocument {
	asset, err := box.String(name)
	if err != nil {
		panic(fmt.Errorf("unable to get content for graphql examples file %q: %w", name, err))
	}

	return static.GraphqlDocument(asset)
}

func oneWeekAgo() string {
	return time.Now().Add(-7 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
}

func dateOffsetByBlock(blockCount int) string {
	return time.Now().Add(time.Duration(blockCount) * 500 * time.Millisecond).UTC().Format("2006-01-02T15:04:05Z")
}

func r(rawJSON string) json.RawMessage {
	return json.RawMessage(rawJSON)
}
