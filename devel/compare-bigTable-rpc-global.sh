#!/bin/bash
echo "STARTING GLOBAL COMPARING!"

# Initialize these variables as needed
runningTimes=10
current_block=1690600
eachNumberOfBlocks=10000000
stop_block=240000000

source ../.envrc

rpc_endpoint="$RPC_ENDPOINT"
reference_storage="$REFERENCE_STORAGE"
current_storage="$CURRENT_STORAGE"

# Now use the variables directly
echo "RPC Endpoint: ${rpc_endpoint}"
echo "Reference Storage: $REFERENCE_STORAGE"
echo "Current Storage: $CURRENT_STORAGE"

function fetch_rpc_blocks_for_range() {
  local rpc_endpoint=$1
  local start_block=$2
  local stop_block=$3
  rm -rf ~/t/data/

  firecore start reader-node merger -c ""  --merger-stop-block ${stop_block} --data-dir=/Users/arnaudberger/t/data --reader-node-data-dir=/Users/arnaudberger/t/data --reader-node-path=firesol --reader-node-arguments="fetch rpc "${rpc_endpoint}" ${start_block} --state-dir /Users/arnaudberger/t/data" --common-first-streamable-block=${start_block}
}

function compare_reference_rpc_for_range() {
  local range=$1
  local reference_storage=$2
  local current_storage=$3
  firesol tools compare-blocks ${reference_storage} ${current_storage}  ${range}
}

for i in $(seq 1 $runningTimes); do
  current_start_block=$((current_block + eachNumberOfBlocks * i))
  current_stop_block=$((current_start_block + 100))

  block_range="${current_start_block}:${current_stop_block}"

  echo "Fetching blocks from rpc for range $block_range"
  fetch_rpc_blocks_for_range "$rpc_endpoint" "$current_start_block" "$current_stop_block"

  echo "Comparing merged rpc blocks for range $block_range with reference blocks"
  compare_reference_rpc_for_range "$block_range" "$reference_storage" "$current_storage"

  # Break the loop if the end of the block range is reached
  if [ $current_stop_block -eq $stop_block ]; then
      break
  fi

  echo "COMPARING SUCCESSFULLY FINISHED FOR RANGE $block_range"
done
