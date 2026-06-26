// Package optionsdata — liquidity quality checks.
package optionsdata

import "fmt"

// LiquidityResult holds the result of a liquidity check.
type LiquidityResult struct {
	Pass   bool
	Reason string
	Score  int // 0-25
}

// CheckLiquidity scores a contract's liquidity against configured thresholds.
func CheckLiquidity(c *Contract, maxSpreadPct float64, minOI, minVol int) LiquidityResult {
	if c.Bid <= 0 {
		return LiquidityResult{false, "bid is zero — no market", 0}
	}
	if c.OpenInterest < minOI {
		return LiquidityResult{false, fmt.Sprintf("open interest %d < minimum %d", c.OpenInterest, minOI), 0}
	}
	if c.Volume < minVol {
		return LiquidityResult{false, fmt.Sprintf("volume %d < minimum %d", c.Volume, minVol), 5}
	}
	spread := c.BidAskSpreadPct()
	if spread > maxSpreadPct {
		return LiquidityResult{false, fmt.Sprintf("bid/ask spread %.1f%% > max %.1f%%", spread*100, maxSpreadPct*100), 5}
	}

	// Score: higher OI and tighter spread = higher score.
	score := 25
	if spread > 0.05 {
		score -= 5
	}
	if c.OpenInterest < 500 {
		score -= 5
	}
	if c.Volume < 50 {
		score -= 3
	}
	if score < 0 {
		score = 0
	}
	return LiquidityResult{true, "liquid", score}
}
