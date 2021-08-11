SELECT
    CASE
        WHEN base_usd.price > 0 AND quote_usd.price > 0 THEN
            ((base_usd.price * base_amount) + (quote_usd.price * quote_amount))/2
        WHEN base_usd.price > 0 THEN
            (base_usd.price * base_amount)
        WHEN quote_usd.price > 0 THEN
            (quote_usd.price * quote_amount)
        ELSE
            0
    END as usd_volume,
    market_address,
    base_address,
    quote_address,
    timestamp,
    slot_num,
    trx_idx,
    inst_idx
FROM
    ${dataset}.usd_priced_fills