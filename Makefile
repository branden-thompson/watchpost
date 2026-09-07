# watchpost — build & quality gates (architecture.md §7/§10; C-4: binaries to ./dist)
.PHONY: cache-clean build test race verify fmt vet tidy vuln lint-imports lint-watermark gate-controls mutant-check release-matrix clean alloc-budget quality-bench p10 hygiene test-platforms

BINARY := watchpost
DIST   := dist
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64

# VERSION is stamped into the binary (cmd/watchpost main.version): the tag on a
# tagged commit, else the nearest tag + commit (and -dirty). Override: make VERSION=0.9.0
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	@mkdir -p $(DIST)
	go build -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY) ./cmd/watchpost

test:
	go test ./...

race:
	go test -race -count=1 ./...

fmt:
	@test -z "$$(gofmt -l . | grep -v '^06_docs/')" || (gofmt -l . | grep -v '^06_docs/'; echo 'gofmt: files need formatting'; exit 1)

vet:
	go vet ./...

# THE BUILD-TAGGED SOURCE IS SOURCE, and nothing was compiling it. The injector
# lives behind `watchpost_debug` (P10-08) so it cannot ship, which also means
# `go vet ./...` never sees it: app/inject_seam_test.go — the test the whole
# injector stands on — stopped compiling at T3.10b and stayed dark until the
# BUILD-exit red team found it by hand (I-3). A tag with no gate is a tag that
# rots. `mutants` is excluded deliberately: mutant-check owns it, and it costs
# ~140s.
vet-tags:
	go vet -tags watchpost_debug ./...

# AND RUN THEM. vet-tags proves the tagged tree COMPILES; it does not run a
# single assertion in it. app/inject_seam_test.go — "the test the whole injector
# stands on" — is behind the tag, so until now no gate in this repository had
# ever executed it. It stayed dark once already, through a compile break that
# vet alone would not have caught either, and was found by hand at a red team.
#
# The injector is the one capability that must never ship, and B3 makes its
# surface user-facing. Asserting things about code no gate runs is how a
# capability gets a green check and no measurement.
#
# ./app ONLY: it is the sole package with tagged tests, and the whole tree under
# the tag costs a second full suite for nothing.
test-tags:
	go test -tags watchpost_debug -count=1 ./app

