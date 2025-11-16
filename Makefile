#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
BINDIR ?= $(GOPATH)/bin
BUILDDIR ?= $(CURDIR)/build
PROJECT_NAME = $(shell git remote get-url origin | xargs basename -s .git)

# Build flags
build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=paw \
	-X github.com/cosmos/cosmos-sdk/version.AppName=pawd \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

ifeq (,$(findstring nostrip,$(BUILD_OPTIONS)))
  ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

###############################################################################
###                                 Help                                    ###
###############################################################################

help:
	@echo "============================================================================"
	@echo "PAW Blockchain - Makefile Targets"
	@echo "============================================================================"
	@echo ""
	@echo "Installation & Setup:"
	@echo "  install               ## Install pawd & pawcli binaries"
	@echo "  install-tools         ## Install development tools"
	@echo "  install-hooks         ## Install Git pre-commit hooks"
	@echo ""
	@echo "Development:"
	@echo "  build                 ## Build pawd & pawcli binaries"
	@echo "  build-linux           ## Build Linux binaries (requires go)"
	@echo "  format                ## Format code with gofmt, goimports, misspell"
	@echo "  format-all            ## Format all code (Go, Python, JS, Proto)"
	@echo "  lint                  ## Run golangci-lint"
	@echo "  proto-all             ## Format, lint, and generate protobuf files"
	@echo "  proto-gen             ## Generate protobuf code"
	@echo ""
	@echo "Testing:"
	@echo "  test                  ## Run all tests with coverage"
	@echo "  test-unit             ## Run unit tests only"
	@echo "  test-integration      ## Run integration tests"
	@echo "  test-coverage         ## Generate coverage report"
	@echo "  test-keeper           ## Run keeper module tests"
	@echo "  test-types            ## Run type definition tests"
	@echo "  benchmark             ## Run benchmark tests"
	@echo "  test-invariants       ## Run invariant tests"
	@echo "  test-properties       ## Run property-based tests"
	@echo "  test-simulation       ## Run simulation tests"
	@echo "  test-cometmock        ## Run E2E tests with CometMock"
	@echo "  test-all-advanced     ## Run all advanced tests"
	@echo ""
	@echo "Docker & Services:"
	@echo "  docker-up             ## Start Docker services (dev)"
	@echo "  docker-down           ## Stop Docker services"
	@echo "  docker-logs           ## View Docker service logs"
	@echo "  monitoring-start      ## Start monitoring stack (Prometheus, Grafana, Jaeger)"
	@echo "  monitoring-stop       ## Stop monitoring stack"
	@echo ""
	@echo "Blockchain Operations:"
	@echo "  init-testnet          ## Initialize local testnet"
	@echo "  start-node            ## Start blockchain node"
	@echo "  reset-testnet         ## Reset testnet data"
	@echo "  localnet-start        ## Start local network with Docker"
	@echo "  localnet-stop         ## Stop local network"
	@echo ""
	@echo "Maintenance:"
	@echo "  clean                 ## Remove build artifacts and caches"
	@echo "  clean-all             ## Deep clean (modules, caches, data)"
	@echo ""

###############################################################################
###                                  Build                                  ###
###############################################################################

all: install lint test

build: go.sum
	@echo "--> Building pawd & pawcli"
	mkdir -p $(BUILDDIR)
	go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/pawd ./cmd/pawd
	go build -mod=readonly $(BUILD_FLAGS) -o $(BUILDDIR)/pawcli ./cmd/pawcli

build-linux: go.sum
	LEDGER_ENABLED=false GOOS=linux GOARCH=amd64 $(MAKE) build

install: go.sum
	@echo "--> Installing pawd & pawcli"
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/pawd
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/pawcli

###############################################################################
###                          Tools & Dependencies                           ###
###############################################################################

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

install-tools:
	@echo "--> Installing development tools"
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Installing misspell..."
	@go install github.com/client9/misspell/cmd/misspell@latest
	@echo "Installing buf..."
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "Installing goreleaser..."
	@go install github.com/goreleaser/goreleaser@latest
	@echo "Installing statik..."
	@go install github.com/rakyll/statik@latest
	@echo "--> Development tools installed"

