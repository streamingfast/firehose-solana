# Change log

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html). See [MAINTAINERS.md](./MAINTAINERS.md)
for instructions to keep up to date.

## v0.2.5

* bump firehose-core to `v0.1.11` with a regression fix for when a substreams has a start block in the reversible segment

## v0.2.4

### Changed
* bump firehose-core to `v0.1.10` with new metrics `substreams_active_requests` and `substreams_counter`

## v0.2.3

> [!IMPORTANT]
> The Substreams service exposed from this version will send progress messages that cannot be decoded by substreams clients prior to v1.1.12.
> Streaming of the actual data will not be affected. Clients will need to be upgraded to properly decode the new progress messages.

### Changed

* Bumped firehose-core to `0.1.8`
* Bumped substreams to `v1.1.12` to support the new progress message format. Progression now relates to **stages** instead of modules. You can get stage information using the `substreams info` command starting at version `v1.1.12`.
* Migrated to firehose-core
* change block reader-node block encoding from hex to base64

### Removed

*  Removed --substreams-tier1-request-stats and --substreams-tier1-request-stats (substreams request-stats are now always sent to clients)

### Fixed

* More tolerant retry/timeouts on filesource (prevent "Context Deadline Exceeded")

## v0.2.2-rc1

This release candidate is a hotfix for an issue introduced at v0.2.1 and affecting `production-mode` where the stream will hang and some `map_outputs` will not be produced over some specific ranges of the chains.

## v0.2.1

### Highlights

#### Substreams Scheduler Improvements for Parallel Processing

The `substreams` scheduler has been improved to reduce the number of required jobs for parallel processing. This affects `backprocessing` (preparing the states of modules up to a "start-block") and `forward processing` (preparing the states and the outputs to speed up streaming in production-mode).

Jobs on `tier2` workers are now divided in "stages", each stage generating the partial states for all the modules that have the same dependencies. A `substreams` that has a single store won't be affected, but one that has 3 top-level stores, which used to run 3 jobs for every segment now only runs a single job per segment to get all the states ready.


#### Substreams State Store Selection

The `substreams` server now accepts `X-Sf-Substreams-Cache-Tag` header to select which Substreams state store URL should be used by the request. When performing a Substreams request, the servers will optionally pick the state store based on the header. This enable consumers to stay on the same cache version when the operators needs to bump the data version (reasons for this could be a bug in Substreams software that caused some cached data to be corrupted on invalid).

To benefit from this, operators that have a version currently in their state store URL should move the version part from `--substreams-state-store-url` to the new flag `--substreams-state-store-default-tag`. For example if today you have in your config:

```yaml
start:
  ...
  flags:
    substreams-state-store-url: /<some>/<path>/v3
```

You should convert to:

```yaml
start:
  ...
  flags:
    substreams-state-store-url: /<some>/<path>
    substreams-state-store-default-tag: v3
```

### Operators Upgrade

* The app `substreams-tier1` and `substreams-tier2` should be upgraded concurrently. Some calls will fail while versions are misaligned.

* Remove the flag `--substreams-tier1-subrequests-size` from your config, it is not used anymore.


### Backend Changes

* Authentication plugin `trust` can now specify an exclusive list of `allowed` headers (all lowercase), ex: `trust://?allowed=x-sf-user-id,x-sf-api-key-id,x-real-ip,x-sf-substreams-cache-tag`

* The `tier2` app no longer uses the `common-auth-plugin`, `trust` will always be used, so that `tier1` can pass down its headers (ex: `X-Sf-Substreams-Cache-Tag`).

* Added support for *continuous authentication* via the grpc auth plugin (allowing cutoff triggered by the auth system).


## v0.2.0

### BREAKING CHANGES

#### Substreams protocol change
* Bumps substreams from v1.0.x to v1.1.1 -> RPC protocol changed from sf.substreams.v1.Stream/Blocks to sf.substreams.rpc.v2.Stream/Blocks. See release notes from github.com/streamingfast/substreams for details.

### Added

* Added support for "requester pays" buckets on Google Storage in url, ex: `gs://my-bucket/path?project=my-project-id`

### Changed

* Config value `substreams-stores-save-interval` and `substreams-output-cache-save-interval` have been merged together as a single value to avoid potential bugs that would arise when the value is different for those two. The new configuration value is called `substreams-cache-save-interval`.

    *  To migrate, remove usage of `substreams-stores-save-interval: <number>` and `substreams-output-cache-save-interval: <number>` if defined in your config file and replace with `substreams-cache-save-
interval: <number>`, if you had two different value before, pick the biggest of the two as the new value to put. We are currently setting to `1000` for Ethereum Mainnet.

* Updated to Substreams `v0.2.0`, please refer to [release page](https://github.com/streamingfast/substreams/releases/tag/v0.2.0) for further info about Substreams changes.

* Updated `--substreams-output-cache-save-interval` default value to 1000.

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

### Project Rename

* The repo name has changed from `sf-solana` to `firehose-solana`
* The binary name has changed from `sfsol` to `firesol` (aligned with https://firehose.streamingfast.io/references/naming-conventions)

### Flags and environment variables rename
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

