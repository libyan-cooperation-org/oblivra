.PHONY: build build-headless test clean soak-test smoke-test docs

default: build

build:
	@echo "Building Wails GUI App..."
	wails build

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
