#!/bin/bash
set -e

# Run the Go AST architecture boundary test
echo "Running Architectural Boundary Checks..."
go test ./internal/architecture/... -v

echo "Architecture checks passed successfully."
