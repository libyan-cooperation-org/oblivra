package main

import (
	"regexp"
	"strings"
)

// Edge-side DLP: redact sensitive substrings in event messages BEFORE
// shipping. Mirrors the server-side internal/dlp patterns but happens
// at the source, so secrets never leave the host that produced them.
//
// Trade-off: the platform's audit chain still hashes the original
// event content (provenance hash is computed before send), but
// applied-at-edge redaction means even a wire-tapper can't read the
// secret. For compliance regimes where "PII must not leave the host"
// is a requirement (PCI-DSS 3.4, HIPAA 164.312), this is the path.
//
// Toggled per-input via `redact: true` on the input config block, or
// at the top level via `redact: true` to apply to every input.
//
// Costs ~1µs per event with all patterns enabled. Off by default
// because some operators want raw payloads in the audit store; set
// it explicitly when you want masking.

type dlpPattern struct {
	Name string
	RE   *regexp.Regexp
	Mask string
}

var edgePatterns = []dlpPattern{
	{
		Name: "credit-card",
		RE:   regexp.MustCompile(`\b(?:\d[ -]?){13,19}\b`),
		Mask: "[REDACTED:CC]",
	},
	{
		Name: "aws-access-key",
		RE:   regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
		Mask: "[REDACTED:AWS-KEY]",
	},
	{
		Name: "aws-secret",
		RE:   regexp.MustCompile(`\b(?i:aws_secret(?:_access)?_key)\s*[:=]\s*[A-Za-z0-9/+=]{40}\b`),
		Mask: "$0=[REDACTED]",
	},
	{
		Name: "github-pat",
		RE:   regexp.MustCompile(`\bghp_[A-Za-z0-9]{30,}\b`),
		Mask: "[REDACTED:GH-PAT]",
	},
	{
		Name: "jwt",
		RE:   regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`),
		Mask: "[REDACTED:JWT]",
	},
	{
		Name: "password-kv",
		RE:   regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
		Mask: "$1=[REDACTED]",
	},
	{
		Name: "bearer-token",
		RE:   regexp.MustCompile(`(?i)Authorization\s*:\s*Bearer\s+\S+`),
		Mask: "Authorization: Bearer [REDACTED]",
	},
	{
		Name: "ssn-us",
		RE:   regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Mask: "[REDACTED:SSN]",
	},
	{
		Name: "private-key",
		RE:   regexp.MustCompile(`-----BEGIN[A-Z ]+PRIVATE KEY-----[\s\S]*?-----END[A-Z ]+PRIVATE KEY-----`),
		Mask: "[REDACTED:PRIVATE-KEY-BLOCK]",
	},
}

// redactLine masks every pattern in `s`. Returns the redacted string
// plus the list of pattern names that fired (for the redacted
// counter / per-event provenance).
func redactLine(s string) (string, []string) {
	var hits []string
	for _, p := range edgePatterns {
		if p.RE.MatchString(s) {
			s = p.RE.ReplaceAllString(s, p.Mask)
			hits = append(hits, p.Name)
		}
	}
	return s, hits
}

// stillHasSecrets is the post-mask sanity check the agent runs to
// detect leaks the patterns missed. If any of these fragments
// survives, we replace the whole event message with a fully-redacted
// fingerprint string. Conservative — favours data loss over leak.
var leakCanaries = []string{
	"BEGIN RSA PRIVATE KEY",
	"BEGIN OPENSSH PRIVATE KEY",
	"BEGIN PGP PRIVATE KEY",
}

func stillHasSecrets(s string) bool {
	for _, c := range leakCanaries {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}
