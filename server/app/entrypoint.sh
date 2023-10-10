#!/usr/bin/env bash
set -eEuo pipefail

_shutdown() {
  trap '' TERM INT ERR

  kill 0
  wait

  exit 0
}

trap _shutdown TERM INT ERR

nats-server -c nats-server.conf &
wait
echo "nats-server exited! sleep 10"
sleep 10
