SELECT
    DISTINCT (slot_num),
    timestamp
FROM
 ${dataset}.fills
ORDER BY slot_num ASC