# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.


## v1.0.0


#### Project Rename

* The repo name has changed from `sf-solana` to `firehose-solana`
* The binary name has changed from `sfsol` to `firesol` (aligned with https://firehose.streamingfast.io/references/naming-conventions)

#### Flags and environment variables
* Renamed `common-blocks-store-url` to `common-merged-blocks-store-url`
* Renamed `common-oneblock-store-url` to `common-one-block-store-url`
* Renamed `common-blockstream-addr` to `common-live-blocks-addr`
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
  
* Renamed all instances of `deepmind` to `firehose`
  * Renamed `path-to-deepmind-batch-files` to `path-to-firehose-batch-files`
  * Renamed `mindreader-node-deepmind-batch-files-path` to `reader-node-firehose-batch-files-path`
  
* Renamed `debug-deepmind` to `debug-firehose-logs`
  * Renamed `mindreader-node-debug-deep-mind` to `reader-node-debug-firehose-logs`

* Renamed `dmlog` to `firelog`
  * Flag `<path_to_dmlog.dmlog>` changed to `<path_to_firelog.firelog>`
* Renamed `DMLOG` prefix to `FIRE`

