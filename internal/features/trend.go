// Package features — trend detection.
package features

import "github.com/davidmiguel22573/options-scout/internal/marketdata"

// TrendDirection represents a directional bias.
type TrendDirection string

const (
	TrendBullish TrendDirection = "bullish"
	TrendBearish TrendDirection = "bearish"
	TrendNeutral TrendDirection = "neutral"
)

// TrendResult holds the detected trend.
type TrendResult struct {
	Direction TrendDirection
	Above20MA bool
	Above50MA bool
	Note      string
}

// DetectTrend runs a simple moving average trend check on daily candles.
// Uses 20-day and 50-day SMAs relative to the last close.
func DetectTrend(candles []marketdata.Candle) TrendResult {
	if len(candles) < 20 {
		return TrendResult{TrendNeutral, false, false, "insufficient data"}
	}

	closes := marketdata.Closes(candles)
	last := closes[len(closes)-1]

	sma20 := sma(closes, 20)
	above20 := last > sma20

	var above50 bool
	if len(closes) >= 50 {
		above50 = last > sma(closes, 50)
	}

	var dir TrendDirection
	switch {
	case above20 && above50:
		dir = TrendBullish
	case !above20 && !above50:
		dir = TrendBearish
	default:
		dir = TrendNeutral
	}

	return TrendResult{
		Direction: dir,
		Above20MA: above20,
		Above50MA: above50,
		Note:      "",
	}
}

// sma calculates the simple moving average of the last n values.
func sma(values []float64, n int) float64 {
	if len(values) < n {
		return 0
	}
	slice := values[len(values)-n:]
	sum := 0.0
	for _, v := range slice {
		sum += v
	}
	return sum / float64(n)
}