install-hooks:
	@echo "--> Installing Git hooks"
	@if command -v pre-commit > /dev/null 2>&1; then \
		echo "Installing pre-commit hooks..."; \
		pre-commit install; \
		pre-commit install --hook-type commit-msg; \
		echo "✓ Pre-commit hooks installed"; \
	elif command -v npm > /dev/null 2>&1; then \
		echo "Installing Husky hooks..."; \
		npm install; \
		npx husky install; \
		echo "✓ Husky hooks installed"; \
	else \
		echo "⚠ Neither pre-commit nor npm found."; \
		echo "  Install Python (pip install pre-commit) or Node.js (npm install)"; \
		echo "  Then run: make install-hooks"; \
	fi

install-hooks-all:
	@echo "--> Installing all Git hooks (pre-commit + husky)"
	@bash scripts/install-hooks.sh --method=both

update-hooks:
	@echo "--> Updating Git hooks to latest versions"
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit autoupdate; \
		echo "✓ Pre-commit hooks updated"; \
	fi
	@if [ -f "package.json" ] && command -v npm > /dev/null 2>&1; then \
		npm update; \
		echo "✓ npm dependencies updated"; \
	fi

run-hooks:
	@echo "--> Running all pre-commit hooks on all files"
	@if command -v pre-commit > /dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "⚠ pre-commit not installed. Run: make install-hooks"; \
		exit 1; \
	fi

###############################################################################
###                              Documentation                              ###
###############################################################################

update-swagger-docs: statik
	$(BINDIR)/statik -src=client/docs/swagger-ui -dest=client/docs -f -m
	@if [ -n "$(git status --porcelain)" ]; then \
        echo "\033[91mSwagger docs are out of sync!!!\033[0m";\
        exit 1;\
    else \
        echo "\033[92mSwagger docs are in sync\033[0m";\
    fi

###############################################################################
###                           Tests & Simulation                            ###
###############################################################################

test:
	@echo "--> Running all tests with race detection and coverage"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

test-unit:
	@echo "--> Running unit tests"
	@go test -v -race -short ./x/...

test-integration:
	@echo "--> Running integration tests"
	@go test -v -race ./app/... ./tests/e2e/...

test-coverage:
	@echo "--> Running tests with coverage report"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-keeper:
	@echo "--> Running keeper tests"
	@go test -v -race ./x/dex/keeper/... ./x/compute/keeper/... ./x/oracle/keeper/...

test-types:
	@echo "--> Running types tests"
	@go test -v ./x/dex/types/... ./x/compute/types/... ./x/oracle/types/...

benchmark:
	@echo "--> Running benchmarks"
	@go test -mod=readonly -bench=. -benchmem ./...

test-invariants:
	@echo "--> Running invariant tests"
	@go test -v ./tests/invariants/...

test-properties:
	@echo "--> Running property-based tests"
	@go test -v ./tests/property/...

test-simulation:
	@echo "--> Running simulation tests (this may take a while)"
	@go test -v ./tests/simulation/... -timeout 30m

test-cometmock:
	@echo "--> Running E2E tests with CometMock"
	@USE_COMETMOCK=true go test -v ./tests/e2e/...

test-all-advanced:
	@echo "--> Running all advanced tests"
	@$(MAKE) test-invariants
	@$(MAKE) test-properties
	@$(MAKE) test-simulation
	@$(MAKE) test-cometmock

test-simulation-determinism:
	@echo "--> Testing simulation determinism"
	@go test -v ./tests/simulation/... -run TestAppStateDeterminism -timeout 30m

test-simulation-import-export:
	@echo "--> Testing simulation state import/export"
	@go test -v ./tests/simulation/... -run TestAppSimulationAfterImport -timeout 30m

test-simulation-with-invariants:
	@echo "--> Running simulation with all invariants enabled"
	@go test -v ./tests/simulation/... -run TestSimulationWithInvariants -SimulationAllInvariants=true -timeout 60m

