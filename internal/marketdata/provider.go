// Package marketdata defines core types and the provider interface for underlying data.
package marketdata

import (
	"time"
)

// Candle represents a single OHLCV bar.
type Candle struct {
	Time   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// Quote is a real-time or delayed underlying quote.
type Quote struct {
	Symbol    string
	Last      float64
	Bid       float64
	Ask       float64
	Volume    float64
	RelVolume float64 // ratio to average volume; 1.0 = average
	Timestamp time.Time
}

// Provider is the interface that wraps all underlying data operations.
// Implementations may be live (Alpaca, Polygon) or manual (file-based).
type Provider interface {
	// Quote returns the latest quote for a symbol.
	Quote(symbol string) (*Quote, error)

	// DailyCandles returns daily OHLCV bars for the given symbol.
	// count is the number of bars requested, most recent last.
	DailyCandles(symbol string, count int) ([]Candle, error)

	// IntradayCandles returns intraday bars at the given resolution.
	// resolution examples: "1m", "5m", "15m", "1h"
	IntradayCandles(symbol string, resolution string, count int) ([]Candle, error)
}
