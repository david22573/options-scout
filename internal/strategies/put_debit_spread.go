// Package strategies — put debit spread selector.
package strategies

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
	"github.com/davidmiguel22573/options-scout/internal/risk"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
)

// PutDebitSpread finds the best put debit spread from a chain for
// a bearish setup. Returns a Recommendation with Decision=PUT_DEBIT_SPREAD
// or Decision=WAIT.
func PutDebitSpread(
	chain *optionsdata.Chain,
	targetDTE int,
	score scoring.Score,
	cfg *config.Config,
) Recommendation {
	puts := chain.PutsForExpiry(targetDTE)
	if len(puts) < 2 {
		return waitRec(chain.Symbol, "insufficient put contracts for target DTE", score)
	}

	underlying := chain.UnderlyingLast

	// Long leg: near-ATM put (highest strike at or just below spot).
	var longLeg *optionsdata.Contract
	for i := range puts {
		c := &puts[i]
		if c.Strike <= underlying*1.01 { // within 1% above spot
			liq := optionsdata.CheckLiquidity(c, cfg.Filters.MaxBidAskSpreadPct,
				cfg.Filters.MinOpenInterest, cfg.Filters.MinVolume)
			if !liq.Pass {
				continue
			}
			if longLeg == nil || c.Strike > longLeg.Strike {
				longLeg = c
			}
		}
	}
	if longLeg == nil {
		return waitRec(chain.Symbol, "no liquid near-ATM put found", score)
	}

	// Short leg: first liquid put 1-5% below long leg.
	var shortLeg *optionsdata.Contract
	for i := range puts {
		c := &puts[i]
		if c.Strike < longLeg.Strike && c.Strike >= longLeg.Strike*0.95 {
			liq := optionsdata.CheckLiquidity(c, cfg.Filters.MaxBidAskSpreadPct,
				cfg.Filters.MinOpenInterest, cfg.Filters.MinVolume)
			if !liq.Pass {
				continue
			}
			if shortLeg == nil || c.Strike > shortLeg.Strike {
				shortLeg = c
			}
		}
	}
	if shortLeg == nil {
		return waitRec(chain.Symbol, "no liquid OTM put for short leg within 5% of long", score)
	}

	debit := longLeg.Mid - shortLeg.Mid
	if debit <= 0 {
		return waitRec(chain.Symbol, fmt.Sprintf("debit %.2f is negative — chain inversion", debit), score)
	}

	sm, err := risk.CalcDebitSpread(longLeg.Strike, shortLeg.Strike, debit, false)
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
		Decision:      DecisionPutDebitSpread,
		Setup:         "Put debit spread — bearish momentum",
		Underlying:    underlying,
		Expiration:    longLeg.Expiration,
		LongStrike:    longLeg.Strike,
		ShortStrike:   shortLeg.Strike,
		Debit:         debit,
		MaxRisk:       sm.MaxRisk,
		MaxProfit:     sm.MaxProfit,
		Breakeven:     sm.Breakeven,
		EntryTrigger:  fmt.Sprintf("break below %.2f with volume", longLeg.Strike),
		Invalidation:  fmt.Sprintf("reclaim above %.2f", underlying*1.01),
		TakeProfitPct: [2]float64{0.30, 0.70},
		StopLossPct:   [2]float64{0.25, 0.40},
		Confidence:    grade,
		ScoreTotal:    score.Total,
		ScoreDetail:   scoring.Explain(score),
	}
}