benchmark-cometmock:
	@echo "--> Benchmarking CometMock block production"
	@USE_COMETMOCK=true go test -bench=BenchmarkCometMockBlockProduction ./tests/e2e/... -benchtime=10s

benchmark-invariants:
	@echo "--> Benchmarking invariant checks"
	@go test -bench=BenchmarkBankInvariants ./tests/invariants/... -benchtime=10s

###############################################################################
###                                Linting                                  ###
###############################################################################

lint:
	@echo "--> Running linter"
	@golangci-lint run --timeout=10m

format:
	@echo "--> Formatting code"
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" | xargs gofmt -w -s
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" | xargs misspell -w
	@find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/docs/statik/statik.go" | xargs goimports -w -local github.com/paw-chain/paw

format-all:
	@echo "--> Formatting all code (Go, Python, JS, Proto)"
	@./scripts/format-all.sh

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "--> Generating Protobuf files"
	@./scripts/protocgen.sh

proto-format:
	@echo "--> Formatting Protobuf files"
	@find . -name '*.proto' -not -path "./third_party/*" -exec clang-format -i {} \;

proto-lint:
	@echo "--> Linting Protobuf files"
	@buf lint --error-format=json

proto-check-breaking:
	@echo "--> Checking for breaking changes in Protobuf"
	@buf breaking --against $(HTTPS_GIT)#branch=main

###############################################################################
###                              Local Network                              ###
###############################################################################

localnet-start: build
	@echo "--> Starting local testnet"
	@./scripts/localnet-start.sh

localnet-stop:
	@echo "--> Stopping local testnet"
	@pkill pawd || true

dev:
	@echo "--> Starting development environment with Docker"
	@docker-compose -f docker-compose.dev.yml up --build

dev-down:
	@echo "--> Stopping development environment"
	@docker-compose -f docker-compose.dev.yml down

###############################################################################
###                           Docker Short Targets                          ###
###############################################################################

docker-up: dev
	@echo "✓ Docker development environment started"

docker-down: dev-down
	@echo "✓ Docker development environment stopped"

docker-logs:
	@echo "--> Viewing Docker development logs"
	@docker-compose -f docker-compose.dev.yml logs -f

###############################################################################
###                        Blockchain Node Operations                       ###
###############################################################################

init-testnet: build
	@echo "--> Initializing testnet"
	@mkdir -p ~/.paw/testnet
	@$(BUILDDIR)/pawd init test-1 --home ~/.paw/testnet --chain-id test-chain-1
	@echo "✓ Testnet initialized"

start-node: build
	@echo "--> Starting blockchain node"
	@$(BUILDDIR)/pawd start --home ~/.paw/testnet
	@echo "✓ Node started"

reset-testnet:
	@echo "--> Resetting testnet data"
	@rm -rf ~/.paw/testnet
	@$(MAKE) init-testnet

###############################################################################
###                                 Clean                                   ###
###############################################################################

clean:
	@echo "--> Cleaning build artifacts"
	@rm -rf $(BUILDDIR)
	@rm -rf $(HOME)/.paw
	@rm -f coverage.txt coverage.html *.coverprofile
	@go clean -testcache
	@go clean -cache

clean-all: clean
	@echo "--> Deep cleaning (modules, caches, data)"
	@./scripts/clean.sh

###############################################################################
###                                Release                                  ###
###############################################################################

release:
	@echo "--> Building release binaries with GoReleaser"
	@goreleaser release --snapshot --clean

release-test:
	@echo "--> Testing release configuration"
	@goreleaser check

###############################################################################
###                            Development Setup                            ###
###############################################################################

dev-setup:
	@echo "--> Running development setup script"
	@./scripts/dev-setup.sh

###############################################################################
###                      Monitoring & Observability                         ###
###############################################################################

monitoring-start:
	@echo "--> Starting monitoring stack"
	@docker-compose -f docker-compose.monitoring.yml up -d
	@echo "✓ Monitoring stack started"
	@echo "  Grafana: http://localhost:3000 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Jaeger: http://localhost:16686"
	@echo "  Alertmanager: http://localhost:9093"

