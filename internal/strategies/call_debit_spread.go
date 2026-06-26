// Package strategies — call debit spread selector.
package strategies

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
	"github.com/davidmiguel22573/options-scout/internal/risk"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
)

// CallDebitSpread finds the best call debit spread from a chain for
// a bullish setup. Returns a Recommendation with Decision=CALL_DEBIT_SPREAD
// or Decision=WAIT with a reason.
func CallDebitSpread(
	chain *optionsdata.Chain,
	targetDTE int,
	score scoring.Score,
	cfg *config.Config,
) Recommendation {
	calls := chain.CallsForExpiry(targetDTE)
	if len(calls) < 2 {
		return waitRec(chain.Symbol, "insufficient call contracts for target DTE", score)
	}

	underlying := chain.UnderlyingLast

	// Find the near-ATM long leg (lowest strike above underlying or ATM).
	var longLeg *optionsdata.Contract
	for i := range calls {
		c := &calls[i]
		if c.Strike >= underlying*0.99 { // within 1% below spot
			liq := optionsdata.CheckLiquidity(c, cfg.Filters.MaxBidAskSpreadPct,
				cfg.Filters.MinOpenInterest, cfg.Filters.MinVolume)
			if !liq.Pass {
				continue
			}
			if longLeg == nil || c.Strike < longLeg.Strike {
				longLeg = c
			}
		}
	}
	if longLeg == nil {
		return waitRec(chain.Symbol, "no liquid near-ATM call found", score)
	}

	// Find the short leg: first liquid call 1-5% above long leg.
	var shortLeg *optionsdata.Contract
	for i := range calls {
		c := &calls[i]
		if c.Strike > longLeg.Strike && c.Strike <= longLeg.Strike*1.05 {
			liq := optionsdata.CheckLiquidity(c, cfg.Filters.MaxBidAskSpreadPct,
				cfg.Filters.MinOpenInterest, cfg.Filters.MinVolume)
			if !liq.Pass {
				continue
			}
			if shortLeg == nil || c.Strike < shortLeg.Strike {
				shortLeg = c
			}
		}
	}
	if shortLeg == nil {
		return waitRec(chain.Symbol, "no liquid OTM call for short leg within 5% of long", score)
	}

	debit := longLeg.Mid - shortLeg.Mid
	if debit <= 0 {
		return waitRec(chain.Symbol, fmt.Sprintf("debit %.2f is negative — chain inversion", debit), score)
	}

	sm, err := risk.CalcDebitSpread(longLeg.Strike, shortLeg.Strike, debit, true)
	if err != nil {
		return waitRec(chain.Symbol, err.Error(), score)
	}

	if sm.MaxRisk > cfg.Account.MaxRiskPerTrade {
		return waitRec(chain.Symbol,
			fmt.Sprintf("max risk $%.0f exceeds configured limit $%.0f", sm.MaxRisk, cfg.Account.MaxRiskPerTrade), score)
	}

	grade := scoring.Classify(score.Total)
	return Recommendation{
		Symbol:        chain.Symbol,
		Decision:      DecisionCallDebitSpread,
		Setup:         "Call debit spread — bullish momentum",
		Underlying:    underlying,
		Expiration:    longLeg.Expiration,
		LongStrike:    longLeg.Strike,
		ShortStrike:   shortLeg.Strike,
		Debit:         debit,
		MaxRisk:       sm.MaxRisk,
		MaxProfit:     sm.MaxProfit,
		Breakeven:     sm.Breakeven,
		EntryTrigger:  fmt.Sprintf("break above %.2f with volume", longLeg.Strike),
		Invalidation:  fmt.Sprintf("close below %.2f", underlying*0.99),
		TakeProfitPct: [2]float64{0.30, 0.70},
		StopLossPct:   [2]float64{0.25, 0.40},
		Confidence:    grade,
		ScoreTotal:    score.Total,
		ScoreDetail:   scoring.Explain(score),
	}
}

// waitRec returns a WAIT recommendation with a reason.
func waitRec(symbol, reason string, score scoring.Score) Recommendation {
	return Recommendation{
		Symbol:      symbol,
		Decision:    DecisionWait,
		WaitReason:  reason,
		ScoreTotal:  score.Total,
		ScoreDetail: scoring.Explain(score),
	}
}
