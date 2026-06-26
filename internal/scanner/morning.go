// Package scanner — morning confirmation logic.
package scanner

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/features"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

// MorningInput groups all data needed for a morning confirmation pass.
type MorningInput struct {
	Symbol        string
	Provider      marketdata.Provider
	ChainProvider optionsdata.Provider
	NightlyLabel  strategies.NightlyLabel
	DTE           int
	Cfg           *config.Config
}

// MorningResult is the output for a single symbol.
type MorningResult struct {
	Rec  strategies.Recommendation
	Note string
}

// RunMorning evaluates a single symbol and returns a Recommendation.
func RunMorning(in MorningInput) MorningResult {
	// Fetch intraday candles (1-min or 5-min bars).
	intraday, _ := in.Provider.IntradayCandles(in.Symbol, "5m", 78) // ~6.5h session
	vwap := features.CalculateVWAP(intraday)

	// Opening range from first 6 bars (30 minutes).
	or := features.OpeningRange(intraday, 6)

	// Fetch option chain.
	chain, err := in.ChainProvider.Chain(in.Symbol)
	if err != nil {
		return MorningResult{
			Rec:  waitReasonRec(in.Symbol, fmt.Sprintf("chain unavailable: %v", err)),
			Note: "chain load failed",
		}
	}

	// Build a score.
	liquidScore := scoreLiquidity(chain, in.Cfg)
	setupScore := scoreSetup(vwap, or, in.NightlyLabel)
	mktScore := 10 // V1: fixed market context — improve with SPY comparison later
	optScore := scoreOptionQuality(chain, in.DTE, in.Cfg)
	rrScore := 5 // V1: fixed — improve with actual spread RR calculation

	sc := scoring.Calculate(liquidScore, setupScore, mktScore, optScore, rrScore)

	if !scoring.ShouldTrade(sc, in.Cfg.Scoring.MinScoreTrade) {
		reason := fmt.Sprintf("score %d below threshold %d", sc.Total, in.Cfg.Scoring.MinScoreTrade)
		if !or.BelowLow && !or.AboveHigh {
			reason = "price still inside opening range — no directional confirmation"
		} else if !vwap.AboveVWAP && in.NightlyLabel == strategies.NightlyWatchCall {
			reason = "bullish watchlist but price below VWAP — no confirmation"
		}
		return MorningResult{
			Rec:  waitReasonRec(in.Symbol, reason),
			Note: scoring.Explain(sc),
		}
	}

	// Select strategy based on nightly bias and morning confirmation.
	switch in.NightlyLabel {
	case strategies.NightlyWatchCall:
		if or.AboveHigh && vwap.AboveVWAP {
			rec := strategies.CallDebitSpread(chain, in.DTE, sc, in.Cfg)
			rec.Setup = "Opening range breakout + above VWAP"
			return MorningResult{Rec: rec, Note: scoring.Explain(sc)}
		}
	case strategies.NightlyWatchPut:
		if or.BelowLow && !vwap.AboveVWAP {
			rec := strategies.PutDebitSpread(chain, in.DTE, sc, in.Cfg)
			rec.Setup = "Opening range breakdown + below VWAP"
			return MorningResult{Rec: rec, Note: scoring.Explain(sc)}
		}
	}

	return MorningResult{
		Rec:  waitReasonRec(in.Symbol, "morning trigger not confirmed — directional bias exists but entry not clean"),
		Note: scoring.Explain(sc),
	}
}

func scoreLiquidity(chain *optionsdata.Chain, cfg *config.Config) int {
	liquid := 0
	for _, c := range chain.Contracts {
		r := optionsdata.CheckLiquidity(&c, cfg.Filters.MaxBidAskSpreadPct,
			cfg.Filters.MinOpenInterest, cfg.Filters.MinVolume)
		if r.Pass {
			liquid++
		}
	}
	if liquid >= 20 {
		return 25
	}
	if liquid >= 10 {
		return 18
	}
	if liquid >= 5 {
		return 12
	}
	return 5
}

func scoreSetup(vwap features.VWAPResult, or features.OpeningRangeResult, label strategies.NightlyLabel) int {
	score := 0
	if label != strategies.NightlyIgnore {
		score += 10
	}
	if or.AboveHigh || or.BelowLow {
		score += 10
	}
	if (label == strategies.NightlyWatchCall && vwap.AboveVWAP) ||
		(label == strategies.NightlyWatchPut && !vwap.AboveVWAP) {
		score += 5
	}
	return score
}

func scoreOptionQuality(chain *optionsdata.Chain, dte int, cfg *config.Config) int {
	if dte < cfg.Filters.MinDTEDebit || dte > cfg.Filters.MaxDTEDebit {
		return 5
	}
	return 15
}

func waitReasonRec(symbol, reason string) strategies.Recommendation {
	return strategies.Recommendation{
		Symbol:     symbol,
		Decision:   strategies.DecisionWait,
		WaitReason: reason,
	}
}
