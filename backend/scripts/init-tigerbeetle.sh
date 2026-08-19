#!/bin/sh
# Formats the TigerBeetle data file once — idempotent, since the file
# persists in the tigerbeetle_data volume across restarts, so this is a
# no-op check on every boot after the first — then starts the server.
set -e

DATA_FILE="/data/0.tigerbeetle"

if [ ! -f "$DATA_FILE" ]; then
  echo "Formatting TigerBeetle data file..."
  /tigerbeetle format --cluster=0 --replica=0 --replica-count=1 "$DATA_FILE"
fi

exec /tigerbeetle start --addresses=0.0.0.0:3000 "$DATA_FILE"
