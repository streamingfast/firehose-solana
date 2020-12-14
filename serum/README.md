

GraphQL query:
```
query {
  // Return last 100 fills for the given pubkey, irreversible only.
  serumFillHistory(pubkey: String!, market: String) [SerumFill]
}

type SerumFill {
  pubkey String
  orderId String

  baseToken Token
  quoteToken Token
  side MarketSide

  market SerumMarket
  size String  // Amount
  price String

  feeTier SerumFeeTier
}

enum SerumFeeTier {
  Base
  SRM2
  SRM3
  SRM4
  SRM5
  SRM6
  MSRM
}

enum MarketSide {
  ASK
  BID
}

type SerumMarket  {
  address String
  name String

  // phase 2
  coinToken Token
  pcToken Token
}

type Token {
  name String
  address String
}

```


EventQueue:
  write:
    orders:[market]:[order_seq_num]:[rev_slot_num] => FillData(side)

NewOrder:
  write:
    // to query all markets, for a pubkey
    order_pubkey:[pubkey]:[rev_slot_num]:[market]:[rev_order_seq_num] => nil
    // to query a single market for a given pubkey
    order_market:[market]:[pubkey]:[rev_slot_num]:[rev_order_seq_num] => nil

LastWrittenBlock:
  write:
    last_written_block => PB:slot_num:slot_id



Query with only a pubkey:

  Scan:
    START: order_pubkey:AGIVENPUBKEY:000000000000000
    END:   order_pubkey:AGIVENPUBKEY;
    LIMIT: 100

Query with a pubkey and market:

  Scan:
    START: order_market:AGIVENMARKET:AGIVENPUBKEY:000000000000000
    END:   order_market:AGIVENMARKET:AGIVENPUBKEY;
    LIMIT: 100

Next page query for a given pubkey, with a cursor of the last order_seq_num returned:

  Scan:
    START: order_pubkey:AGIVENPUBKEY:00000000f232123 <- based on cursor
    END:   order_pubkey:AGIVENPUBKEY;
    LIMIT: 100

---

* mindreader node, writing merged files and serving relayer-style endpoint
* Firehose dans `dfuse-solana`
* deploy firehose that connects to mindreader node + blocks storage.
* Write `serum/` injector service, based on firehose
  * Writes to the KVDB
  * Whip up the `serum-injector` app
* `serum/client` to query fills
* dgraphql integration to do the `SerumClient::Fills` query.
* dgraphql integration to resolve market and token metadata
