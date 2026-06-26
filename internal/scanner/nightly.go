// Package scanner — nightly scan logic.
package scanner

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

// NightlyReport is the output of the nightly scan.
type NightlyReport struct {
	Date       string
	Candidates []strategies.NightlyCandidate
	Note       string
}

// RunNightly scans the watchlist and returns a NightlyReport.
func RunNightly(
	symbols []string,
	provider marketdata.Provider,
	cfg *config.Config,
	maxCandidates int,
) (*NightlyReport, error) {
	if len(symbols) == 0 {
		return nil, fmt.Errorf("nightly: no symbols in watchlist")
	}

	filtered := FilterCandidates(symbols, provider, cfg, maxCandidates)
	if len(filtered) == 0 {
		return &NightlyReport{Note: "No candidates passed initial filters. All symbols IGNORE."}, nil
	}

	var nightlyCandidates []strategies.NightlyCandidate
	for _, c := range filtered {
		nc := strategies.NightlyCandidate{
			Symbol:       c.Symbol,
			Label:        c.InitialBias,
			Bias:         nightlyBiasNote(c),
			Plan:         nightlyPlan(c),
			Trigger:      "TBD — confirm at market open",
			Invalidation: "TBD — set after watching morning price action",
			SuggestedDTE: "1-7",
			MaxRisk:      fmt.Sprintf("$%.0f-$%.0f", cfg.Account.MaxRiskPerTrade*0.33, cfg.Account.MaxRiskPerTrade),
			Note:         c.Note,
		}
		nightlyCandidates = append(nightlyCandidates, nc)
	}

	return &NightlyReport{
		Candidates: nightlyCandidates,
		Note:       fmt.Sprintf("%d candidate(s) from %d symbols scanned", len(nightlyCandidates), len(symbols)),
	}, nil
}

func nightlyBiasNote(c Candidate) string {
	switch c.InitialBias {
	case strategies.NightlyWatchCall:
		return fmt.Sprintf("Bullish — above 20MA, above VWAP. ATR %.2f (expected move ±%.2f/day)",
			c.ATR.ATR14, c.ATR.ExpectedMove1)
	case strategies.NightlyWatchPut:
		return fmt.Sprintf("Bearish — below 20MA, below VWAP. ATR %.2f (expected move ±%.2f/day)",
			c.ATR.ATR14, c.ATR.ExpectedMove1)
	default:
		return "Neutral — no clear directional bias"
	}
}

func nightlyPlan(c Candidate) string {
	switch c.InitialBias {
	case strategies.NightlyWatchCall:
		return "CALL_DEBIT_SPREAD only after morning confirmation of hold above key level"
	case strategies.NightlyWatchPut:
		return "PUT_DEBIT_SPREAD only after morning breakdown confirmation + failed retest"
	default:
		return "No plan — WAIT for clearer setup"
	}
}
