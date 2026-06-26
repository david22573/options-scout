// Package scoring — human-readable score explanation.
package scoring

import "fmt"

// Explain returns a multi-line breakdown of a Score.
func Explain(s Score) string {
	grade := Classify(s.Total)
	return fmt.Sprintf(
		"Score: %d/100 [%s]\n"+
			"  Liquidity:      %2d/25\n"+
			"  Setup:          %2d/25\n"+
			"  Market Context: %2d/20\n"+
			"  Option Quality: %2d/20\n"+
			"  Risk/Reward:    %2d/10",
		s.Total, grade,
		s.LiquidityScore,
		s.SetupScore,
		s.MarketContextScore,
		s.OptionQualityScore,
		s.RiskRewardScore,
	)
}
