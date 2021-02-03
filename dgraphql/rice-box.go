package dgraphql

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file2 := &embedded.EmbeddedFile{
		Filename:    ".prettierrc",
		FileModTime: time.Unix(1611886197, 0),

		Content: string("{}"),
	}
	file3 := &embedded.EmbeddedFile{
		Filename:    "get_all_tokens.graphql",
		FileModTime: time.Unix(1612206158, 0),

		Content: string("query($cursor: String!) {\n  tokens(cursor: $cursor) {\n    pageInfo {\n      hasNextPage\n      startCursor\n      endCursor\n    }\n    edges {\n      cursor\n      node {\n        address\n        mintAuthority\n        freezeAuthority\n        supply\n        decimals\n        verified\n        meta {\n          name\n          symbol\n          logo\n          website\n        }\n      }\n    }\n  }\n}"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "get_serum_fill.graphql",
		FileModTime: time.Unix(1612375422, 0),

		Content: string("query($trader: String!, $market: String) {\n  serumFillHistory(trader: $trader, market: $market) {\n    pageInfo {\n      startCursor\n      endCursor\n    }\n    edges {\n      cursor\n      node {\n        slotNum\n        transactionIndex\n        instructionIndex\n        trader\n        side\n        market {\n          address\n          name\n          baseToken {\n            address\n            meta {\n              name\n              symbol\n            }\n          }\n          quoteToken {\n            address\n            meta {\n              name\n              symbol\n            }\n          }\n        }\n        quantityReceived {\n          display\n          value\n        }\n        quantityPaid {\n          display\n          value\n        }\n        price\n        feeTier\n      }\n    }\n  }\n}\n"),
	}
	file5 := &embedded.EmbeddedFile{
		Filename:    "get_serum_markets.graphql",
		FileModTime: time.Unix(1612375422, 0),

		Content: string("{\n  serumMarkets(count: 100) {\n    pageInfo {\n      startCursor\n      endCursor\n      hasNextPage\n    }\n    edges {\n      cursor\n      node {\n        address\n        name\n        baseToken {\n          address\n          decimals\n          supply\n          meta {\n            name\n            symbol\n          }\n        }\n        quoteToken {\n          address\n          decimals\n          supply\n          meta {\n            name\n            symbol\n          }\n        }\n      }\n    }\n  }\n}\n"),
	}
	file6 := &embedded.EmbeddedFile{
		Filename:    "get_token.graphql",
		FileModTime: time.Unix(1612211908, 0),

		Content: string("query($address: String!){\n  token(address: $address) {\n    cursor\n    node {\n      address\n      mintAuthority\n      freezeAuthority\n      supply\n      decimals\n      verified\n      meta {\n        name  \n        symbol\n        logo\n        website\n      }\n    }\n  }\n}\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1612375422, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file2, // ".prettierrc"
			file3, // "get_all_tokens.graphql"
			file4, // "get_serum_fill.graphql"
			file5, // "get_serum_markets.graphql"
			file6, // "get_token.graphql"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`examples`, &embedded.EmbeddedBox{
		Name: `examples`,
		Time: time.Unix(1612375422, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"": dir1,
		},
		Files: map[string]*embedded.EmbeddedFile{
			".prettierrc":               file2,
			"get_all_tokens.graphql":    file3,
			"get_serum_fill.graphql":    file4,
			"get_serum_markets.graphql": file5,
			"get_token.graphql":         file6,
		},
	})
}
