WITH priced_fills AS
(SELECT
  m.name,
  CASE
    WHEN f.side = 'ASK' THEN
      f.native_qty_paid / POWER(10, base_t.decimals)
    ELSE
      f.native_qty_received / POWER(10, base_t.decimals)
  END AS base_amount,
  CASE
    WHEN f.side = 'ASK' THEN
      f.native_qty_received / POWER(10, quote_t.decimals)
    ELSE
      f.native_qty_paid / POWER(10, quote_t.decimals)
  END AS quote_amount,
  base_t.address AS base_address,
  base_t.decimals AS base_decimals,
  base_t.symbol AS base_symbol,
  quote_t.address AS quote_address,
  quote_t.decimals AS quote_decimals,
  quote_t.symbol AS quote_symbol,
  f.*,
FROM ${dataset}.fills AS f
  LEFT JOIN serum.markets AS m ON f.market = m.address
  LEFT JOIN serum.tokens AS base_t ON m.base_token = base_t.address
  LEFT JOIN serum.tokens AS quote_t ON m.quote_token = quote_t.address
)
SELECT
  quote_amount / base_amount as price,
  base_amount,
  base_decimals,
  base_symbol,
  quote_amount,
  quote_decimals,
  quote_symbol,
  base_address,
  quote_address,
  market as market_address,
  timestamp,
  slot_num,
  trx_idx,
  inst_idx,
FROM priced_fills;