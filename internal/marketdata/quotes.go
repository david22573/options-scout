// Package marketdata — manual (file-based) provider for V1 development.
package marketdata

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ManualQuote is the JSON shape for a manually supplied quote.
type ManualQuote struct {
	Symbol    string  `json:"symbol"`
	Last      float64 `json:"last"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	Volume    float64 `json:"volume"`
	RelVolume float64 `json:"rel_volume"`
}

// ManualProvider reads quotes and candles from JSON files.
// It satisfies the Provider interface for offline/testing use.
type ManualProvider struct {
	quotePath   string
	candlesPath string
}

// NewManualProvider returns a ManualProvider pointed at the given files.
// quotePath and candlesPath may be empty if not needed.
func NewManualProvider(quotePath, candlesPath string) *ManualProvider {
	return &ManualProvider{
		quotePath:   quotePath,
		candlesPath: candlesPath,
	}
}

// Quote loads the quote from quotePath JSON.
func (p *ManualProvider) Quote(symbol string) (*Quote, error) {
	if p.quotePath == "" {
		return nil, fmt.Errorf("manual provider: no quote file configured")
	}
	data, err := os.ReadFile(p.quotePath)
	if err != nil {
		return nil, fmt.Errorf("manual provider: read quote: %w", err)
	}
	var mq ManualQuote
	if err := json.Unmarshal(data, &mq); err != nil {
		return nil, fmt.Errorf("manual provider: parse quote: %w", err)
	}
	return &Quote{
		Symbol:    mq.Symbol,
		Last:      mq.Last,
		Bid:       mq.Bid,
		Ask:       mq.Ask,
		Volume:    mq.Volume,
		RelVolume: mq.RelVolume,
		Timestamp: time.Now(),
	}, nil
}

// DailyCandles returns an empty slice for the manual provider.
// In V1, candles are embedded in the chain JSON.
func (p *ManualProvider) DailyCandles(symbol string, count int) ([]Candle, error) {
	return []Candle{}, nil
}

// IntradayCandles returns an empty slice for the manual provider.
func (p *ManualProvider) IntradayCandles(symbol, resolution string, count int) ([]Candle, error) {
	return []Candle{}, nil
}