monitoring-stop:
	@echo "--> Stopping monitoring stack"
	@docker-compose -f docker-compose.monitoring.yml down

monitoring-restart:
	@echo "--> Restarting monitoring stack"
	@docker-compose -f docker-compose.monitoring.yml restart

monitoring-logs:
	@echo "--> Viewing monitoring stack logs"
	@docker-compose -f docker-compose.monitoring.yml logs -f

explorer-start:
	@echo "--> Starting block explorers"
	@docker network create paw-network 2>/dev/null || true
	@docker-compose -f explorer/docker-compose.yml up -d
	@echo "✓ Block explorers started"
	@echo "  Big Dipper: http://localhost:3001"
	@echo "  Mintscan API: http://localhost:8080"

explorer-stop:
	@echo "--> Stopping block explorers"
	@docker-compose -f explorer/docker-compose.yml down

explorer-logs:
	@echo "--> Viewing explorer logs"
	@docker-compose -f explorer/docker-compose.yml logs -f

metrics:
	@echo "--> Opening monitoring dashboards"
	@echo "Opening Prometheus..."
	@if command -v xdg-open > /dev/null; then \
		xdg-open http://localhost:9090; \
	elif command -v open > /dev/null; then \
		open http://localhost:9090; \
	elif command -v start > /dev/null; then \
		start http://localhost:9090; \
	else \
		echo "  Prometheus: http://localhost:9090"; \
	fi
	@sleep 1
	@echo "Opening Grafana..."
	@if command -v xdg-open > /dev/null; then \
		xdg-open http://localhost:3000; \
	elif command -v open > /dev/null; then \
		open http://localhost:3000; \
	elif command -v start > /dev/null; then \
		start http://localhost:3000; \
	else \
		echo "  Grafana: http://localhost:3000"; \
	fi
	@sleep 1
	@echo "Opening Jaeger..."
	@if command -v xdg-open > /dev/null; then \
		xdg-open http://localhost:16686; \
	elif command -v open > /dev/null; then \
		open http://localhost:16686; \
	elif command -v start > /dev/null; then \
		start http://localhost:16686; \
	else \
		echo "  Jaeger: http://localhost:16686"; \
	fi

monitoring-status:
	@echo "--> Monitoring stack status"
	@docker-compose -f docker-compose.monitoring.yml ps

full-stack-start: monitoring-start explorer-start
	@echo "✓ Full observability stack started"

full-stack-stop: monitoring-stop explorer-stop
	@echo "✓ Full observability stack stopped"

###############################################################################
###                        Load Testing & Performance                       ###
###############################################################################

load-test:
	@echo "--> Running k6 blockchain load test"
	@k6 run tests/load/k6/blockchain-load-test.js

load-test-dex:
	@echo "--> Running k6 DEX load test"
	@k6 run tests/load/k6/dex-swap-test.js

load-test-websocket:
	@echo "--> Running k6 WebSocket load test"
	@k6 run tests/load/k6/websocket-test.js

load-test-locust:
	@echo "--> Running Locust load test (headless mode)"
	@locust -f tests/load/locust/locustfile.py --headless --users 100 --spawn-rate 10 --run-time 10m --host http://localhost:1317

load-test-locust-ui:
	@echo "--> Starting Locust web UI"
	@echo "Open http://localhost:8089 in your browser"
	@locust -f tests/load/locust/locustfile.py

load-test-all:
	@echo "--> Running comprehensive load test suite"
	@chmod +x ./scripts/run-load-test.sh
	@./scripts/run-load-test.sh

benchmark-dex:
	@echo "--> Running DEX benchmarks"
	@go test -bench=. -benchmem ./tests/benchmarks/ -run BenchmarkDEX

benchmark-compute:
	@echo "--> Running Compute benchmarks"
	@go test -bench=. -benchmem ./tests/benchmarks/ -run BenchmarkCompute

