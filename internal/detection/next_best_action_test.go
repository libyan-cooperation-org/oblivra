package detection

import "testing"

// Stable, intent-revealing tests for the NBA recommender. The point of
// these isn't to lock the exact confidence numbers (those will tune over
// time) — it's to lock the *direction* of the recommendation per
// canonical scenario. If a future tweak inverts one of these, we want
// to see the test go red and re-confirm intent.

func TestRecommend_CriticalIOCWithKnownHost(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:     "a-1",
		Severity:    "critical",
		Category:    "command-and-control",
		HasIOCMatch: true,
		HostKnown:   true,
	})
	if got.Action != ActionQuarantineHost {
		t.Fatalf("expected quarantine_host for critical+IOC+known host, got %q (conf=%.2f reason=%q)",
			got.Action, got.Confidence, got.Reason)
	}
	if got.Confidence < 0.7 {
		t.Fatalf("expected high confidence for stacked signals, got %.2f", got.Confidence)
	}
}

func TestRecommend_RepeatOffenderBiasesToSuppress(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:          "a-2",
		Severity:         "medium",
		Category:         "discovery",
		IsRepeatOffender: true,
		HostKnown:        true,
	})
	if got.Action != ActionSuppressFP {
		t.Fatalf("expected suppress_as_fp for noisy repeat-offender rule, got %q", got.Action)
	}
}

func TestRecommend_CrownJewelHostBlocksUnilateralQuarantine(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:          "a-3",
		Severity:         "critical",
		Category:         "execution",
		HostKnown:        true,
		IsFromCrownJewel: true,
	})
	// Critical + crown-jewel should bias toward escalation/evidence,
	// not autonomous quarantine of a tier-1 box.
	if got.Action == ActionQuarantineHost {
		t.Fatalf("expected non-quarantine on crown-jewel host, got %q", got.Action)
	}
	if got.Action != ActionEvidenceCapt && got.Action != ActionEscalateT3 {
		t.Fatalf("expected evidence_capture or escalate_tier_3 on crown-jewel host, got %q", got.Action)
	}
}

func TestRecommend_UnknownHostCantBeQuarantined(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:     "a-4",
		Severity:    "critical",
		Category:    "exfiltration",
		HasIOCMatch: true,
		HostKnown:   false, // no agent — can't isolate
	})
	if got.Action == ActionQuarantineHost {
		t.Fatalf("must not recommend quarantine when HostKnown=false, got %q", got.Action)
	}
}

func TestRecommend_InfoSeverityWatchOnly(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:  "a-5",
		Severity: "info",
		Category: "discovery",
	})
	if got.Action != ActionWatchOnly && got.Action != ActionSuppressFP {
		t.Fatalf("expected watch_only / suppress for info-level alerts, got %q", got.Action)
	}
}

func TestRecommend_ServiceAccountEscalates(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:       "a-6",
		Severity:      "high",
		Category:      "credential-access",
		HostKnown:     true,
		UserIsService: true,
	})
	if got.Action == ActionQuarantineHost {
		t.Fatalf("must not auto-quarantine service-account alerts, got %q", got.Action)
	}
}

func TestRecommend_AlternativesAreDistinct(t *testing.T) {
	got := Recommend(NBAFacts{
		AlertID:     "a-7",
		Severity:    "high",
		Category:    "execution",
		HasIOCMatch: true,
		HostKnown:   true,
	})
	for _, a := range got.Alternatives {
		if a == got.Action {
			t.Fatalf("primary action %q duplicated in alternatives %v", got.Action, got.Alternatives)
		}
	}
}

func TestRecommend_DeterministicOrdering(t *testing.T) {
	// Same input, run twice — must produce byte-identical output. The
	// UI relies on this so the highlighted button doesn't flicker
	// across alert-list refreshes.
	f := NBAFacts{
		AlertID:     "a-8",
		Severity:    "high",
		Category:    "execution",
		HasIOCMatch: true,
		HostKnown:   true,
	}
	a := Recommend(f)
	b := Recommend(f)
	if a.Action != b.Action || a.Confidence != b.Confidence {
		t.Fatalf("non-deterministic output: %+v vs %+v", a, b)
	}
}
