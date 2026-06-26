// Package marketdata — candle helpers.
package marketdata

import "fmt"

// Validate checks that a candle slice has at least minCount bars.
func Validate(candles []Candle, minCount int) error {
	if len(candles) < minCount {
		return fmt.Errorf("marketdata: need %d candles, got %d", minCount, len(candles))
	}
	return nil
}

// Closes returns just the closing prices from a candle slice.
func Closes(candles []Candle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.Close
	}
	return out
}

// Highs returns the high prices.
func Highs(candles []Candle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.High
	}
	return out
}

// Lows returns the low prices.
func Lows(candles []Candle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.Low
	}
	return out
}

// Volumes returns the volumes.
func Volumes(candles []Candle) []float64 {
	out := make([]float64, len(candles))
	for i, c := range candles {
		out[i] = c.Volume
	}
	return out
}
