// Package features — Average True Range (ATR) calculation.
package features

import (
	"math"

	"github.com/davidmiguel22573/options-scout/internal/marketdata"
)

// ATRResult holds the ATR value and the expected move derived from it.
type ATRResult struct {
	ATR14         float64 // 14-period ATR
	ExpectedMove1 float64 // single-day expected move (1x ATR)
	ExpectedMove5 float64 // 5-day expected move (sqrt5 * ATR)
}

// CalculateATR computes the 14-period ATR from daily candles.
// Requires at least 15 candles (14 + 1 for first TR).
func CalculateATR(candles []marketdata.Candle) ATRResult {
	if len(candles) < 15 {
		last := 0.0
		if len(candles) > 0 {
			last = candles[len(candles)-1].Close
		}
		return ATRResult{
			ATR14:         0,
			ExpectedMove1: last * 0.01, // fallback: 1% of price
			ExpectedMove5: last * 0.01 * math.Sqrt(5),
		}
	}

	trs := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		c := candles[i]
		prev := candles[i-1].Close
		tr := math.Max(c.High-c.Low,
			math.Max(math.Abs(c.High-prev), math.Abs(c.Low-prev)))
		trs[i-1] = tr
	}

	// Use the last 14 TRs.
	period := 14
	if len(trs) < period {
		period = len(trs)
	}
	sum := 0.0
	for _, tr := range trs[len(trs)-period:] {
		sum += tr
	}
	atr := sum / float64(period)

	return ATRResult{
		ATR14:         atr,
		ExpectedMove1: atr,
		ExpectedMove5: atr * math.Sqrt(5),
	}
}
