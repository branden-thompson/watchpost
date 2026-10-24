#!/usr/bin/env sh
# workload.sh — W14's standard workload v1 (0.18.0 D-154): the one defined
# workload every before-and-after number is taken on. perf-measurement.md says
# why: the missing piece was never a tool, it was that two runs on different
# workloads could not be compared.
#
#   scripts/quality/workload.sh session <default|heavy> <outdir> [phase_minutes=10]
#   scripts/quality/workload.sh cold    <default|heavy> <outdir> [runs=5]
#   scripts/quality/workload.sh warm    <default|heavy> <outdir> [runs=4]
#   scripts/quality/workload.sh g1      <default|heavy|hourly> <outdir> [minutes=2880] [cycle_minutes=60]
#
# Runs ./dist/watchpost (make build first) under a SCRATCH HOME holding
# 06_docs/perf/workload-v1/config-<variant>.toml, so the listener's own config
# and caches are never read or written; a cold run gets an empty HOME each
# time. WATCHPOST_DEBUG_TIMING=1 turns the timing instrument on (M5, M6), and
# WATCHPOST_DEBUG_PPROF=1 the counters it is read from. A session is sampled
# beside the driver by soak.sh every 20 s. Live network: the run records when
# and where it ran, and a comparison is between runs of the same workload.
#
# g1 is 0.19.0's G1 (W10.2): three processes side by side for 48 hours, each
# in its own HOME and on its own debug address. <outdir>/use opens the map and
# its Propagation mode and, each cycle, pans, steps an hour and enters a
# frequency; <outdir>/map opens the map in Forecast mode and pans on the same
# cycle; <outdir>/idle never opens the map. Each is sampled every two minutes,
# its counters read every ten and a post-GC heap profile kept every hour
# (<arm>/heap-<utc>.pb.gz); each measures 5 minutes past the span asked, so its
# last sample lands past it (D-241). `go run ./tools/perfsum -g1 <outdir>`
# reads the differences against G1's targets. The hourly variant is G1's
# hourly-setting run (A-79, A-83): perfsum judges its downloads alone, at
# 3.3 MB an hour.
set -eu
mode=${1:?session|cold|warm|g1}; variant=${2:?default|heavy|hourly}; out=${3:?outdir}; arg=${4:-}; arg2=${5:-}
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

# drive runs the expect driver under a HOME with the switches on:
# drive <home> <mode> <arg> [debug_addr] [outdir] [arg2].
drive() {
  a=${4:-$addr}; o=${5:-$out}
  HOME=$1 XDG_CONFIG_HOME= XDG_CACHE_HOME= XDG_STATE_HOME= WATCHPOST_DEBUG_TIMING=1 WATCHPOST_DEBUG_PPROF=1 WATCHPOST_DEBUG_PPROF_ADDR=$a \
    expect "$root/scripts/quality/workload.expect" "$2" "$o" "$3" "$bin" "http://$a/debug/counters" "${6:-}"
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
g1)
  mins=${arg:-2880}; cycle=${arg2:-60}
  # Each arm: its name, its debug port; the drivers' and samplers' pids and
  # the HOMEs in the same order.
  arms="use:6072 map:6074 idle:6073"
  drivers=; homes=
  for a in $arms; do
    n=${a%%:*}; p=${a##*:}
    mkdir -p "$out/$n"; rm -f "$out/$n/pid"
    h=$(fresh_home); homes="$homes $h"
    drive "$h" "g1-$n" "$((mins + 5))" "127.0.0.1:$p" "$out/$n" "$cycle" & # past the span: samples come every 2 minutes (D-241)
    drivers="$drivers $!"
  done
  for a in $arms; do
    n=${a%%:*}; i=0
    while [ ! -s "$out/$n/pid" ] && [ "$i" -lt 300 ]; do sleep 1; i=$((i + 1)); done
    [ -s "$out/$n/pid" ] || { echo "workload: the $n arm never started"; exit 1; }
  done
  hours=$(( (mins + 5 + 59) / 60 + 1 ))
  samplers=
  for a in $arms; do
    n=${a%%:*}; p=${a##*:}
    WATCHPOST_DEBUG_PPROF_ADDR="127.0.0.1:$p" "$root/scripts/quality/soak.sh" "$(cat "$out/$n/pid")" "$hours" "$out/$n/samples.csv" 120 &
    samplers="$samplers $!"
  done
  running() { for d in $drivers; do kill -0 "$d" 2>/dev/null && return 0; done; return 1; }
  tick=0
  while running; do
    for a in $arms; do
      n=${a%%:*}; p=${a##*:}
      curl -fs --max-time 10 "http://127.0.0.1:$p/debug/counters" >> "$out/$n/counters.jsonl" 2>/dev/null && echo >> "$out/$n/counters.jsonl" || true
      # Every hour, a post-GC heap profile, kept in the run's folder: the
      # scratch HOME, and the dumps the app keeps there, go at the end (D-242).
      [ $((tick % 6)) -eq 0 ] && curl -fs --max-time 60 "http://127.0.0.1:$p/debug/pprof/heap?gc=1" -o "$out/$n/heap-$(date -u +%Y%m%dT%H%MZ).pb.gz" 2>/dev/null || true
    done
    tick=$((tick + 1))
    sleep 600
  done
  failed=0
  for d in $drivers; do wait "$d" || { echo "workload: an arm's driver failed - see $out/*/phases.log"; failed=1; }; done
  for s in $samplers; do kill "$s" 2>/dev/null || true; done
  wait 2>/dev/null || true
  for h in $homes; do rm -rf "$h"; done
  # A DRIVER THAT FAILED IS A RUN THAT FAILED (RC-1): the run's own exit
  # says so, never only a line in its output.
  [ "$failed" = 0 ] || exit 1 ;;
*) echo "workload: mode is session, cold, warm or g1"; exit 2 ;;
esac
echo "workload: $mode $variant done - $out"
