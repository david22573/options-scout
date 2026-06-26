// Package risk — position sizing and per-trade risk limits.
package risk

import (
	"fmt"

	"github.com/davidmiguel22573/options-scout/internal/config"
)

// SizeResult holds the recommended position size.
type SizeResult struct {
	Contracts    int
	TotalRisk    float64
	Allowed      bool
	RejectReason string
}

// SizePosition returns the number of contracts that fit within maxRisk.
// It also checks the configured per-trade limit.
func SizePosition(sm SpreadMath, maxRiskUSD float64, cfg *config.AccountConfig) SizeResult {
	if sm.MaxRisk <= 0 {
		return SizeResult{0, 0, false, "spread has no defined max risk"}
	}
	if maxRiskUSD <= 0 {
		maxRiskUSD = cfg.MaxRiskPerTrade
	}
	if maxRiskUSD > cfg.MaxRiskPerTrade {
		maxRiskUSD = cfg.MaxRiskPerTrade
	}

	contracts := int(maxRiskUSD / sm.MaxRisk)
	if contracts < 1 {
		return SizeResult{
			0, 0, false,
			fmt.Sprintf("max risk $%.0f < single contract risk $%.0f", maxRiskUSD, sm.MaxRisk),
		}
	}

	totalRisk := float64(contracts) * sm.MaxRisk
	return SizeResult{
		Contracts: contracts,
		TotalRisk: totalRisk,
		Allowed:   true,
	}
}
