#!/bin/sh
# soak.sh — the quality pass's soak sampler (plan §2.1, Q0 task 6; R2-1, RT-10).
#
# Samples one running watchpost every INTERVAL seconds for HOURS hours and
# appends a CSV row per sample; every hour it also asks the process for a
# diagnostic dump (profiles + counters.json under the cache dir). The CSV
# is what `tools/slope` reads (-col heap_alloc) and what the baseline
# document diffs.
#
# usage: scripts/quality/soak.sh <pid> <hours> <out.csv> [interval_s=300]
#
# The process should be launched with WATCHPOST_DEBUG_PPROF=1 so the
# post-GC heap and the gauges can be read from 127.0.0.1:6060/debug/counters;
# without it the row still carries the OS view (RSS, footprint, threads)
# and the heap columns are empty. Only `ps -o` columns and our own JSON are
# ever read — never a raw process listing (red-team RT-15).
#
# Works on macOS (vmmap footprint) and Linux (/proc smaps_rollup Pss).

set -eu

pid=${1:?pid}; hours=${2:?hours}; out=${3:?out.csv}; iv=${4:-300}
n=$(( hours * 3600 / iv ))
per_hour=$(( 3600 / iv )); [ "$per_hour" -lt 1 ] && per_hour=1
addr=${WATCHPOST_DEBUG_PPROF_ADDR:-127.0.0.1:6060}   # the instance's debug address (WATCHPOST_DEBUG_PPROF_ADDR at launch)
counters=http://$addr/debug/counters
dump=http://$addr/debug/dump

if [ ! -s "$out" ]; then
  echo "utc,elapsed,rss_kb,footprint_kb,threads,pcpu,heap_alloc,heap_inuse,heap_objects,goroutines,fds,disk_files,disk_bytes,publishes_recent" >> "$out"
fi

# json_num NAME JSON — first "NAME":<number> in a compact JSON document.
json_num() { printf '%s' "$2" | grep -o "\"$1\":[0-9.eE+-]*" | head -1 | cut -d: -f2; }
# gauge NAME FIELD JSON — len/bytes of one gauge object.
gauge() { printf '%s' "$3" | grep -o "{\"name\":\"$1\",\"len\":[0-9]*,\"bytes\":[0-9]*}" | grep -o "\"$2\":[0-9]*" | cut -d: -f2; }

footprint_kb() {
  case "$(uname -s)" in
    Darwin)
      vmmap --summary "$pid" 2>/dev/null | awk '/^Physical footprint:/ {v=$3; u=substr(v,length(v)); x=substr(v,1,length(v)-1);
        if (u=="K") print int(x); else if (u=="M") print int(x*1024); else if (u=="G") print int(x*1048576); else print int(v/1024)}' ;;
    Linux)
      awk '/^Pss:/ {print $2}' "/proc/$pid/smaps_rollup" 2>/dev/null ;;
  esac
}
threads() {
  case "$(uname -s)" in
    Darwin) ps -M -p "$pid" | tail -n +2 | wc -l | tr -d ' ' ;;
    Linux)  awk '/^Threads:/ {print $2}' "/proc/$pid/status" ;;
  esac
}

i=0
while [ "$i" -lt "$n" ] && kill -0 "$pid" 2>/dev/null; do
  rss=$(ps -o rss= -p "$pid" | tr -d ' '); cpu=$(ps -o pcpu= -p "$pid" | tr -d ' '); el=$(ps -o etime= -p "$pid" | tr -d ' ')
  fp=$(footprint_kb); th=$(threads)
  j=$(curl -fs --max-time 5 "$counters" 2>/dev/null || true)
  ha=; hi=; ho=; go=; fd=; df=; db=; pr=
  if [ -n "$j" ]; then
    ha=$(json_num heap_alloc "$j"); hi=$(json_num heap_inuse "$j"); ho=$(json_num heap_objects "$j")
    go=$(json_num goroutines "$j"); fd=$(json_num fds "$j")
    df=$(gauge disk.cache len "$j"); db=$(gauge disk.cache bytes "$j")
    pr=$(printf '%s' "$j" | grep -o '"recent":{"publishes":[0-9]*' | grep -o '[0-9]*$')
  fi
  printf '%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n' "$(date -u +%FT%TZ)" "$el" "$rss" "$fp" "$th" "$cpu" "$ha" "$hi" "$ho" "$go" "$fd" "$df" "$db" "$pr" >> "$out"
  if [ $(( i % per_hour )) -eq 0 ]; then
    curl -fs --max-time 30 "$dump" >/dev/null 2>&1 || kill -USR1 "$pid" 2>/dev/null || true
  fi
  i=$(( i + 1 ))
  sleep "$iv"
done
