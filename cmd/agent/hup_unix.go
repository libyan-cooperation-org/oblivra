//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func setupHUPPlatform() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	return c
}
