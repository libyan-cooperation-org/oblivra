//go:build windows

package main

import "os"

// Windows has no SIGHUP equivalent we can portably catch — operators
// restart the service to apply config changes.
func setupHUPPlatform() chan os.Signal { return nil }
