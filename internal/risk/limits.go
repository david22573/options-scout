// Package risk — daily and portfolio-level risk limit enforcement.
package risk

import "fmt"

// DailyLimitCheck verifies that a proposed trade fits within daily and
// total open risk limits. In V1 limits are enforced based on configured
// values only; a future version will track open positions from the journal.
type DailyLimitCheck struct {
	CurrentDayRisk  float64
	CurrentOpenRisk float64
	MaxDayRisk      float64
	MaxOpenRisk     float64
}

// Check returns an error if adding newTradeRisk would breach a limit.
func (d DailyLimitCheck) Check(newTradeRisk float64) error {
	if d.CurrentDayRisk+newTradeRisk > d.MaxDayRisk {
		return fmt.Errorf("risk limit: daily limit $%.0f would be exceeded (current $%.0f + new $%.0f)",
			d.MaxDayRisk, d.CurrentDayRisk, newTradeRisk)
	}
	if d.CurrentOpenRisk+newTradeRisk > d.MaxOpenRisk {
		return fmt.Errorf("risk limit: open risk limit $%.0f would be exceeded (current $%.0f + new $%.0f)",
			d.MaxOpenRisk, d.CurrentOpenRisk, newTradeRisk)
	}
	return nil
}
