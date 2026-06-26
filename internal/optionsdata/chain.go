// Package optionsdata — JSON file-based chain loader for V1 manual mode.
package optionsdata

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// chainJSON is the JSON schema for manual chain files.
type chainJSON struct {
	Symbol         string         `json:"symbol"`
	UnderlyingLast float64        `json:"underlying_last"`
	Contracts      []contractJSON `json:"contracts"`
}

type contractJSON struct {
	Expiration   string  `json:"expiration"` // "2026-06-27"
	Strike       float64 `json:"strike"`
	OptionType   string  `json:"option_type"` // "call" | "put"
	Bid          float64 `json:"bid"`
	Ask          float64 `json:"ask"`
	Volume       int     `json:"volume"`
	OpenInterest int     `json:"open_interest"`
	IV           float64 `json:"iv"`
	Delta        float64 `json:"delta"`
	Theta        float64 `json:"theta"`
}

// FileProvider loads an option chain from a JSON file.
type FileProvider struct {
	path string
}

// NewFileProvider returns a FileProvider reading from path.
func NewFileProvider(path string) *FileProvider {
	return &FileProvider{path: path}
}

// Chain loads the chain from the configured JSON file.
func (p *FileProvider) Chain(symbol string) (*Chain, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return nil, fmt.Errorf("chain file: read %s: %w", p.path, err)
	}

	var raw chainJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("chain file: parse %s: %w", p.path, err)
	}

	now := time.Now()
	ch := &Chain{
		Symbol:         raw.Symbol,
		UnderlyingLast: raw.UnderlyingLast,
	}

	for _, r := range raw.Contracts {
		exp, err := time.Parse("2006-01-02", r.Expiration)
		if err != nil {
			return nil, fmt.Errorf("chain file: bad expiration %q: %w", r.Expiration, err)
		}
		dte := int(exp.Sub(now).Hours()/24) + 1
		mid := (r.Bid + r.Ask) / 2
		ch.Contracts = append(ch.Contracts, Contract{
			Symbol:       raw.Symbol,
			Expiration:   exp,
			Strike:       r.Strike,
			OptionType:   strings.ToLower(r.OptionType),
			Bid:          r.Bid,
			Ask:          r.Ask,
			Mid:          mid,
			Volume:       r.Volume,
			OpenInterest: r.OpenInterest,
			IV:           r.IV,
			Delta:        r.Delta,
			Theta:        r.Theta,
			DTE:          dte,
		})
	}

	return ch, nil
}
