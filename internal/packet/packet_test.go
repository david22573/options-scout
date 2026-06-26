package packet

import (
	"testing"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

func TestTemplateContainsRequiredFields(t *testing.T) {
	p := Template("QQQ")
	if p.Symbol != "QQQ" {
		t.Fatalf("symbol = %q", p.Symbol)
	}
	if len(p.OptionCandidates) != 1 {
		t.Fatalf("expected one blank candidate, got %d", len(p.OptionCandidates))
	}
	if p.Underlying.AboveVWAP == nil {
		t.Fatal("expected above_vwap pointer")
	}
}

func TestIncompletePacketValidation(t *testing.T) {
	p := Template("QQQ")
	v := Validate(p)
	if v.Valid {
		t.Fatal("expected invalid template packet")
	}
	if len(v.MissingFields) == 0 {
		t.Fatal("expected missing fields")
	}
}

func TestRecommendGoodCallDebitPacket(t *testing.T) {
	p := goodBasePacket()
	p.Bias = BiasBullish
	p.Setup.SetupType = SetupBreakout
	p.OptionCandidates = []OptionCandidate{{
		StrategyHint:         StrategyCallDebit,
		Expiration:           "2026-06-26",
		LongLegType:          OptionTypeCall,
		LongLegStrike:        500,
		LongLegBid:           2.45,
		LongLegAsk:           2.55,
		LongLegVolume:        250,
		LongLegOpenInterest:  1500,
		ShortLegType:         OptionTypeCall,
		ShortLegStrike:       501,
		ShortLegBid:          2.00,
		ShortLegAsk:          2.08,
		ShortLegVolume:       200,
		ShortLegOpenInterest: 1200,
		EstimatedDebit:       0.48,
		Notes:                "tight market",
	}}

	r := Recommend(p, "x.json", "sha", 50, config.Default())
	if r.Decision != strategies.DecisionCallDebitSpread {
		t.Fatalf("decision = %s", r.Decision)
	}
}

func TestRecommendGoodPutDebitPacket(t *testing.T) {
	p := goodBasePacket()
	p.Bias = BiasBearish
	p.Setup.SetupType = SetupBreakdown
	p.OptionCandidates = []OptionCandidate{{
		StrategyHint:         StrategyPutDebit,
		Expiration:           "2026-06-26",
		LongLegType:          OptionTypePut,
		LongLegStrike:        499,
		LongLegBid:           2.30,
		LongLegAsk:           2.40,
		LongLegVolume:        250,
		LongLegOpenInterest:  1500,
		ShortLegType:         OptionTypePut,
		ShortLegStrike:       498,
		ShortLegBid:          1.85,
		ShortLegAsk:          1.95,
		ShortLegVolume:       200,
		ShortLegOpenInterest: 1200,
		EstimatedDebit:       0.45,
		Notes:                "tight market",
	}}

	r := Recommend(p, "x.json", "sha", 50, config.Default())
	if r.Decision != strategies.DecisionPutDebitSpread {
		t.Fatalf("decision = %s", r.Decision)
	}
}

func TestBadLiquidityForcesWait(t *testing.T) {
	p := goodBasePacket()
	p.OptionCandidates = []OptionCandidate{{
		StrategyHint:         StrategyCallDebit,
		Expiration:           "2026-06-26",
		LongLegType:          OptionTypeCall,
		LongLegStrike:        500,
		LongLegBid:           2.00,
		LongLegAsk:           3.00,
		LongLegVolume:        250,
		LongLegOpenInterest:  1500,
		ShortLegType:         OptionTypeCall,
		ShortLegStrike:       501,
		ShortLegBid:          1.40,
		ShortLegAsk:          2.30,
		ShortLegVolume:       200,
		ShortLegOpenInterest: 1200,
		EstimatedDebit:       0.65,
	}}
	r := Recommend(p, "x.json", "sha", 50, config.Default())
	if r.Decision != strategies.DecisionWait {
		t.Fatalf("decision = %s", r.Decision)
	}
}

func TestOverMaxRiskForcesWait(t *testing.T) {
	p := goodBasePacket()
	p.OptionCandidates = []OptionCandidate{{
		StrategyHint:         StrategyCallDebit,
		Expiration:           "2026-06-26",
		LongLegType:          OptionTypeCall,
		LongLegStrike:        500,
		LongLegBid:           2.45,
		LongLegAsk:           2.55,
		LongLegVolume:        250,
		LongLegOpenInterest:  1500,
		ShortLegType:         OptionTypeCall,
		ShortLegStrike:       501,
		ShortLegBid:          2.00,
		ShortLegAsk:          2.08,
		ShortLegVolume:       200,
		ShortLegOpenInterest: 1200,
		EstimatedDebit:       0.75,
	}}
	r := Recommend(p, "x.json", "sha", 50, config.Default())
	if r.Decision != strategies.DecisionWait {
		t.Fatalf("decision = %s", r.Decision)
	}
}

func TestNoTriggerInvalidationForcesWait(t *testing.T) {
	p := goodBasePacket()
	p.Setup.TriggerLevel = 0
	p.Setup.InvalidationLevel = 0
	r := Recommend(p, "x.json", "sha", 50, config.Default())
	if r.Decision != strategies.DecisionWait {
		t.Fatalf("decision = %s", r.Decision)
	}
}

func goodBasePacket() Packet {
	aboveVWAP := true
	return Packet{
		Symbol:         "QQQ",
		Timestamp:      "2026-06-25T09:45:00Z",
		AccountMaxRisk: 50,
		TradeWindow:    TradeWindowDTE13,
		Bias:           BiasBullish,
		Goal:           GoalDayTrade,
		Underlying: Underlying{
			CurrentPrice:       499.2,
			PremarketDirection: "UP",
			DayHigh:            500.1,
			DayLow:             497.8,
			VWAP:               498.7,
			AboveVWAP:          &aboveVWAP,
			OpeningRangeHigh:   499.5,
			OpeningRangeLow:    498.3,
			MajorSupport:       497.5,
			MajorResistance:    500.0,
			RelativeVolume:     1.8,
			SPYDirection:       "UP",
			QQQDirection:       "UP",
			Notes:              "clean trend",
		},
		Setup: Setup{
			SetupType:         SetupBreakout,
			TriggerLevel:      499.6,
			InvalidationLevel: 498.5,
			ConfirmationNotes: "opening range break with tape support",
			EarningsSoon:      false,
			MajorNews:         "",
		},
	}
}
