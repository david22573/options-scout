// Package packet implements the manual intake recommendation workflow.
package packet

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TradeWindow string

const (
	TradeWindowSameDay TradeWindow = "SAME_DAY"
	TradeWindowDTE13   TradeWindow = "DTE_1_3"
	TradeWindowDTE17   TradeWindow = "DTE_1_7"
	TradeWindowDTE1445 TradeWindow = "DTE_14_45"
)

type Bias string

const (
	BiasBullish Bias = "BULLISH"
	BiasBearish Bias = "BEARISH"
	BiasUnknown Bias = "UNKNOWN"
)

type Goal string

const (
	GoalScalp         Goal = "SCALP"
	GoalDayTrade      Goal = "DAY_TRADE"
	GoalHoldOvernight Goal = "HOLD_OVERNIGHT"
	GoalSwing         Goal = "SWING"
)

type SetupType string

const (
	SetupBreakout     SetupType = "BREAKOUT"
	SetupBreakdown    SetupType = "BREAKDOWN"
	SetupVWAPReclaim  SetupType = "VWAP_RECLAIM"
	SetupVWAPReject   SetupType = "VWAP_REJECTION"
	SetupFailedRetest SetupType = "FAILED_RETEST"
	SetupNewsPump     SetupType = "NEWS_PUMP"
	SetupChop         SetupType = "CHOP"
	SetupUnknown      SetupType = "UNKNOWN"
)

type StrategyHint string

const (
	StrategyCallDebit StrategyHint = "CALL_DEBIT_SPREAD"
	StrategyPutDebit  StrategyHint = "PUT_DEBIT_SPREAD"
	StrategyBearCall  StrategyHint = "BEAR_CALL_CREDIT_SPREAD"
	StrategyBullPut   StrategyHint = "BULL_PUT_CREDIT_SPREAD"
	StrategyUnknown   StrategyHint = "UNKNOWN"
)

type OptionType string

const (
	OptionTypeCall OptionType = "CALL"
	OptionTypePut  OptionType = "PUT"
)

type Packet struct {
	Symbol           string            `json:"symbol"`
	Timestamp        string            `json:"timestamp"`
	AccountMaxRisk   float64           `json:"account_max_risk"`
	TradeWindow      TradeWindow       `json:"trade_window"`
	Bias             Bias              `json:"bias"`
	Goal             Goal              `json:"goal"`
	TemplateMode     bool              `json:"template_mode,omitempty"`
	Underlying       Underlying        `json:"underlying"`
	Setup            Setup             `json:"setup"`
	OptionCandidates []OptionCandidate `json:"option_candidates"`
}

type Underlying struct {
	CurrentPrice       float64 `json:"current_price"`
	PremarketDirection string  `json:"premarket_direction"`
	DayHigh            float64 `json:"day_high"`
	DayLow             float64 `json:"day_low"`
	VWAP               float64 `json:"vwap"`
	AboveVWAP          *bool   `json:"above_vwap"`
	OpeningRangeHigh   float64 `json:"opening_range_high"`
	OpeningRangeLow    float64 `json:"opening_range_low"`
	MajorSupport       float64 `json:"major_support"`
	MajorResistance    float64 `json:"major_resistance"`
	RelativeVolume     float64 `json:"relative_volume"`
	SPYDirection       string  `json:"spy_direction"`
	QQQDirection       string  `json:"qqq_direction"`
	Notes              string  `json:"notes"`
}

type Setup struct {
	SetupType         SetupType `json:"setup_type"`
	TriggerLevel      float64   `json:"trigger_level"`
	InvalidationLevel float64   `json:"invalidation_level"`
	ConfirmationNotes string    `json:"confirmation_notes"`
	EarningsSoon      bool      `json:"earnings_soon"`
	MajorNews         string    `json:"major_news"`
}

type OptionCandidate struct {
	StrategyHint         StrategyHint `json:"strategy_hint"`
	Expiration           string       `json:"expiration"`
	LongLegType          OptionType   `json:"long_leg_type"`
	LongLegStrike        float64      `json:"long_leg_strike"`
	LongLegBid           float64      `json:"long_leg_bid"`
	LongLegAsk           float64      `json:"long_leg_ask"`
	LongLegVolume        int          `json:"long_leg_volume"`
	LongLegOpenInterest  int          `json:"long_leg_open_interest"`
	ShortLegType         OptionType   `json:"short_leg_type"`
	ShortLegStrike       float64      `json:"short_leg_strike"`
	ShortLegBid          float64      `json:"short_leg_bid"`
	ShortLegAsk          float64      `json:"short_leg_ask"`
	ShortLegVolume       int          `json:"short_leg_volume"`
	ShortLegOpenInterest int          `json:"short_leg_open_interest"`
	EstimatedDebit       float64      `json:"estimated_debit"`
	EstimatedCredit      float64      `json:"estimated_credit"`
	Notes                string       `json:"notes"`
}

type ValidationResult struct {
	Valid          bool     `json:"valid"`
	TemplateMode   bool     `json:"template_mode"`
	MissingFields  []string `json:"missing_fields,omitempty"`
	Errors         []string `json:"errors,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	CandidateCount int      `json:"candidate_count"`
}

func Template(symbol string) Packet {
	return Packet{
		Symbol:         symbol,
		Timestamp:      "",
		AccountMaxRisk: 0,
		TradeWindow:    "",
		Bias:           "",
		Goal:           "",
		TemplateMode:   true,
		Underlying: Underlying{
			AboveVWAP: boolPtr(false),
		},
		Setup:            Setup{SetupType: ""},
		OptionCandidates: []OptionCandidate{{}},
	}
}

func Load(path string) (Packet, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Packet{}, nil, fmt.Errorf("packet: read %s: %w", path, err)
	}
	var p Packet
	if err := json.Unmarshal(data, &p); err != nil {
		return Packet{}, nil, fmt.Errorf("packet: parse %s: %w", path, err)
	}
	return p, data, nil
}

func Write(path string, p Packet) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("packet: mkdir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("packet: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("packet: write %s: %w", path, err)
	}
	return nil
}

func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func ParseTimestamp(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	t, err := time.Parse(time.RFC3339, v)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("2006-01-02 15:04:05", v)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("timestamp must be RFC3339 or 2006-01-02 15:04:05")
}

func boolPtr(v bool) *bool {
	return &v
}
