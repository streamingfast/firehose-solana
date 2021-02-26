SELECT
    DISTINCT (slot_num),
    timestamp
FROM
 serum.fills
ORDER BY slot_num ASC