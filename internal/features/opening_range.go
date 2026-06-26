// Package features — opening range calculation.
package features

import "github.com/davidmiguel22573/options-scout/internal/marketdata"

// OpeningRangeResult holds the first-N-bar high/low and where price is relative.
type OpeningRangeResult struct {
	High        float64
	Low         float64
	Width       float64
	LastClose   float64
	AboveHigh   bool // potential breakout
	BelowLow    bool // potential breakdown
	InsideRange bool
}

// OpeningRange computes the high/low of the first N intraday candles.
// Typically N=4 for first 15m bars (1m chart) or N=1 for first 30m bar.
func OpeningRange(candles []marketdata.Candle, firstNBars int) OpeningRangeResult {
	if len(candles) == 0 || firstNBars <= 0 {
		return OpeningRangeResult{}
	}
	n := firstNBars
	if n > len(candles) {
		n = len(candles)
	}

	high := candles[0].High
	low := candles[0].Low
	for _, c := range candles[1:n] {
		if c.High > high {
			high = c.High
		}
		if c.Low < low {
			low = c.Low
		}
	}

	last := candles[len(candles)-1].Close
	return OpeningRangeResult{
		High:        high,
		Low:         low,
		Width:       high - low,
		LastClose:   last,
		AboveHigh:   last > high,
		BelowLow:    last < low,
		InsideRange: last >= low && last <= high,
	}
}