# Dependency hygiene (quality pass Q0, red-team PH-1/IS-9): go.mod must be tidy,
# the module cache must match go.sum, and no known vulnerability may be reachable.
tidy:
	go mod tidy -diff
	go mod verify

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Import-direction gate: modes/* may import platform/* but NEVER domains/* (architecture §1).
lint-imports:
	@./scripts/lint-imports.sh

# Human-Accountability Attribution gate: no AI watermarks in tracked files or commit messages.
lint-watermark:
	@./scripts/lint-watermark.sh

# Positive controls: prove the custom gates still fire on known-bad input (calibration:
# "Guard Tests Require Positive Controls"). Runs the linters against embedded bad fixtures.
gate-controls:
	@./scripts/lint-imports.sh --self-test
	@./scripts/lint-watermark.sh --self-test
	@./scripts/sync-go-studs.sh --self-test
	@./scripts/quality/p10-unmatched_test.sh

# The mutation corpus and the harness that reads it (06_docs/mutants, Go, behind
# the `mutants` build tag so its ~140s does not land in `go test ./...` and thus
# in `make race`). Every mutant still applies to the tip, still compiles with its
# tests, and is not inert; the harness still reports each verdict with the right
# exit code, still refuses a red baseline and a dirty tree, and still calls a
# crash CAUGHT. All of that was shell until 2026-09-03, and 17 of one session's
# 34 defects were in that shell.
mutant-check:
	@mkdir -p $(DIST)
# -v ON PURPOSE. Without it the durable record is one line — "ok ... 241s" — which
# proves the gate ran and nothing else, and a verify step that accepts that is a
# verify step in name only. With it the log names every property of the mutation
# harness that held, which IS this gate's result. (The per-mutant CAUGHT/SURVIVED
# verdicts come from run.sh, invoked one mutant at a time, and are recorded by
# hand in the batch build logs; this gate guards the harness, not the corpus.)
# THIS GATE'S LOG IS TRANSIENT, AND SAYING SO IS THE POINT. It is printed to
# stdout in the same breath it is written, and it is promoted into the feature's
# 07-readiness folder deliberately at release time — so it is evidence for one
# run, not a kept record. It calls cache-clean, NOT hygiene: hygiene certifies a
# durable record before deleting, and a gate that runs on every verify has no
# durable record to certify. Claiming otherwise here cost a silently skipped
# clean-up inside a 900-line log while verify still reported green (0.14.2), and
# then a 730-line file rewritten on every verify when that was "fixed" wrong.
	@go test -tags mutants -v -count=1 ./06_docs/mutants > $(DIST)/mutant-check.log 2>&1; rc=$$?; \
	  cat $(DIST)/mutant-check.log; \
	  $(MAKE) --no-print-directory cache-clean || exit 1; \
	  exit $$rc

# THE SUITE AS ANOTHER PLATFORM. The first Linux run of 0.14.0 was its release
# PR, and it panicked: app/voices.go branched on runtime.GOOS behind the seam's
# back, so two tests that had pinned darwin walked the Piper install path anyway.
# On a Mac the seam and the real OS agree, so nothing could see it.
#
# This flips the seam for the whole app package (WATCHPOST_TEST_GOOS, honoured by
# a test-only TestMain) and runs the suite as the other platform. It simulates
# the SEAM, not the operating system: real syscalls, paths and audio are still
# the host's, so green here is evidence about platform BRANCHING and never a
# substitute for running on Linux.
# -count=1 IS LOAD-BEARING. Without it `go test` can answer from the cache, and a
# cached "ok" is not a run — the whole point here is to execute the suite in a
# configuration it has not been executed in.
test-platforms:
	@echo "--- app suite as linux ---"   && WATCHPOST_TEST_GOOS=linux  go test -count=1 ./app
	@echo "--- app suite as darwin ---"  && WATCHPOST_TEST_GOOS=darwin go test -count=1 ./app

# --- hygiene: clean up after ourselves, every time, without doing arithmetic ---
#
# ON 2026-09-06 THE VOLUME REACHED 1.2 GiB FREE OF 926 GiB, and 274 GB of that
# was the Go build cache. A mutation run compiles the tree once per mutant, and
# every variant is a cache entry that will never be reused again, so the cache
# grows without bound and nothing trims it. This is not untidiness: the gates
# were flaky, journey steps that need the network failed, and a 14.2 s radio
# read was measured on a machine with no disk left. A FULL DISK IS A VECTOR FOR
# BAD RESULTS, not only for no space.
#
# The protocol is deterministic and consults the size of nothing, because a
# threshold means the cleanup only runs once the damage is already done:
#
#   do the thing -> collect the results -> put them somewhere durable ->
#   VERIFY they are there -> delete the build variants
#
# The verify step is the whole point. Without it this target is `rm` with a
# comment, and the first time a run dies early it would delete the artefacts
# and the evidence together.
#
# WHAT IS KEPT is the stated exception: a small number of real binaries for UAT
# and for comparing versions. dist/watchpost is what `make journey` drives and
# what the HUM LEAD runs for UAT, and the previous release is what the M3 ear
# test and perf-protocol §4 measure against — so the pinned comparator moves up
# with each release, and the one before it goes (its assets stay on the tag). Everything else in dist that is
# executable is a build variant and goes.
#
# RECORDS are not deleted here, but dist is NOT durable: it is the directory
# this target empties. A run's record belongs in the feature's 06_docs tree, on
# a TRACKED path — filed is not the same as committed, and until 0.14.1 eight
# records under 06_docs were ignored by `*.log` and only looked filed.
# `.gitignore` now re-includes `06_docs/**/*.log`; the check below is what
# proves it for any path you pass. hygiene also names any record still sitting
# in dist, so it gets promoted rather than lost on the next run.
HYGIENE_KEEP := watchpost watchpost-0.14.1

hygiene:
	@test -n "$(RESULTS)" || { echo "hygiene: RESULTS must name the run's record; refusing to delete"; exit 1; }
	@test -s "$(RESULTS)" || { echo "hygiene: $(RESULTS) is missing or empty — the run left no record, so NOTHING is deleted"; exit 1; }
	@case "$(RESULTS)" in $(DIST)/*) echo "hygiene: $(RESULTS) is in $(DIST), which this target empties — promote it to the feature's 06_docs tree first"; exit 1;; esac
	@! git check-ignore -q "$(RESULTS)" || { echo "hygiene: $(RESULTS) is git-ignored, so it is filed but not committed — put the record on a tracked path under 06_docs"; exit 1; }
	@echo "hygiene: results durable in $(RESULTS) ($$(wc -l < $(RESULTS) | tr -d ' ') lines, tracked path)"
	@for f in $(DIST)/*; do \
	  b=$$(basename "$$f"); \
	  case " $(HYGIENE_KEEP) " in *" $$b "*) continue;; esac; \
	  if [ -f "$$f" ] && [ -x "$$f" ]; then rm -f "$$f" && echo "hygiene: removed build variant $$b"; fi; \
	done
	@for f in $(DIST)/*; do \
	  b=$$(basename "$$f"); \
	  case " $(HYGIENE_KEEP) " in *" $$b "*) continue;; esac; \
	  if [ -e "$$f" ] && [ ! -x "$$f" ]; then echo "hygiene: RECORD still in $(DIST): $$b — promote it to 06_docs or it dies with the next run"; fi; \
	done
	@$(MAKE) --no-print-directory cache-clean
	@echo "hygiene: kept $(HYGIENE_KEEP)"

# cache-clean is the cleaning half on its own, for callers that have nothing to
# certify. `go clean -cache` is machine-wide, not repo-scoped — that is the
# blast radius and it is deliberate, since the cache it clears is the one this
# repo filled.
cache-clean:
	@go clean -cache -testcache
	@echo "cache-clean: build and test caches cleared"

verify: fmt vet vet-tags test-tags tidy vuln race lint-imports lint-watermark gate-controls mutant-check
	@echo "verify: ALL GATES GREEN"

# Deterministic allocation pins (quality pass §1). They count mallocs, which the race
# detector distorts, so they run in their own non-race step (red-team R2-8); under
# `make race` they skip themselves via the raceEnabled build tag.
# pty-severe machine-verifies the Severe Weather / Disaster Events window on a real pty (0.13.0).
pty-severe: build
	expect scripts/quality/severe-modal.expect

# The VALIDATE core journey on the real binary, real feeds, a FRESH HOME (a
# first run). Its exit code is the count of FAILs, and the M2 step measures the
# keypresses from the dashboard to "role X speaks in voice Y".
#
# IT HAD NO TARGET UNTIL 2026-09-06, and gates.md documented it as a command you
# type. So it went red at the Setup -> Settings rename — it waited for a label
# that no longer existed — and stayed red, unnoticed, exactly as the severe-window
# pty smoke had (F-D3). A gate nothing runs is not a gate.
#
# Not in `verify`: it needs the network and a few minutes. Local, before SHIP.
journey: build
	@d=$$(mktemp -d) && HOME=$$d expect scripts/quality/validate-journey.expect dist/journey.log; \
		rc=$$?; rm -rf $$d; \
		echo "--- dist/journey.log ---"; grep -E "FAIL|M2:" dist/journey.log || true; \
		test $$rc -eq 0 || { echo "journey: $$rc step(s) FAILED"; exit 1; }; \
		echo "journey: every step PASSED"

alloc-budget:
	go test -count=1 -run 'AllocBudget$$' ./...

# Wall-clock benchmarks: recorded, never gated (quality pass §0.1). Local, HUM LEAD.
# Needs benchstat: go install golang.org/x/perf/cmd/benchstat@latest
quality-bench:
	go test ./modes/tty ./platform/snapshot ./domains/fire/hms ./platform/render -run '^$$' -bench . -benchmem -count 10 | tee $(DIST)/bench.txt

# P10 safety-critical check (quality pass §1, red-team R2-2). The harness CLI and the
# exemptions ledger live outside the public tree, so this is a LOCAL gate that must fail
# loud, never skip, when the CLI is absent. A2DH=/path/to/a2dh overrides the lookup.
A2DH ?= a2dh
P10_OUT ?= $(DIST)/p10.json
p10:
	@command -v $(A2DH) >/dev/null 2>&1 || { echo "p10: '$(A2DH)' not found — this gate cannot be skipped; set A2DH=/path/to/a2dh"; exit 1; }
	@mkdir -p $(DIST)
	@$(A2DH) p10 check --json > $(P10_OUT) || { echo "p10: live findings — see $(P10_OUT)"; exit 1; }
	@./scripts/quality/p10-unmatched.sh $(P10_OUT)
	@echo "p10: 0 live, 0 unmatched ($(P10_OUT))"

# T-M (§10.12): cross-compile matrix — every milestone proves it stays green.
release-matrix:
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; ext=""; [ $$os = windows ] && ext=".exe"; \
	  echo "  building $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags '$(LDFLAGS)' -o $(DIST)/$(BINARY)-$$os-$$arch$$ext ./cmd/watchpost || exit 1; \
	done
	@cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum $(BINARY)-* || shasum -a 256 $(BINARY)-*) > checksums.txt
# NFR-2, AND IT RUNS HERE RATHER THAN IN verify FOR A REASON. verify runs before
# the published artifacts exist, so a check living there inspects a binary nobody
# ships. These are the files the release workflow uploads.
	@./scripts/lint-injector.sh $(DIST)/$(BINARY)-*
	@echo "release-matrix: OK ($(VERSION))"

# Installer smoke test: serve the release matrix locally and run scripts/install.sh
# against it (no GitHub involved), then check the installed binary reports the version.
install-test: release-matrix
	@./scripts/install-test.sh

clean:
	rm -rf $(DIST)

# Regenerate the checked-in JSON Schema (TestPublishedSchemaMatchesGenerator keeps it honest).
schema:
	go run ./cmd/watchpost schema > pkg/schema/watchpost-report.v1.0.0-rc.schema.json
.PHONY: schema
