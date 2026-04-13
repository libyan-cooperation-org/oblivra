package agent

import "runtime"

// runtime_helpers.go — thin wrappers so transport.go does not import "runtime" directly.
// This keeps the import graph clean and the platform detection in one place.

func runtimeGOOS() string   { return runtime.GOOS }
func runtimeGOARCH() string { return runtime.GOARCH }
