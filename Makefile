# watchpost — build & quality gates (architecture.md §7/§10; C-4: binaries to ./dist)
.PHONY: build test race verify fmt vet tidy vuln lint-imports lint-watermark gate-controls release-matrix clean alloc-budget quality-bench p10

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

verify: fmt vet tidy vuln race lint-imports lint-watermark gate-controls
	@echo "verify: ALL GATES GREEN"

# Deterministic allocation pins (quality pass §1). They count mallocs, which the race
# detector distorts, so they run in their own non-race step (red-team R2-8); under
# `make race` they skip themselves via the raceEnabled build tag.
alloc-budget:
	go test -count=1 -run 'AllocBudget$$' ./...

# Wall-clock benchmarks: recorded, never gated (quality pass §0.1). Local, HUM LEAD.
# Needs benchstat: go install golang.org/x/perf/cmd/benchstat@latest
quality-bench:
	go test ./modes/tty ./platform/snapshot ./domains/fire/hms ./platform/render -run '^$$' -bench . -benchmem -count 10 | tee $(DIST)/bench.txt

# P10 safety-critical check (quality pass §1, red-team R2-2). The li-A2DH CLI and the
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
	@cd $(DIST) && (command -v sha256sum >/dev/null && sha256sum $(BINARY)-* || shasum -a 256 $(BINARY)-*) > checksums.txt && echo "release-matrix: OK ($(VERSION))"

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
