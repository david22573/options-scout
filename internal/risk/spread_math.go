// Package risk — spread math for defined-risk options positions.
package risk

import "fmt"

// SpreadMath holds calculated risk/reward metrics for a vertical spread.
type SpreadMath struct {
	SpreadWidth  float64 // distance between strikes in dollars
	Debit        float64 // net debit paid (debit spread) or credit received
	MaxRisk      float64 // maximum dollar loss per 1-lot
	MaxProfit    float64 // maximum dollar gain per 1-lot
	Breakeven    float64 // underlying price at breakeven
	RiskReward   float64 // max profit / max risk
	CreditSpread bool    // true if this is a credit spread
}

// CalcDebitSpread calculates risk/reward for a debit spread.
//
//	longStrike: the strike you BUY (near-the-money)
//	shortStrike: the strike you SELL (further OTM)
//	debit: net premium paid per share (use mid prices)
//	isCall: true for call debit spread, false for put debit spread
func CalcDebitSpread(longStrike, shortStrike, debit float64, isCall bool) (SpreadMath, error) {
	if debit <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: debit must be > 0")
	}

	width := shortStrike - longStrike
	if isCall && width <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: call spread requires short > long strike")
	}
	if !isCall {
		width = longStrike - shortStrike
		if width <= 0 {
			return SpreadMath{}, fmt.Errorf("spread math: put spread requires long > short strike")
		}
	}

	maxRisk := debit * 100 // 1 contract = 100 shares
	maxProfit := (width - debit) * 100
	if maxProfit <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: debit %.2f >= spread width %.2f — no profit possible", debit, width)
	}

	breakeven := 0.0
	if isCall {
		breakeven = longStrike + debit
	} else {
		breakeven = longStrike - debit
	}

	rr := 0.0
	if maxRisk > 0 {
		rr = maxProfit / maxRisk
	}

	return SpreadMath{
		SpreadWidth:  width,
		Debit:        debit,
		MaxRisk:      maxRisk,
		MaxProfit:    maxProfit,
		Breakeven:    breakeven,
		RiskReward:   rr,
		CreditSpread: false,
	}, nil
}

// CalcCreditSpread calculates risk/reward for a credit spread.
//
//	shortStrike: the strike you SELL (closer to money)
//	longStrike: the strike you BUY (further from money — protection)
//	credit: net premium received per share
//	isCall: true for bear call credit spread, false for bull put credit spread
func CalcCreditSpread(shortStrike, longStrike, credit float64, isCall bool) (SpreadMath, error) {
	if credit <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: credit must be > 0")
	}

	width := longStrike - shortStrike
	if isCall && width <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: call credit requires long > short strike")
	}
	if !isCall {
		width = shortStrike - longStrike
		if width <= 0 {
			return SpreadMath{}, fmt.Errorf("spread math: put credit requires short > long strike")
		}
	}

	maxProfit := credit * 100
	maxRisk := (width - credit) * 100
	if maxRisk <= 0 {
		return SpreadMath{}, fmt.Errorf("spread math: credit %.2f >= spread width %.2f — undefined risk", credit, width)
	}

	breakeven := 0.0
	if isCall {
		breakeven = shortStrike + credit
	} else {
		breakeven = shortStrike - credit
	}

	rr := 0.0
	if maxRisk > 0 {
		rr = maxProfit / maxRisk
	}

	return SpreadMath{
		SpreadWidth:  width,
		Debit:        credit,
		MaxRisk:      maxRisk,
		MaxProfit:    maxProfit,
		Breakeven:    breakeven,
		RiskReward:   rr,
		CreditSpread: true,
	}, nil
}
