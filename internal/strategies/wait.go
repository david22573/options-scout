// Package strategies — shared recommendation types.
package strategies

import (
	"time"

	"github.com/davidmiguel22573/options-scout/internal/scoring"
)

// Decision is the final output label.
type Decision string

const (
	DecisionCallDebitSpread Decision = "CALL_DEBIT_SPREAD"
	DecisionPutDebitSpread  Decision = "PUT_DEBIT_SPREAD"
	DecisionBearCallCredit  Decision = "BEAR_CALL_CREDIT_SPREAD"
	DecisionBullPutCredit   Decision = "BULL_PUT_CREDIT_SPREAD"
	DecisionWait            Decision = "WAIT"
)

// NightlyLabel is the watchlist classification from the nightly scan.
type NightlyLabel string

const (
	NightlyWatchCall  NightlyLabel = "WATCHLIST_CALL"
	NightlyWatchPut   NightlyLabel = "WATCHLIST_PUT"
	NightlyCreditFade NightlyLabel = "WATCHLIST_CREDIT_FADE"
	NightlyIgnore     NightlyLabel = "IGNORE"
)

// Confidence is an A/B/C/D grade.
type Confidence = scoring.Grade

// Recommendation is the full output of a strategy evaluation.
type Recommendation struct {
	Symbol        string
	Decision      Decision
	Setup         string // human label e.g. "Opening range breakdown + VWAP retest"
	Underlying    float64
	Expiration    time.Time
	LongStrike    float64
	ShortStrike   float64
	Debit         float64 // net debit (positive) or credit (negative)
	MaxRisk       float64 // dollars per contract
	MaxProfit     float64 // dollars per contract
	Breakeven     float64
	EntryTrigger  string
	Invalidation  string
	TakeProfitPct [2]float64 // e.g. [0.30, 0.70]
	StopLossPct   [2]float64
	Confidence    Confidence
	ScoreTotal    int
	ScoreDetail   string
	WaitReason    string // only populated when Decision == WAIT
}

// NightlyCandidate is the nightly scan output for a single symbol.
type NightlyCandidate struct {
	Symbol       string
	Label        NightlyLabel
	Bias         string
	Plan         string
	Trigger      string
	Invalidation string
	SuggestedDTE string
	MaxRisk      string
	Note         string
}
