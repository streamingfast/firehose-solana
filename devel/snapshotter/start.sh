#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfusesol="$ROOT/../dfusesol"

main() {
  pushd "$ROOT" &> /dev/null

  clean=

  while getopts "hc" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

  exec $dfusesol -c $(basename $ROOT).yaml start "$@"
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
  echo "usage: start.sh [-c]"
  echo ""
  echo "Start $(basename $ROOT)"
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "Examples (grpc)"
  echo '    Stream Blocks'

}

main "$@"