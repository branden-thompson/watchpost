#!/bin/zsh
# ARCHIVED SPIKE EVIDENCE — DIR below pointed at the original session scratchpad;
# set DIR to this directory to re-run against the retained binaries/sources.
# usage: measure.sh <mode> <dur> <url> <label>
set -u
DIR=/private/tmp/claude-5932/-Users-bthompso-Desktop-PERSONAL-PROJECTS-watchpost/0102dbfd-9988-47f0-ab80-799ebbcfda44/scratchpad/spike-s1
MODE=$1; DUR=$2; URL=$3; LABEL=$4
LOG=$DIR/$LABEL.log
SAMPLES=$DIR/$LABEL.samples
: > $SAMPLES
$DIR/spike-cgo1 -mode $MODE -dur $DUR -url $URL > $LOG 2>&1 &
PID=$!
echo "started pid=$PID label=$LABEL"
sleep 15
while kill -0 $PID 2>/dev/null; do
  ps -o %cpu=,rss= -p $PID >> $SAMPLES 2>/dev/null
  sleep 5
done
wait $PID
RC=$?
echo "exit=$RC label=$LABEL"
awk '{c+=$1; if($1>cm)cm=$1; r+=$2; if($2>rm)rm=$2; n++}
     END{if(n>0) printf "n=%d cpu_mean=%.2f%% cpu_max=%.2f%% rss_mean=%.1fMB rss_max=%.1fMB\n", n, c/n, cm, r/n/1024, rm/1024}' $SAMPLES
