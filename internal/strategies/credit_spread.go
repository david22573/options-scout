// Package strategies — bear call credit spread and bull put credit spread.
package strategies

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
	"github.com/davidmiguel22573/options-scout/internal/risk"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
)

// BearCallCreditSpread finds a bear call credit spread above resistance.
// Use after a hype pump or failed breakout with elevated IV.
func BearCallCreditSpread(
	chain *optionsdata.Chain,
	targetDTE int,
	resistanceLevel float64,
	score scoring.Score,
	cfg *config.Config,
) Recommendation {
	calls := chain.CallsForExpiry(targetDTE)
	underlying := chain.UnderlyingLast

	// Short leg: first liquid call above resistance.
	var shortLeg *optionsdata.Contract
	for i := range calls {
		c := &calls[i]
		if c.Strike > resistanceLevel {
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
		return waitRec(chain.Symbol, fmt.Sprintf("no liquid call above resistance %.2f", resistanceLevel), score)
	}

	// Long leg: first liquid call 1 spread-width above short.
	var longLeg *optionsdata.Contract
	for i := range calls {
		c := &calls[i]
		if c.Strike > shortLeg.Strike {
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
		return waitRec(chain.Symbol, "no liquid call for long leg of bear call spread", score)
	}

	credit := shortLeg.Mid - longLeg.Mid
	if credit <= 0 {
		return waitRec(chain.Symbol, fmt.Sprintf("credit %.2f is not positive", credit), score)
	}

	sm, err := risk.CalcCreditSpread(shortLeg.Strike, longLeg.Strike, credit, true)
	if err != nil {
		return waitRec(chain.Symbol, err.Error(), score)
	}
	if sm.MaxRisk > cfg.Account.MaxRiskPerTrade {
		return waitRec(chain.Symbol,
			fmt.Sprintf("max risk $%.0f exceeds limit $%.0f", sm.MaxRisk, cfg.Account.MaxRiskPerTrade), score)
	}

	grade := scoring.Classify(score.Total)
	return Recommendation{
		Symbol:        chain.Symbol,
		Decision:      DecisionBearCallCredit,
		Setup:         "Bear call credit spread — fade above resistance",
		Underlying:    underlying,
		Expiration:    shortLeg.Expiration,
		LongStrike:    longLeg.Strike,
		ShortStrike:   shortLeg.Strike,
		Debit:         -credit, // negative = received
		MaxRisk:       sm.MaxRisk,
		MaxProfit:     sm.MaxProfit,
		Breakeven:     sm.Breakeven,
		EntryTrigger:  fmt.Sprintf("price fails at/below %.2f after hype pump", resistanceLevel),
		Invalidation:  fmt.Sprintf("close above short strike %.2f", shortLeg.Strike),
		TakeProfitPct: [2]float64{0.30, 0.50},
		StopLossPct:   [2]float64{0.25, 0.40},
		Confidence:    grade,
		ScoreTotal:    score.Total,
		ScoreDetail:   scoring.Explain(score),
	}
}

// BullPutCreditSpread finds a bull put credit spread below support.
func BullPutCreditSpread(
	chain *optionsdata.Chain,
	targetDTE int,
	supportLevel float64,
	score scoring.Score,
	cfg *config.Config,
) Recommendation {
	puts := chain.PutsForExpiry(targetDTE)
	underlying := chain.UnderlyingLast

	// Short leg: highest liquid put below support.
	var shortLeg *optionsdata.Contract
	for i := range puts {
		c := &puts[i]
		if c.Strike < supportLevel {
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
		return waitRec(chain.Symbol, fmt.Sprintf("no liquid put below support %.2f", supportLevel), score)
	}

	// Long leg: first liquid put below short leg.
	var longLeg *optionsdata.Contract
	for i := range puts {
		c := &puts[i]
		if c.Strike < shortLeg.Strike {
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
		return waitRec(chain.Symbol, "no liquid put for long leg of bull put spread", score)
	}

	credit := shortLeg.Mid - longLeg.Mid
	if credit <= 0 {
		return waitRec(chain.Symbol, fmt.Sprintf("credit %.2f is not positive", credit), score)
	}

	sm, err := risk.CalcCreditSpread(shortLeg.Strike, longLeg.Strike, credit, false)
	if err != nil {
		return waitRec(chain.Symbol, err.Error(), score)
	}
	if sm.MaxRisk > cfg.Account.MaxRiskPerTrade {
		return waitRec(chain.Symbol,
			fmt.Sprintf("max risk $%.0f exceeds limit $%.0f", sm.MaxRisk, cfg.Account.MaxRiskPerTrade), score)
	}

	grade := scoring.Classify(score.Total)
	return Recommendation{
		Symbol:        chain.Symbol,
		Decision:      DecisionBullPutCredit,
		Setup:         "Bull put credit spread — support hold",
		Underlying:    underlying,
		Expiration:    shortLeg.Expiration,
		LongStrike:    longLeg.Strike,
		ShortStrike:   shortLeg.Strike,
		Debit:         -credit,
		MaxRisk:       sm.MaxRisk,
		MaxProfit:     sm.MaxProfit,
		Breakeven:     sm.Breakeven,
		EntryTrigger:  fmt.Sprintf("price holds above support %.2f", supportLevel),
		Invalidation:  fmt.Sprintf("close below short strike %.2f", shortLeg.Strike),
		TakeProfitPct: [2]float64{0.30, 0.50},
		StopLossPct:   [2]float64{0.25, 0.40},
		Confidence:    grade,
		ScoreTotal:    score.Total,
		ScoreDetail:   scoring.Explain(score),
	}
}
