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
* Renamed all the `mindreader-node-*` flags to `reader-node-*`
* Renamed all instances of `deepmind` to `firehose`
* Renamed `debug-deepmind` to `debug-firehose-logs`
* Renamed `dmlog` to `firelog`
* Renamed `DMLOG` prefix to `FIRE`
* 
