# watchpost — build & quality gates (architecture.md §7/§10; C-4: binaries to ./dist)
.PHONY: build test race verify fmt vet lint-imports lint-watermark gate-controls release-matrix clean

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

verify: fmt vet race lint-imports lint-watermark gate-controls
	@echo "verify: ALL GATES GREEN"

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
