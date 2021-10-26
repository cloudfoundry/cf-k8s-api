#!/bin/bash

set -euo pipefail

wait-for-daemon() {
  while ! docker ps; do
    echo waiting for docker daemon to start...
    sleep 1
  done
}

dockerd-entrypoint.sh &
wait-for-daemon

$@
