#!/bin/sh
# Regenerates the embedded geodata payloads (JD2: the previously-undocumented
# S2 trim pipeline). Run from the repo root; requires curl, awk, python3.
# Data: GeoNames (CC-BY 4.0) — attribution stays in the About view (OQ-15).
set -eu
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
DEST=domains/locations/geodata/data
echo "downloading GeoNames snapshots..."
curl -fsSL -o "$WORK/cities.zip" https://download.geonames.org/export/dump/cities15000.zip
curl -fsSL -o "$WORK/US.zip"     https://download.geonames.org/export/zip/US.zip
unzip -q "$WORK/cities.zip" -d "$WORK"; unzip -q "$WORK/US.zip" -d "$WORK"
echo "trimming columns (S2 layout: cities name,ascii,admin1,country,lat,lon,pop,tz; zips zip,place,state,lat,lon)..."
awk -F'\t' 'BEGIN{OFS="\t"}{print $2,$3,$11,$9,$5,$6,$15,$18}' "$WORK/cities15000.txt" > "$WORK/cities_trim.tsv"
awk -F'\t' 'BEGIN{OFS="\t"}{print $2,$3,$5,$10,$11}' "$WORK/US.txt" > "$WORK/zips_trim.tsv"
echo "pre-sorting (Load asserts sortedness fail-closed)..."
python3 - "$WORK" <<'PY'
import sys, gzip
w = sys.argv[1]
cities = open(f"{w}/cities_trim.tsv", encoding="utf-8").read().rstrip("\n").split("\n")
cities.sort(key=lambda l: l.split("\t")[1].lower())
zips = open(f"{w}/zips_trim.tsv", encoding="utf-8").read().rstrip("\n").split("\n")
zips.sort(key=lambda l: l.split("\t")[0])
for name, rows in (("cities_trim", cities), ("zips_trim", zips)):
    with gzip.open(f"{w}/{name}.tsv.gz", "wt", encoding="utf-8", compresslevel=9) as f:
        f.write("\n".join(rows) + "\n")
PY
cp "$WORK/cities_trim.tsv.gz" "$WORK/zips_trim.tsv.gz" "$DEST/"
echo "done. NOW REQUIRED: update the SHA-256 pins in"
echo "  domains/locations/geodata/checksums_test.go  (go test ./domains/locations/geodata/ shows the new hashes)"
echo "and note the new snapshot date in the geodata package doc — same commit."
