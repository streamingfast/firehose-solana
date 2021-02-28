package schemas

import (
	"time"

	"github.com/GeertJohan/go.rice/embedded"
)

func init() {

	// define files
	file2 := &embedded.EmbeddedFile{
		Filename:    "fills.json",
		FileModTime: time.Unix(1614523347, 0),

		Content: string("[\n  {\n    \"name\": \"trader\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"market\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"order_id\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"side\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"maker\",\n    \"type\": \"BOOLEAN\"\n  },\n  {\n    \"name\": \"native_qty_paid\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"native_qty_received\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"native_fee_or_rebate\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"fee_tier\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"timestamp\",\n    \"type\": \"TIMESTAMP\"\n  },\n  {\n    \"name\": \"slot_num\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"slot_hash\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trx_id\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trx_idx\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"inst_idx\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"order_seq_num\",\n    \"type\": \"INTEGER\"\n  }\n]"),
	}
	file3 := &embedded.EmbeddedFile{
		Filename:    "markets.json",
		FileModTime: time.Unix(1614305982, 0),

		Content: string("[\n  {\n    \"name\": \"name\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"address\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"deprecated\",\n    \"type\": \"BOOLEAN\"\n  },\n  {\n    \"name\": \"program_id\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"base_token\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"quote_token\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"base_lot_size\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"quote_lot_size\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"request_queue\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"event_queue\",\n    \"type\": \"STRING\"\n  }\n]\n"),
	}
	file4 := &embedded.EmbeddedFile{
		Filename:    "orders.json",
		FileModTime: time.Unix(1614305982, 0),

		Content: string("[\n  {\n    \"name\": \"num\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"market\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trader\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"side\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"limit_price\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"max_quantity\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"type\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"timestamp\",\n    \"type\": \"TIMESTAMP\"\n  },\n  {\n    \"name\": \"slot_num\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"slot_hash\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trx_id\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trx_idx\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"inst_idx\",\n    \"type\": \"INTEGER\"\n  }\n]\n"),
	}
	file5 := &embedded.EmbeddedFile{
		Filename:    "processed_files.json",
		FileModTime: time.Unix(1614305982, 0),

		Content: string("[\n  {\n    \"name\": \"table\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"file\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"timestamp\",\n    \"type\": \"TIMESTAMP\"\n  }\n]\n"),
	}
	file6 := &embedded.EmbeddedFile{
		Filename:    "tokens.json",
		FileModTime: time.Unix(1614305982, 0),

		Content: string("[\n  {\n    \"name\": \"name\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"symbol\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"address\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"mint_authority_option\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"mint_authority\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"supply\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"decimals\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"is_initialized\",\n    \"type\": \"BOOLEAN\"\n  },\n  {\n    \"name\": \"freeze_authority_option\",\n    \"type\": \"INTEGER\"\n  },\n  {\n    \"name\": \"freeze_authority\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"verified\",\n    \"type\": \"BOOLEAN\"\n  }\n]\n"),
	}
	file7 := &embedded.EmbeddedFile{
		Filename:    "traders.json",
		FileModTime: time.Unix(1614305982, 0),

		Content: string("[\n  {\n    \"name\": \"account\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"trader\",\n    \"type\": \"STRING\"\n  },\n  {\n    \"name\": \"slot_num\",\n    \"type\": \"INTEGER\"\n  }\n]\n"),
	}

	// define dirs
	dir1 := &embedded.EmbeddedDir{
		Filename:   "",
		DirModTime: time.Unix(1614523347, 0),
		ChildFiles: []*embedded.EmbeddedFile{
			file2, // "fills.json"
			file3, // "markets.json"
			file4, // "orders.json"
			file5, // "processed_files.json"
			file6, // "tokens.json"
			file7, // "traders.json"

		},
	}

	// link ChildDirs
	dir1.ChildDirs = []*embedded.EmbeddedDir{}

	// register embeddedBox
	embedded.RegisterEmbeddedBox(`V1`, &embedded.EmbeddedBox{
		Name: `V1`,
		Time: time.Unix(1614523347, 0),
		Dirs: map[string]*embedded.EmbeddedDir{
			"": dir1,
		},
		Files: map[string]*embedded.EmbeddedFile{
			"fills.json":           file2,
			"markets.json":         file3,
			"orders.json":          file4,
			"processed_files.json": file5,
			"tokens.json":          file6,
			"traders.json":         file7,
		},
	})
}
