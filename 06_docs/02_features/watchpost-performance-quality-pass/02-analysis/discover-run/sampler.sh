#!/bin/sh
# ps/vmmap every 5 min; pprof heap/goroutine/threadcreate snapshots every 30 min.
out=$1; dir=$2; total=$3; start=$(date +%s); n=0
[ -s "$out" ] || printf 't_s rss_kb footprint_mb threads pcpu goroutines\n' > "$out"
sleep 20
while [ $(( $(date +%s) - start )) -lt "$total" ]; do
  pid=$(pgrep -n -x watchpost); [ -n "$pid" ] || { sleep 30; continue; }
  el=$(( $(date +%s) - start ))
  rss=$(ps -o rss= -p "$pid" | tr -d ' '); cpu=$(ps -o pcpu= -p "$pid" | tr -d ' ')
  thr=$(ps -M -p "$pid" | tail -n +2 | wc -l | tr -d ' ')
  fp=$(vmmap --summary "$pid" 2>/dev/null | awk '/^Physical footprint:/ {print $3}')
  gr=$(curl -s -m 5 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=1' | head -1 | awk '{print $4}')
  printf '%s %s %s %s %s %s\n' "$el" "$rss" "$fp" "$thr" "$cpu" "$gr" >> "$out"
  if [ $(( n % 6 )) -eq 0 ]; then
    for p in heap goroutine threadcreate allocs; do curl -s -m 20 "http://127.0.0.1:6060/debug/pprof/$p" -o "$dir/$p-$el.pb.gz"; done
    curl -s -m 10 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2' -o "$dir/goroutines-$el.txt"
  fi
  n=$((n+1)); sleep 300
done
