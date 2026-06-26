// Package scoring — composite score for an options setup.
package scoring

// Score holds the component and total score for a setup.
type Score struct {
	LiquidityScore     int      // 0-25
	SetupScore         int      // 0-25
	MarketContextScore int      // 0-20
	OptionQualityScore int      // 0-20
	RiskRewardScore    int      // 0-10
	Total              int      // 0-100
	Components         []string // human-readable component notes
}

// Calculate sums the component scores and clamps each to its max.
func Calculate(
	liquidity int,
	setup int,
	marketCtx int,
	optionQuality int,
	riskReward int,
) Score {
	liq := clamp(liquidity, 0, 25)
	stp := clamp(setup, 0, 25)
	mkt := clamp(marketCtx, 0, 20)
	opt := clamp(optionQuality, 0, 20)
	rr := clamp(riskReward, 0, 10)

	return Score{
		LiquidityScore:     liq,
		SetupScore:         stp,
		MarketContextScore: mkt,
		OptionQualityScore: opt,
		RiskRewardScore:    rr,
		Total:              liq + stp + mkt + opt + rr,
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
