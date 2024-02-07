#!/usr/bin/env jq -f

def head:
  .[0:1];

def tail:
  .[1:];

def capitalize:
  (head | ascii_upcase) + tail;

def snake_to_camel:
  split("_") |
  head + (tail | map(capitalize)) |
  join("");

def camel_to_snake:
  [
    splits("(?=[A-Z])")
  ]
  |map(
    select(. != "")
    | ascii_downcase
  )
  | join("_");

def map_keys(mapper):
  walk(if type == "object" then with_entries(.key |= mapper) else . end);

map_keys(camel_to_snake)