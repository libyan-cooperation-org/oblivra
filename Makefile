.PHONY: build build-headless test clean soak-test smoke-test docs

default: build

# Set LICENSE_PUB_KEY env var to inject Ed25519 public key (hex) for commercial builds.
# Leave unset for Community/dev builds — manager defaults to Community tier.
build:
	@echo "Building Wails GUI App..."
	wails build -ldflags "-X github.com/kingknull/oblivrashell/internal/core.licensePubKey=$(LICENSE_PUB_KEY)"

build-headless:
	@echo "Building Headless CLI/Server..."
	go build -o build/bin/oblivrashell-server ./cmd/cli

test:
	@echo "Running unit tests..."
	go test -v ./...

bench-siem:
	@echo "Running SIEM benchmarking..."
	go run ./cmd/bench_siem

soak-test:
	@echo "Running 5,000 EPS 30s soak test on localhost..."
	go run ./cmd/soak_test/main.go --duration 30s

smoke-test:
	@echo "Running API smoke tests..."
	bash ./scripts/api_smoke_test.sh

clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/bin/*
