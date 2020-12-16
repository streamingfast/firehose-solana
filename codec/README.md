

To ensure deterministic ordering of the transactions, we'll want to have something like:

```
[SLOT1]

  [Entry: 1 trx]
DMLOG next batch
  [Entry: 1 trx]
DMLOG next batch
  [Entry: 2 trx]
DMLOG next batch
  [Entry: 0 trx]
DMLOG next batch
  [Entry: 25 trx]
DMLOG next batch
  [Entry: 25 trx]
DMLOG next batch
  [Entry: 0 trx]

[SLOT2]
```

so that between each batch, when parallel execution ends, we can use the
transaction IDs that _were_ executed in parallel, sort them based on
the transaction ID, so in a deterministic ordering.
