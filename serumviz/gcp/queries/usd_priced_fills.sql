SELECT
  ANY_VALUE(T1).*,
  ARRAY_AGG(STRUCT(T3.price) ORDER BY T3.slot_num DESC, T3.trx_idx DESC, T3.inst_idx DESC LIMIT 1)[OFFSET(0)] AS base_usd,
  ARRAY_AGG(STRUCT(T2.price) ORDER BY T2.slot_num DESC, T2.trx_idx DESC, T2.inst_idx DESC LIMIT 1)[OFFSET(0)] AS quote_usd,
FROM ${dataset}.priced_fills T1
LEFT JOIN ${dataset}.priced_fills T2
ON
  T2.base_address = T1.quote_address AND
  T2.quote_symbol IN ("USDT","USDC") AND
  (
    (T2.slot_num < T1.slot_num)
    OR (
      T2.slot_num = T1.slot_num AND
      T2.trx_idx <= T1.trx_idx AND
      T2.inst_idx <= T1.inst_idx
    )
  )
LEFT JOIN ${dataset}.priced_fills T3
ON
  T3.base_address = T1.base_address AND
  T3.quote_symbol IN ("USDT","USDC") AND
  (
    (T3.slot_num < T1.slot_num)
    OR (
      T3.slot_num = T1.slot_num AND
      T3.trx_idx <= T1.trx_idx AND
      T3.inst_idx <= T1.inst_idx
    )
  )
GROUP BY FORMAT('%t', T1)