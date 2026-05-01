package httpserver

import "crypto/sha256"

func sha256SumInternal(b []byte) [32]byte { return sha256.Sum256(b) }
