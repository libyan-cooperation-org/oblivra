//go:build server

package main

import (
	"log"
)

func runGUI() {
	log.Fatal("FATAL: GUI launch requested in headless server build. Use -tags !server to enable GUI.")
}
