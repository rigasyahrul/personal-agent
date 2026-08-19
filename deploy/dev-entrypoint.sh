#!/bin/sh
set -eu

cleanup() {
  trap - INT TERM EXIT
  kill "${vite_pid:-}" "${air_pid:-}" 2>/dev/null || true
  wait "${vite_pid:-}" "${air_pid:-}" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

npm --prefix web ci
npm --prefix web run dev &
vite_pid=$!
air -c deploy/air.toml &
air_pid=$!
while kill -0 "$vite_pid" 2>/dev/null && kill -0 "$air_pid" 2>/dev/null; do sleep 1; done
exit 1
