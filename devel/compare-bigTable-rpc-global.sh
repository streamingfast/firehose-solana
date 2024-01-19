#!/bin/bash
echo "STARTING GLOBAL COMPARING!"

# Initialize these variables as needed
runningTimes=10
current_block=200000000
eachNumberOfBlocks=1000000
stop_block=240000000
rpc_endpoint="https://frequent-sparkling-hill.solana-mainnet.quiknode.pro/ff194d392c35948e3ef04003d141cda78dbf9733/"
reference_storage="gs://dfuseio-global-blocks-uscentral/sol-mainnet/v1?project=dfuseio-global"
current_storage="file:///Users/arnaudberger/t/data/storage/merged-blocks"
function fetch_rpc_blocks_for_range() {
  local rpc_endpoint=$1
  local start_block=$2
  rm -rf ~/t/data/

  firecore start reader-node merger -c "" --data-dir=/Users/arnaudberger/t/data --reader-node-data-dir=/Users/arnaudberger/t/data --reader-node-path=firesol --reader-node-arguments="fetch rpc ${rpc_endpoint} ${start_block} --state-dir /Users/arnaudberger/t/data" --common-first-streamable-block=${start_block} > ~/t/firecore_output.txt 2>&1 &
  local firecore_pid=$!

  tail -f ~/t/firecore_output.txt | while read -r line; do
    echo "$line"
    if [[ "$line" == *"merged and uploaded"* ]]; then
      kill $firecore_pid
      break
    fi
  done

  # Clean up
  rm ~/t/firecore_output.txt
}

function compare_reference_rpc_for_range() {
  local range=$1
  local reference_storage=$2
  local current_storage=$3
  firesol tools compare-blocks ${reference_storage} ${current_storage}  ${range} --ignore-error-when-JSON-matches --diff
}

for i in $(seq 1 $runningTimes); do
  current_start_block=$((current_block + eachNumberOfBlocks * i))
  current_stop_block=$((current_start_block + 100))

  block_range="${current_start_block}:${current_stop_block}"

  echo "Fetching blocks from rpc for range $block_range"
  fetch_rpc_blocks_for_range "$rpc_endpoint" "$current_start_block"

  echo "Comparing merged rpc blocks for range $block_range with reference blocks"
  compare_reference_rpc_for_range "$block_range" "$reference_storage" "$current_storage"

  # Break the loop if the end of the block range is reached
  if [ $current_stop_block -eq $stop_block ]; then
      break
  fi

  echo "COMPARING SUCCESSFULLY FINISHED FOR RANGE $block_range"
done
