#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfusesol="$ROOT/../dfusesol"

main() {
  pushd "$ROOT" &> /dev/null

  clean=
  network="mainnet-beta"

  while getopts "hcn:" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      n) network="$OPTARG";;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $network == "" || $network == "development" ]]; then
    usage_error "The network value '$network' is invalid."
  fi

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  exec $dfusesol -c $(basename $ROOT).yaml start --mindreader-node-network="$network" "$@"
}

usage_error() {
  message="$1"
  exit_code="$2"

  echo "ERROR: $message"
  echo ""
  usage
  exit ${exit_code:-1}
}

usage() {
  echo "usage: start.sh [-c] [-n <network>]"
  echo ""
  echo "Start $(basename $ROOT) environment syncing mindreader with the pre-defined <network>. When nothing is specified,"
  echo "sync with 'mainnet-beta' network."
  echo ""
  echo "Available networks:"
  echo "  mainnet-beta"
  echo "  testnet"
  echo "  devnet"
  echo "  custom"
  echo ""
  echo "When providing 'custom', you must manully provide the extra flags required for mindreader to know where"
  echo "to connect to:"
  echo '  start.sh custom -- --mindreader-node-extra-arguments="--entrypoint <value> --trusted-validator <value1> --trusted-validator <value2>... --expected-genesis-hash <value>"'
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -n <network>   Actual network to connect to, values can be any '--mindreader-node-network' accepted value (expect 'development' which will not work properly if chosen)"
}

main "$@"