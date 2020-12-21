// Code generated by rice embed-go; DO NOT EDIT.
package dgraphql

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file2 := &embedded.EmbeddedFile{
		Filename:    "get_all_registered_tokens.graphql",
		FileModTime: time.Unix(1608158040, 0),

		Content: string("{\n  registeredTokens {\n    address\n    mintAuthority\n    freezeAuthority\n    supply\n    decimals\n    symbol\n    name\n    logo\n    website\n  }\n}\n"),
	}
	file3 := &embedded.EmbeddedFile{
		Filename:    "get_all_tokens.graphql",
		FileModTime: time.Unix(1608158030, 0),

		Content: string("{\n  tokens {\n    address\n    mintAuthority\n    freezeAuthority\n    supply\n    decimals\n  }\n}\n"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "get_registered_token.graphql",
		FileModTime: time.Unix(1608158059, 0),

		Content: string("query ($account: String!) {\n  registeredToken(address: $account) {\n    address\n    mintAuthority\n    freezeAuthority\n    supply\n    decimals\n    symbol\n    name\n    logo\n    website\n  }\n}\n"),
	}
	file5 := &embedded.EmbeddedFile{
		Filename:    "get_serum_fill.graphql",
		FileModTime: time.Unix(1608325405, 0),

		Content: string("query ($trader: String!, $market: String) {\n  serumFillHistory(trader: $trader, market: $market) {\n    pageInfo {\n      startCursor\n      endCursor\n    }\n    edges {\n      cursor\n      node {\n        orderId\n        side\n        market {\n          address\n          name\n        }\n        baseToken {\n          address\n          name\n        }\n        quoteToken {\n          address\n          name\n        }\n        lotCount\n        price\n        feeTier\n      }\n    }\n  }\n}\n"),
	}
	file6 := &embedded.EmbeddedFile{
		Filename:    "get_token.graphql",
		FileModTime: time.Unix(1608158051, 0),

		Content: string("query ($account: String!) {\n  token(address: $account) {\n    address\n    mintAuthority\n    freezeAuthority\n    supply\n    decimals\n  }\n}\n"),
	}
	file7 := &embedded.EmbeddedFile{
		Filename:    "stream_serum_instructions.graphql",
		FileModTime: time.Unix(1608158158, 0),

		Content: string("subscription ($account: String!) {\n  serumInstructionHistory(account: $account) {\n    instruction {\n      __typename\n      ... on UndecodedInstruction {\n        programIDIndex\n        accountCount\n        rawAccounts: accounts\n        dataLength\n        data\n        error\n      }\n      ... on SerumNewOrder {\n        side\n        limitPrice\n        maxQuantity\n        orderType\n        clientID\n        accounts {\n          market {\n            ...AccountFragment\n          }\n          openOrders {\n            ...AccountFragment\n          }\n          requestQueue {\n            ...AccountFragment\n          }\n          payer {\n            ...AccountFragment\n          }\n          owner {\n            ...AccountFragment\n          }\n          coinVault {\n            ...AccountFragment\n          }\n          pcVault {\n            ...AccountFragment\n          }\n          splTokenProgram {\n            ...AccountFragment\n          }\n          rent {\n            ...AccountFragment\n          }\n          srmDiscount {\n            ...AccountFragment\n          }\n        }\n      }\n      ... on SerumMatchOrder {\n        limit\n        accounts {\n          market {\n            ...AccountFragment\n          }\n          requestQueue {\n            ...AccountFragment\n          }\n          eventQueue {\n            ...AccountFragment\n          }\n          bids {\n            ...AccountFragment\n          }\n          asks {\n            ...AccountFragment\n          }\n          coinFeeReceivable {\n            ...AccountFragment\n          }\n          pcFeeReceivable {\n            ...AccountFragment\n          }\n        }\n      }\n      ... on SerumCancelOrder {\n        side\n        orderId\n        openOrders\n        openOrderSlot\n        accounts {\n          market {\n            ...AccountFragment\n          }\n          requestQueue {\n            ...AccountFragment\n          }\n          owner {\n            ...AccountFragment\n          }\n        }\n      }\n      ... on SerumSettleFunds {\n        __typename\n        accounts {\n          market {\n            ...AccountFragment\n          }\n          openOrders {\n            ...AccountFragment\n          }\n          owner {\n            ...AccountFragment\n          }\n          coinVault {\n            ...AccountFragment\n          }\n          pcVault {\n            ...AccountFragment\n          }\n          pcWallet {\n            ...AccountFragment\n          }\n          signer {\n            ...AccountFragment\n          }\n          splTokenProgram {\n            ...AccountFragment\n          }\n        }\n      }\n      ... on SerumCancelOrderByClientId {\n        clientID\n        accounts {\n          market {\n            ...AccountFragment\n          }\n          openOrders {\n            ...AccountFragment\n          }\n          requestQueue {\n            ...AccountFragment\n          }\n          owner {\n            ...AccountFragment\n          }\n        }\n      }\n    }\n  }\n}\n\nfragment AccountFragment on Account {\n  publicKey\n  isSigner\n  isWritable\n}\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1608237307, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file2, // "get_all_registered_tokens.graphql"
			file3, // "get_all_tokens.graphql"
			file4, // "get_registered_token.graphql"
			file5, // "get_serum_fill.graphql"
			file6, // "get_token.graphql"
			file7, // "stream_serum_instructions.graphql"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`examples`, &embedded.EmbeddedBox{
		Name: `examples`,
		Time: time.Unix(1608237307, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"": dir1,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"get_all_registered_tokens.graphql": file2,
			"get_all_tokens.graphql":            file3,
			"get_registered_token.graphql":      file4,
			"get_serum_fill.graphql":            file5,
			"get_token.graphql":                 file6,
			"stream_serum_instructions.graphql": file7,
		},
	})
}
