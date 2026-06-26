// Package config — watchlist loader.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Watchlist is a named list of ticker symbols.
type Watchlist struct {
	Symbols []string `yaml:"symbols"`
}

// LoadWatchlist reads a YAML watchlist file.
func LoadWatchlist(path string) (*Watchlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("watchlist: read %s: %w", path, err)
	}

	var wl Watchlist
	if err := yaml.Unmarshal(data, &wl); err != nil {
		return nil, fmt.Errorf("watchlist: parse %s: %w", path, err)
	}

	if len(wl.Symbols) == 0 {
		return nil, fmt.Errorf("watchlist: %s has no symbols", path)
	}

	// Normalize to uppercase.
	for i, s := range wl.Symbols {
		wl.Symbols[i] = strings.ToUpper(strings.TrimSpace(s))
	}

	return &wl, nil
}

// DefaultWatchlist returns the liquid core symbols used for paper trading.
func DefaultWatchlist() *Watchlist {
	return &Watchlist{
		Symbols: []string{
			"SPY", "QQQ", "IWM",
			"AAPL", "MSFT", "NVDA",
			"TSLA", "AMD", "META", "AMZN",
		},
	}
}
