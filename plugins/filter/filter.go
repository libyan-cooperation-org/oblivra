//go:build ignore
package main

// This is a TinyGo plugin example.
// To compile: tinygo build -o filter.wasm -target=wasi filter.go

//export host_log
func host_log(ptr, size uint32)

//export drop_event
func drop_event()

// main is required for WASI but can be empty.
func main() {}

//export process
func process(ptr, size uint32) {
	_ = ptr
	_ = size
	// In a real plugin, we would read the event from the shared memory.
	// For this demo, let's just log that we are processing.
	// We'll keep it simple for now.
	drop_event()
}
