//go:build wasip1

package main

import (
	"unsafe"
)

// Exported host functions (implemented in Sovereign Terminal's WasmSandbox)

//go:wasmimport env oblivra_print
func oblivra_print(ptr uint32, len uint32)

//go:wasmimport env oblivra_has_permission
func oblivra_has_permission(ptr uint32, len uint32) uint32

func main() {
	// The main function executes on module instantiate
	logMsg("-> Example WASM Plugin Initialized Successfully!")

	hasPerm := checkPermission("ssh.read")
	if hasPerm {
		logMsg("-> Confirmed we have ssh.read permission")
	} else {
		logMsg("-> Missing ssh.read permission!")
	}
}

// export on_connect so Sovereign can call it
//
//export on_connect
func on_connect(sessionIDPtr uint32, sessionIDLen uint32) {
	// Let's pretend we can read the string safely.
	// In a real plugin framework, we'd use unsafe to stitch the string back together.
	logMsg("-> [WASM Hook] on_connect fired for real")
}

// Helpers to wrap host functions

func logMsg(msg string) {
	ptr, size := stringToPtr(msg)
	oblivra_print(ptr, size)
}

func checkPermission(perm string) bool {
	ptr, size := stringToPtr(perm)
	res := oblivra_has_permission(ptr, size)
	return res == 1
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	if len(buf) == 0 {
		return 0, 0
	}
	ptr := &buf[0]
	unsafePtr := uint32(uintptr(unsafe.Pointer(ptr)))
	return unsafePtr, uint32(len(buf))
}
