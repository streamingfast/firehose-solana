# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.

## unreleased
### BREAKING CHANGES

### Added
* Added `tools bt blocks  --bt-project=<bigtable_project> --bt-instance=<bigtable_instance> <start-block-num> <stop-block-num>` command to scan bigtable rows
  * Added `--firehose-enabled` flag to output FIRE log

* Added `reader-bt` application to sync directly from bigtable
  * Added `--reader-bt-readiness-max-latency` flag
  * Added `--reader-bt-data-dir` flag
  * Added `--reader-bt-debug-firehose-logs` flag
  * Added `--reader-bt-log-to-zap` flag
  * Added `--reader-bt-shutdown-delay` flag
  * Added `--reader-bt-working-dir` flag
  * Added `--reader-bt-blocks-chan-capacity` flag
  * Added `--reader-bt-one-block-suffix` flag
  * Added `--reader-bt-startup-delay` flag
  * Added `--reader-bt-grpc-listen-addr` flag


### Removed
* Removed `dgraphql` application and all associated flags
* Removed `tools reproc` replaced with `tools bt blocks`

#### Project Rename

* The repo name has changed from `sf-solana` to `firehose-solana`
* The binary name has changed from `sfsol` to `firesol` (aligned with https://firehose.streamingfast.io/references/naming-conventions)

#### Flags and environment variables
* All config via environment variables that started with `SFSOL_` now starts with `FIRESOL_`
* Changed `config-file` default from `./sf.yaml` to `""`, preventing failure without this flag.
* Renamed `common-blocks-store-url` to `common-merged-blocks-store-url`
* Renamed `common-oneblock-store-url` to `common-one-block-store-url`
* Renamed `common-blockstream-addr` to `common-live-blocks-addr`
* Renamed `common-protocol-first-streamable-block` to `common-first-streamable-block`
* Added `common-forked-blocks-store-url`

* Renamed the `mindreader` application to `reader`
  * Renamed `mindreaderPlugin` to `readerPlugin`
  
* Renamed all the `mindreader-node-*` flags to `reader-node-*`
  * Renamed `mindreader-node-start-block-num` to `reader-node-start-block-num`
  * Renamed `mindreader-node-stop-block-num` to `reader-node-stop-block-num`
  * Renamed `mindreader-node-blocks-chan-capacity` to `reader-node-blocks-chan-capacity`
  * Renamed `mindreader-node-wait-upload-complete-on-shutdown` to `reader-node-wait-upload-complete-on-shutdown`
  * Renamed `mindreader-node-oneblock-suffix` to `reader-node-one-block-suffix`
  * Renamed `mindreader-node-deepmind-batch-files-path` to `reader-node-firehose-batch-files-path`
  * Renamed `mindreader-node-purge-account-data` to `reader-node-purge-account-data`
  * Added `reader-node-arguments`
  * Removed `reader-node-merge-and-store-directly`
  * Removed `reader-node-block-data-working-dir`
  * Removed `reader-node-extra-arguments`
  * Removed `reader-node-merge-threshold-block-age`
   
  
* Renamed all instances of `deepmind` to `firehose`
  * Renamed `path-to-deepmind-batch-files` to `path-to-firehose-batch-files`
  * Renamed `mindreader-node-deepmind-batch-files-path` to `reader-node-firehose-batch-files-path`
  
* Renamed `debug-deepmind` to `debug-firehose-logs`
  * Renamed `mindreader-node-debug-deep-mind` to `reader-node-debug-firehose-logs`

* Renamed `dmlog` to `firelog`
  * Flag `<path_to_dmlog.dmlog>` changed to `<path_to_firelog.firelog>`
* Renamed `DMLOG` prefix to `FIRE`

* Added/Removed `merger-*` flags
  * Removed `merger-writers-leeway`
  * Removed `merger-one-block-deletion-threads`
  * Removed `merger-max-one-block-operations-batch-size`
  * Added `merger-time-between-store-pruning`
  * Added `merger-prune-forked-blocks-after`
  * Added `merger-stop-block`

* Added/Removed `firehose-*` flags
  * Removed `firehose-blocks-store-urls`
  * Removed `firehose-real-time-tolerance`
  * Removed `firehose-blocks-store-urls`
  * Removed `firehose-real-time-tolerance`

* Removed `relayer-*` flags
  * Removed `relayer-source-request-burst`
  * Removed `relayer-merger-addr`
  * Removed `relayer-buffer-size`
  * Removed `relayer-min-start-offset`

