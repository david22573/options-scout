// Package scanner — candidate selection for the nightly and morning scans.
package scanner

import (
	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/features"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

// Candidate represents a symbol that passed the initial filter pass.
type Candidate struct {
	Symbol       string
	Quote        *marketdata.Quote
	Trend        features.TrendResult
	ATR          features.ATRResult
	VWAP         features.VWAPResult
	DailyCandles []marketdata.Candle
	InitialBias  strategies.NightlyLabel
	Note         string
}

// FilterCandidates runs symbol-level filters and returns those worth analyzing.
// maxCandidates caps the output list.
func FilterCandidates(
	symbols []string,
	provider marketdata.Provider,
	cfg *config.Config,
	maxCandidates int,
) []Candidate {
	var candidates []Candidate

	for _, sym := range symbols {
		q, err := provider.Quote(sym)
		if err != nil {
			continue // data unavailable — skip
		}

		// Relative volume filter: skip low-activity symbols.
		if q.RelVolume < 0.5 {
			continue
		}

		daily, err := provider.DailyCandles(sym, 55)
		if err != nil || len(daily) < 10 {
			daily = []marketdata.Candle{}
		}

		trend := features.DetectTrend(daily)
		atr := features.CalculateATR(daily)
		vwap := features.CalculateVWAP(daily)

		label := nightlyLabel(trend, vwap)

		candidates = append(candidates, Candidate{
			Symbol:       sym,
			Quote:        q,
			Trend:        trend,
			ATR:          atr,
			VWAP:         vwap,
			DailyCandles: daily,
			InitialBias:  label,
			Note:         "",
		})

		if len(candidates) >= maxCandidates {
			break
		}
	}
	return candidates
}

func nightlyLabel(trend features.TrendResult, vwap features.VWAPResult) strategies.NightlyLabel {
	switch {
	case trend.Direction == features.TrendBullish && vwap.AboveVWAP:
		return strategies.NightlyWatchCall
	case trend.Direction == features.TrendBearish && !vwap.AboveVWAP:
		return strategies.NightlyWatchPut
	case trend.Direction == features.TrendNeutral:
		return strategies.NightlyIgnore
	default:
		return strategies.NightlyIgnore
	}
}
