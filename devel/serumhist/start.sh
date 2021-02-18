#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

dfusesol="$ROOT/../dfusesol"

main() {
  pushd "$ROOT" &> /dev/null

  clean=
  force_injection=

  while getopts "hci" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      i) force_injection=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf dfuse-data &> /dev/null || true
  fi

#  if [[ $force_injection == "true" ]]; then
     echo "running serumhist injector"
#     KILL_AFTER=${KILL_AFTER:-15} $dfusesol -c injector.yaml start "$@"
     exec $dfusesol -c injector.yaml start "$@"
#  fi

#  echo "running serumhist server"
#  exec $dfusesol -c server.yaml start "$@"
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
  echo "usage: start.sh [-c] [-i]"
  echo ""
  echo "Start $(basename $ROOT) environment starting dgraphql connect to Mainnet (Beta) network."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -i             Starts injector"
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    TRACE=true     Enables traces"
  echo "Examples"
  echo " Find Keys with Prefix     dfusesol tools kv prefix 02 --dsn=badger:///Users/julien/codebase/dfuse-io/dfuse-solana/devel/serumhist/dfuse-data/storage/serumhist"
}

main "$@"
