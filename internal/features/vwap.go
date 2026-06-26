// Package features — VWAP calculation.
package features

import "github.com/davidmiguel22573/options-scout/internal/marketdata"

// VWAPResult holds VWAP and the current price relative to it.
type VWAPResult struct {
	VWAP        float64
	LastClose   float64
	AboveVWAP   bool
	PctFromVWAP float64 // positive = above
}

// CalculateVWAP computes cumulative VWAP over intraday candles.
// For daily candles it returns a price-weighted average approximation.
func CalculateVWAP(candles []marketdata.Candle) VWAPResult {
	if len(candles) == 0 {
		return VWAPResult{}
	}

	var cumPV, cumVol float64
	for _, c := range candles {
		typicalPrice := (c.High + c.Low + c.Close) / 3.0
		cumPV += typicalPrice * c.Volume
		cumVol += c.Volume
	}

	vwap := 0.0
	if cumVol > 0 {
		vwap = cumPV / cumVol
	}

	last := candles[len(candles)-1].Close
	pct := 0.0
	if vwap > 0 {
		pct = (last - vwap) / vwap
	}

	return VWAPResult{
		VWAP:        vwap,
		LastClose:   last,
		AboveVWAP:   last >= vwap,
		PctFromVWAP: pct,
	}
}
