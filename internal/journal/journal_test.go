package journal

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

func TestComputeStatsCountsWaitPendingEntries(t *testing.T) {
	t.Parallel()

	entries, err := LoadAll(filepath.Join("testdata", "review_wait_pending.jsonl"))
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	stats := ComputeStats(entries)
	if stats.TotalEntries != 2 {
		t.Fatalf("TotalEntries = %d, want 2", stats.TotalEntries)
	}
	if stats.WaitDecisions != 2 {
		t.Fatalf("WaitDecisions = %d, want 2", stats.WaitDecisions)
	}
	if stats.TradeDecisions != 0 {
		t.Fatalf("TradeDecisions = %d, want 0", stats.TradeDecisions)
	}
	if stats.PendingCount != 2 {
		t.Fatalf("PendingCount = %d, want 2", stats.PendingCount)
	}
}

func TestFromRecommendationAddsWaitMetadataAndSnapshotHash(t *testing.T) {
	t.Parallel()

	snapshotRef := filepath.Join("..", "..", "examples", "chain_QQQ.json")
	wantHash := sha256.Sum256([]byte(snapshotQQQFixture))
	tests := []struct {
		name      string
		score     int
		wantSetup string
		wantBand  string
	}{
		{name: "watch", score: 67, wantSetup: "WATCH_ONLY", wantBand: "WATCH"},
		{name: "wait", score: 54, wantSetup: "WAIT", wantBand: "WAIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := FromRecommendation(strategies.Recommendation{
				Symbol:     "QQQ",
				Decision:   strategies.DecisionWait,
				WaitReason: "chain liquidity score 12/25 - too illiquid for defined-risk spread",
				ScoreTotal: tt.score,
			}, snapshotRef)

			if entry.Mode != "paper" {
				t.Fatalf("Mode = %q, want paper", entry.Mode)
			}
			if entry.Setup != tt.wantSetup {
				t.Fatalf("Setup = %q, want %q", entry.Setup, tt.wantSetup)
			}
			if entry.ScoreBand != tt.wantBand {
				t.Fatalf("ScoreBand = %q, want %q", entry.ScoreBand, tt.wantBand)
			}
			if len(entry.ReasonCodes) == 0 {
				t.Fatal("ReasonCodes is empty, want at least one code")
			}
			if entry.ReasonCodes[0] != "LOW_LIQUIDITY" {
				t.Fatalf("ReasonCodes[0] = %q, want LOW_LIQUIDITY", entry.ReasonCodes[0])
			}
			if entry.SnapshotSHA256 == "" {
				t.Fatal("SnapshotSHA256 is empty")
			}
			if entry.SnapshotSHA256 != fmt.Sprintf("%x", wantHash) {
				t.Fatalf("SnapshotSHA256 = %q, want %x", entry.SnapshotSHA256, wantHash)
			}
		})
	}
}

const snapshotQQQFixture = "{\n  \"symbol\": \"QQQ\",\n  \"underlying_last\": 484.50,\n  \"contracts\": [\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 480.0,\n      \"option_type\": \"call\",\n      \"bid\": 6.80,\n      \"ask\": 7.10,\n      \"volume\": 312,\n      \"open_interest\": 2450,\n      \"iv\": 0.22,\n      \"delta\": 0.55,\n      \"theta\": -0.18\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 485.0,\n      \"option_type\": \"call\",\n      \"bid\": 3.90,\n      \"ask\": 4.10,\n      \"volume\": 528,\n      \"open_interest\": 5100,\n      \"iv\": 0.20,\n      \"delta\": 0.42,\n      \"theta\": -0.20\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 490.0,\n      \"option_type\": \"call\",\n      \"bid\": 1.85,\n      \"ask\": 2.00,\n      \"volume\": 620,\n      \"open_interest\": 7300,\n      \"iv\": 0.19,\n      \"delta\": 0.28,\n      \"theta\": -0.21\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 495.0,\n      \"option_type\": \"call\",\n      \"bid\": 0.62,\n      \"ask\": 0.70,\n      \"volume\": 415,\n      \"open_interest\": 3800,\n      \"iv\": 0.21,\n      \"delta\": 0.14,\n      \"theta\": -0.15\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 480.0,\n      \"option_type\": \"put\",\n      \"bid\": 3.10,\n      \"ask\": 3.30,\n      \"volume\": 290,\n      \"open_interest\": 3100,\n      \"iv\": 0.23,\n      \"delta\": -0.44,\n      \"theta\": -0.19\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 484.0,\n      \"option_type\": \"put\",\n      \"bid\": 4.50,\n      \"ask\": 4.70,\n      \"volume\": 480,\n      \"open_interest\": 4200,\n      \"iv\": 0.22,\n      \"delta\": -0.50,\n      \"theta\": -0.21\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 479.0,\n      \"option_type\": \"put\",\n      \"bid\": 2.80,\n      \"ask\": 3.00,\n      \"volume\": 350,\n      \"open_interest\": 2800,\n      \"iv\": 0.24,\n      \"delta\": -0.38,\n      \"theta\": -0.18\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 475.0,\n      \"option_type\": \"put\",\n      \"bid\": 1.40,\n      \"ask\": 1.55,\n      \"volume\": 510,\n      \"open_interest\": 5500,\n      \"iv\": 0.25,\n      \"delta\": -0.24,\n      \"theta\": -0.16\n    },\n    {\n      \"expiration\": \"2026-07-02\",\n      \"strike\": 470.0,\n      \"option_type\": \"put\",\n      \"bid\": 0.55,\n      \"ask\": 0.65,\n      \"volume\": 380,\n      \"open_interest\": 4100,\n      \"iv\": 0.27,\n      \"delta\": -0.12,\n      \"theta\": -0.11\n    }\n  ]\n}\n"
