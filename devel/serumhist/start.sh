#!/usr/bin/env bash

ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

sfsol="$ROOT/../sfsol"

main() {
  pushd "$ROOT" &> /dev/null

  clean=
  start_injection=
  force_only_injection=

  while getopts "hcif" opt; do
    case $opt in
      h) usage && exit 0;;
      c) clean=true;;
      i) start_injection=true;;
      f) force_only_injection=true;;
      \?) usage_error "Invalid option: -$OPTARG";;
    esac
  done
  shift $((OPTIND-1))
  [[ $1 = "--" ]] && shift

  if [[ $clean == "true" ]]; then
    rm -rf sf-data &> /dev/null || true
  fi

  if [[ $start_injection == "true" ]] || [[ $force_only_injection == "true" ]]; then
     if [[ $force_only_injection == "true" ]]; then
       echo "Running only serumhist injector"
       exec $sfsol -c injector.yaml start "$@"
     else
       echo "Running serumhist injector for 15 seconds"
       KILL_AFTER=${KILL_AFTER:-15} $sfsol -c injector.yaml start "$@"
     fi
  fi
  if [[ force_only_injection != "true" ]]; then
    echo "running serumhist server"
    exec $sfsol -c server.yaml start "$@"
  fi
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
  echo "usage: start.sh [-c] [-i] [-f]"
  echo ""
  echo "Start $(basename $ROOT) environment starting dgraphql connect to Mainnet (Beta) network."
  echo ""
  echo "Options"
  echo "    -c             Clean actual data directory first"
  echo "    -i             Starts injector"
  echo "    -f             Force to only injector mode"
  echo "Environment"
  echo "    INFO=<app>     Turn info logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    DEBUG=<app>    Turn debug logs for <app> (multiple separated by ','), accepts app name or regexp (.* for all)"
  echo "    TRACE=true     Enables traces"
  echo "Examples"
  echo " Find Fill Keys             sfsol tools kv prefix 01 --dsn=badger:///full/path"
  echo " Decode a serumhist key     sfsol tools serumhist keyer decode 01e77dd955f7b84e4736d066e97188f1919ba42d8b4b090752c1d236c3666cf634fffffffffc13258affffffffffffff99fffffffffffffffcffffffffffec00b3"
  echo " Get An Order               sfsol tools serumhist orders get GaeUpY7CT8rjoeVGjY1t3mJJDd1bdXxYWtrGSpsVFors  1310540  --dsn=badger:///full/path"
  echo " Get Fill by market         sfsol tools serumhist orders get GaeUpY7CT8rjoeVGjY1t3mJJDd1bdXxYWtrGSpsVFors --dsn=badger:///full/path"
}

main "$@"
