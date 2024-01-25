#!/bin/bash
echo "STARTING GLOBAL COMPARING!"

# Initialize these variables as needed
runningTimes=10
current_block=11690600
eachNumberOfBlocks=10000000
stop_block=240000000

source ../.envrc

rpc_endpoint="$RPC_ENDPOINT"
reference_storage="$REFERENCE_STORAGE"
current_storage="$CURRENT_STORAGE"
data_dir="$DATA_DIR"

function fetch_rpc_blocks_for_range() {
  local rpc_endpoint=$1
  local start_block=$2
  local stop_block=$3
  rm -rf ~/t/data/

  #Call the rpc here to check if the start_block is skipped, if it is, then start block is equal to start_block +1. If the error code is -32009 or -32007


  while :; do
          json_data='
            {
              "jsonrpc": "2.0",
              "id": 1,
              "method": "getBlock",
              "params": [
                '${start_block}',
                {
                  "encoding": "json",
                  "maxSupportedTransactionVersion": 0,
                  "transactionDetails": "full",
                  "rewards": false
                }
              ]
            }
            '
          json_response=$(curl -s "${rpc_endpoint}" -X POST -H "Content-Type: application/json" -d "$json_data")
          error_code=$(echo "${json_response}" | jq '.error.code')

          if [ "${error_code}" = "-32009" ] || [ "${error_code}" = "-32007" ]; then
              start_block=$((start_block + 1))
          else
              break
          fi
      done

  firecore start reader-node merger -c ""  --merger-stop-block ${stop_block} --data-dir=/Users/arnaudberger/t/data --reader-node-data-dir=/Users/arnaudberger/t/data --reader-node-path=firesol --reader-node-arguments="fetch rpc "${rpc_endpoint}" ${start_block} --state-dir ${data_dir}" --common-first-streamable-block=${start_block}

}

function compare_reference_rpc_for_range() {
  local range=$1
  local reference_storage=$2
  local current_storage=$3
  firesol tools compare-blocks ${reference_storage} ${current_storage}  ${range} --diff
}

for i in $(seq 0 $runningTimes); do
  current_start_block=$((current_block + eachNumberOfBlocks * i))
  current_stop_block=$((current_start_block + 100))

  block_range="${current_start_block}:${current_stop_block}"

  echo "Fetching blocks from rpc for range $block_range"
  fetch_rpc_blocks_for_range "$rpc_endpoint" "$current_start_block" "$current_stop_block"

  echo "Comparing merged rpc blocks for range $block_range with reference blocks"
  compare_reference_rpc_for_range "$block_range" "$reference_storage" "$current_storage"

  if [ $current_stop_block -gt $stop_block ]; then
      break
  fi

  echo "COMPARING SUCCESSFULLY FINISHED FOR RANGE $block_range"
done
