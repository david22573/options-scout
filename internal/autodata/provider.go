package autodata

import (
	"fmt"
	"strings"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
)

type Provider interface {
	Name() string
	Quote(symbol string) (*marketdata.Quote, error)
	DailyCandles(symbol string, count int) ([]marketdata.Candle, error)
	IntradayCandles(symbol, resolution string, count int) ([]marketdata.Candle, error)
	Chain(symbol string) (*optionsdata.Chain, error)
	MarketClock() (*ClockInfo, error)
}

func NewProvider(cfg *config.Config) (Provider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Data.Provider))
	switch provider {
	case "", "manual":
		return nil, fmt.Errorf("market data provider %q is not supported for auto-data commands", cfg.Data.Provider)
	case "alpaca":
		if strings.TrimSpace(cfg.Data.AlpacaAPIKey) == "" || strings.TrimSpace(cfg.Data.AlpacaAPISecret) == "" {
			return nil, ErrMissingAlpacaCredentials
		}
		return NewAlpacaProvider(cfg.Data), nil
	default:
		return nil, fmt.Errorf("market data provider %q is not supported for auto-data commands", cfg.Data.Provider)
	}
}

var ErrMissingAlpacaCredentials = fmt.Errorf(`Missing Alpaca credentials.
Create a local .env file from .env.example and set:
ALPACA_API_KEY_ID
ALPACA_API_SECRET_KEY`)
