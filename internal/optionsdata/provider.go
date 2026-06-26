// Package optionsdata defines core types and the provider interface for options chain data.
package optionsdata

import "time"

// Contract is a single options contract row.
type Contract struct {
	Symbol       string
	Expiration   time.Time
	Strike       float64
	OptionType   string // "call" | "put"
	Bid          float64
	Ask          float64
	Mid          float64
	Volume       int
	OpenInterest int
	IV           float64 // implied volatility as decimal (0.35 = 35%)
	Delta        float64
	Theta        float64
	DTE          int // days to expiration
}

// BidAskSpreadPct returns the bid-ask spread as a fraction of the mid price.
// Returns 1.0 (100%) when mid is zero to force rejection.
func (c *Contract) BidAskSpreadPct() float64 {
	if c.Mid <= 0 {
		return 1.0
	}
	return (c.Ask - c.Bid) / c.Mid
}

// Chain is the full options chain for an underlying at a snapshot point in time.
type Chain struct {
	Symbol         string
	UnderlyingLast float64
	Contracts      []Contract
}

// CallsForExpiry returns all call contracts for a given DTE bucket (±2 days).
func (ch *Chain) CallsForExpiry(dte int) []Contract {
	return ch.filterByDTE("call", dte)
}

// PutsForExpiry returns all put contracts for a given DTE bucket.
func (ch *Chain) PutsForExpiry(dte int) []Contract {
	return ch.filterByDTE("put", dte)
}

func (ch *Chain) filterByDTE(optType string, dte int) []Contract {
	var out []Contract
	for _, c := range ch.Contracts {
		if c.OptionType == optType && abs(c.DTE-dte) <= 2 {
			out = append(out, c)
		}
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Provider is the interface that wraps options chain retrieval.
type Provider interface {
	// Chain returns the option chain for a symbol.
	Chain(symbol string) (*Chain, error)
}
