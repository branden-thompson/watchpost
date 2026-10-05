#!/usr/bin/env sh
# workload.sh — W14's standard workload v1 (0.18.0 D-154): the one defined
# workload every before-and-after number is taken on. perf-measurement.md says
# why: the missing piece was never a tool, it was that two runs on different
# workloads could not be compared.
#
#   scripts/quality/workload.sh session <default|heavy> <outdir> [phase_minutes=10]
#   scripts/quality/workload.sh cold    <default|heavy> <outdir> [runs=5]
#   scripts/quality/workload.sh warm    <default|heavy> <outdir> [runs=4]
#
# Runs ./dist/watchpost (make build first) under a SCRATCH HOME holding
# 06_docs/perf/workload-v1/config-<variant>.toml, so the listener's own config
# and caches are never read or written; a cold run gets an empty HOME each
# time. WATCHPOST_DEBUG_TIMING=1 turns the timing instrument on (M5, M6), and
# WATCHPOST_DEBUG_PPROF=1 the counters it is read from. A session is sampled
# beside the driver by soak.sh every 20 s. Live network: the run records when
# and where it ran, and a comparison is between runs of the same workload.
set -eu
mode=${1:?session|cold|warm}; variant=${2:?default|heavy}; out=${3:?outdir}; arg=${4:-}
root=$(git rev-parse --show-toplevel)
cfg=$root/06_docs/perf/workload-v1/config-$variant.toml
bin=$root/dist/watchpost
[ -f "$cfg" ] || { echo "workload: no config $cfg"; exit 2; }
[ -x "$bin" ] || { echo "workload: $bin is missing - make build first"; exit 2; }
command -v expect >/dev/null || { echo "workload: expect is required"; exit 2; }
mkdir -p "$out"
addr=127.0.0.1:6071
counters=http://$addr/debug/counters
printf 'workload-v1 %s %s commit=%s host=%s at=%s\n' "$mode" "$variant" "$(git -C "$root" rev-parse --short HEAD)" "$(uname -sm)" "$(date -u +%FT%TZ)" >> "$out/run.txt"

# fresh_home is an empty HOME holding the workload's config alone.
fresh_home() {
  h=$(mktemp -d)
  mkdir -p "$h/.config/watchpost"
  cp "$cfg" "$h/.config/watchpost/config.toml"
  echo "$h"
}

# drive runs the expect driver under a HOME with the switches on.
drive() {
  HOME=$1 XDG_CONFIG_HOME= XDG_CACHE_HOME= WATCHPOST_DEBUG_TIMING=1 WATCHPOST_DEBUG_PPROF=1 WATCHPOST_DEBUG_PPROF_ADDR=$addr \
    expect "$root/scripts/quality/workload.expect" "$2" "$out" "$3" "$bin" "$counters"
}

case $mode in
cold)
  n=${arg:-5}; i=1
  while [ "$i" -le "$n" ]; do
    h=$(fresh_home)
    drive "$h" cold "$i"
    rm -rf "$h"
    i=$((i + 1))
  done ;;
warm)
  # One HOME kept across the runs (W14): the first open fills the caches -
  # the zones' disk tier above all (D-161) - and every later one is a
  # returning listener's. Run 1 is the cold one; read runs 2 onward.
  n=${arg:-4}; i=1
  h=$(fresh_home)
  while [ "$i" -le "$n" ]; do
    drive "$h" cold "$i"
    i=$((i + 1))
  done
  rm -rf "$h" ;;
session)
  mins=${arg:-10}
  h=$(fresh_home)
  rm -f "$out/pid"
  drive "$h" session "$mins" &
  drv=$!
  while [ ! -s "$out/pid" ] && kill -0 "$drv" 2>/dev/null; do sleep 1; done
  pid=$(cat "$out/pid")
  WATCHPOST_DEBUG_PPROF_ADDR=$addr "$root/scripts/quality/soak.sh" "$pid" 3 "$out/samples.csv" 20 &
  smp=$!
  # The counters keep the newest intervals; the loop ticks a few times a
  # second, so they are read each minute and the reader dedupes by seq.
  while kill -0 "$drv" 2>/dev/null; do
    curl -fs --max-time 10 "$counters" >> "$out/counters.jsonl" 2>/dev/null || true
    sleep 60
  done
  wait "$drv"
  kill "$smp" 2>/dev/null || true
  rm -rf "$h" ;;
*) echo "workload: mode is session or cold"; exit 2 ;;
esac
echo "workload: $mode $variant done - $out"