benchmark-oracle:
	@echo "--> Running Oracle benchmarks"
	@go test -bench=. -benchmem ./tests/benchmarks/ -run BenchmarkOracle

perf-profile:
	@echo "--> Running benchmarks with profiling"
	@chmod +x ./scripts/benchmark.sh
	@./scripts/benchmark.sh

perf-profile-interactive:
	@echo "--> Running benchmarks with interactive profiling"
	@chmod +x ./scripts/benchmark.sh
	@./scripts/benchmark.sh --interactive

###############################################################################
###                            Security Auditing                            ###
###############################################################################

security-audit:
	@echo "--> Running comprehensive security audit"
	@chmod +x ./scripts/security-audit.sh
	@./scripts/security-audit.sh

security-audit-quick:
	@echo "--> Running quick security checks"
	@echo "Running GoSec..."
	@gosec -conf security/.gosec.yml ./... || true
	@echo "Running govulncheck..."
	@govulncheck ./... || true
	@echo "Running GitLeaks..."
	@gitleaks detect --verbose || true

check-deps:
	@echo "--> Checking dependencies for vulnerabilities"
	@chmod +x ./scripts/check-deps.sh
	@./scripts/check-deps.sh

scan-secrets:
	@echo "--> Scanning for secrets and credentials"
	@gitleaks detect --verbose --report-path=security/gitleaks-report.json

security-all: security-audit check-deps scan-secrets
	@echo "✓ All security checks completed"
	@echo "Reports available in security/ directory"

install-security-tools:
	@echo "--> Installing security scanning tools"
	@echo "Installing gosec..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Installing govulncheck..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "Installing nancy..."
	@go install github.com/sonatype-nexus-community/nancy@latest
	@echo "✓ Security tools installed"
	@echo ""
	@echo "Additional recommended tools (install separately):"
	@echo "  - Trivy: https://aquasecurity.github.io/trivy/"
	@echo "  - GitLeaks: https://github.com/gitleaks/gitleaks"

crypto-check:
	@echo "--> Running custom crypto analysis"
	@go run security/crypto-check.go

security-report:
	@echo "--> Generating security report"
	@mkdir -p security
	@echo "PAW Security Report - $(shell date)" > security/report-latest.txt
	@echo "=====================================" >> security/report-latest.txt
	@echo "" >> security/report-latest.txt
	@echo "GoSec:" >> security/report-latest.txt
	@gosec -conf security/.gosec.yml -fmt json ./... >> security/gosec-latest.json 2>&1 || true
	@echo "  Report: security/gosec-latest.json" >> security/report-latest.txt
	@echo "" >> security/report-latest.txt
	@echo "Govulncheck:" >> security/report-latest.txt
	@govulncheck ./... >> security/report-latest.txt 2>&1 || true
	@echo "" >> security/report-latest.txt
	@cat security/report-latest.txt
	@echo "✓ Security report saved to security/report-latest.txt"

.PHONY: help all build build-linux install \
	go.sum install-tools install-hooks install-hooks-all update-hooks run-hooks \
	test test-unit test-integration test-coverage test-keeper test-types benchmark \
	test-invariants test-properties test-simulation test-cometmock test-all-advanced \
	test-simulation-determinism test-simulation-import-export test-simulation-with-invariants \
	benchmark-cometmock benchmark-invariants \
	lint format format-all \
	proto-all proto-gen proto-format proto-lint proto-check-breaking \
	localnet-start localnet-stop dev dev-down \
	docker-up docker-down docker-logs \
	init-testnet start-node reset-testnet \
	clean clean-all \
	release release-test dev-setup \
	monitoring-start monitoring-stop monitoring-restart monitoring-logs monitoring-status \
	explorer-start explorer-stop explorer-logs \
	metrics full-stack-start full-stack-stop \
	load-test load-test-dex load-test-websocket load-test-locust load-test-locust-ui load-test-all \
	benchmark-dex benchmark-compute benchmark-oracle perf-profile perf-profile-interactive \
	security-audit security-audit-quick check-deps scan-secrets security-all \
	install-security-tools crypto-check security-report
