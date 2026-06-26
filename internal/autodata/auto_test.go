package autodata

import (
	"strings"
	"testing"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
)

type mockProvider struct {
	quotes   map[string]*marketdata.Quote
	daily    map[string][]marketdata.Candle
	intraday map[string][]marketdata.Candle
	chain    *optionsdata.Chain
	chainErr error
}

func (m mockProvider) Name() string { return "alpaca" }
func (m mockProvider) Quote(symbol string) (*marketdata.Quote, error) {
	return m.quotes[strings.ToUpper(symbol)], nil
}
func (m mockProvider) DailyCandles(symbol string, count int) ([]marketdata.Candle, error) {
	return m.daily[strings.ToUpper(symbol)], nil
}
func (m mockProvider) IntradayCandles(symbol, resolution string, count int) ([]marketdata.Candle, error) {
	return m.intraday[strings.ToUpper(symbol)], nil
}
func (m mockProvider) Chain(symbol string) (*optionsdata.Chain, error) {
	return m.chain, m.chainErr
}
func (m mockProvider) MarketClock() (*ClockInfo, error) {
	return &ClockInfo{
		Timestamp: time.Now(),
		IsOpen:    true,
		NextClose: time.Now().Add(2 * time.Hour),
		NextOpen:  time.Now().Add(18 * time.Hour),
	}, nil
}

func TestAutoPacketFillsUnderlyingSnapshot(t *testing.T) {
	provider := goodMockProvider()
	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	if result.Packet.Underlying.CurrentPrice <= 0 || result.Packet.Underlying.DayHigh <= 0 || result.Packet.Underlying.DayLow <= 0 {
		t.Fatalf("expected underlying snapshot fields to be filled: %+v", result.Packet.Underlying)
	}
}

func TestAutoPacketCreatesCallSpreadCandidate(t *testing.T) {
	provider := goodMockProvider()
	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	if countCandidates(result, "CALL_DEBIT_SPREAD") == 0 {
		t.Fatalf("expected at least one call spread candidate, got %#v", result.Packet.OptionCandidates)
	}
}

func TestAutoPacketCreatesPutSpreadCandidate(t *testing.T) {
	provider := goodMockProvider()
	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	if countCandidates(result, "PUT_DEBIT_SPREAD") == 0 {
		t.Fatalf("expected at least one put spread candidate, got %#v", result.Packet.OptionCandidates)
	}
}

func TestWideBidAskCausesCandidateRejection(t *testing.T) {
	provider := goodMockProvider()
	provider.chain.Contracts[0].Bid = 1.00
	provider.chain.Contracts[0].Ask = 3.00
	provider.chain.Contracts[1].Bid = 0.20
	provider.chain.Contracts[1].Ask = 1.80
	provider.chain.Contracts[2].Bid = 1.00
	provider.chain.Contracts[2].Ask = 3.00
	provider.chain.Contracts[3].Bid = 0.20
	provider.chain.Contracts[3].Ask = 1.80

	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Packet.OptionCandidates) != 0 {
		t.Fatalf("expected no candidates, got %#v", result.Packet.OptionCandidates)
	}
	if !hasReason(result.ReasonCodes, "WIDE_BID_ASK") {
		t.Fatalf("expected WIDE_BID_ASK reason, got %v", result.ReasonCodes)
	}
}

func TestMaxRiskViolationCausesCandidateRejection(t *testing.T) {
	provider := goodMockProvider()
	provider.chain.Contracts[0].Bid = 2.50
	provider.chain.Contracts[0].Ask = 2.60
	provider.chain.Contracts[1].Bid = 1.90
	provider.chain.Contracts[1].Ask = 2.00
	provider.chain.Contracts[2].Bid = 2.55
	provider.chain.Contracts[2].Ask = 2.65
	provider.chain.Contracts[3].Bid = 1.95
	provider.chain.Contracts[3].Ask = 2.05

	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Packet.OptionCandidates) != 0 {
		t.Fatalf("expected no candidates, got %#v", result.Packet.OptionCandidates)
	}
	if !hasReason(result.ReasonCodes, "MAX_RISK_EXCEEDED") {
		t.Fatalf("expected MAX_RISK_EXCEEDED reason, got %v", result.ReasonCodes)
	}
}

func TestMissingOptionChainReturnsWait(t *testing.T) {
	provider := goodMockProvider()
	provider.chain = nil

	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	rec := BuildWaitRecommendation(result, "x.json", "sha")
	if rec.Decision != "WAIT" {
		t.Fatalf("decision = %s", rec.Decision)
	}
}

func TestStaleDataReturnsWait(t *testing.T) {
	provider := goodMockProvider()
	provider.quotes["QQQ"].Timestamp = time.Now().Add(-30 * time.Minute)

	result, err := BuildAutoPacket(provider, config.Default(), BuildInput{Symbol: "QQQ", MaxRisk: 50, DTEMin: 1, DTEMax: 7})
	if err != nil {
		t.Fatal(err)
	}
	rec := BuildWaitRecommendation(result, "x.json", "sha")
	if rec.Decision != "WAIT" {
		t.Fatalf("decision = %s", rec.Decision)
	}
	if !hasReason(result.ReasonCodes, "STALE_DATA") {
		t.Fatalf("expected STALE_DATA reason, got %v", result.ReasonCodes)
	}
}

func TestMissingAlpacaCredentialsReturnsCleanError(t *testing.T) {
	cfg := config.Default()
	cfg.Data.Provider = "alpaca"
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
	want := "Missing Alpaca credentials."
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}

func countCandidates(result BuildResult, strategy string) int {
	count := 0
	for _, candidate := range result.Packet.OptionCandidates {
		if string(candidate.StrategyHint) == strategy {
			count++
		}
	}
	return count
}

func goodMockProvider() mockProvider {
	now := time.Now()
	expiration := now.Add(48 * time.Hour).Truncate(24 * time.Hour)
	quote := &marketdata.Quote{
		Symbol:    "QQQ",
		Last:      500.00,
		Bid:       499.95,
		Ask:       500.05,
		Volume:    1000000,
		Timestamp: now,
	}
	spyQuote := &marketdata.Quote{Symbol: "SPY", Last: 600, Timestamp: now}

	daily := []marketdata.Candle{
		{Time: now.AddDate(0, 0, -2), Open: 495, High: 499, Low: 492, Close: 498, Volume: 1000000},
		{Time: now.AddDate(0, 0, -1), Open: 498, High: 501, Low: 497, Close: 499, Volume: 1100000},
		{Time: now, Open: 499, High: 501, Low: 498, Close: 500, Volume: 1200000},
	}
	intraday := []marketdata.Candle{
		{Time: now.Add(-60 * time.Minute), Open: 499.0, High: 499.3, Low: 498.8, Close: 499.2, Volume: 10000},
		{Time: now.Add(-59 * time.Minute), Open: 499.2, High: 499.6, Low: 499.1, Close: 499.5, Volume: 11000},
		{Time: now.Add(-58 * time.Minute), Open: 499.5, High: 500.4, Low: 499.4, Close: 500.2, Volume: 12000},
		{Time: now.Add(-57 * time.Minute), Open: 500.2, High: 500.5, Low: 500.0, Close: 500.3, Volume: 11500},
	}

	return mockProvider{
		quotes: map[string]*marketdata.Quote{
			"QQQ": quote,
			"SPY": spyQuote,
		},
		daily: map[string][]marketdata.Candle{
			"QQQ": daily,
			"SPY": daily,
		},
		intraday: map[string][]marketdata.Candle{
			"QQQ": intraday,
		},
		chain: &optionsdata.Chain{
			Symbol:         "QQQ",
			UnderlyingLast: 500,
			Contracts: []optionsdata.Contract{
				{Symbol: "QQQCALL500", Expiration: expiration, Strike: 500, OptionType: "call", Bid: 2.20, Ask: 2.30, Mid: 2.25, Volume: 200, OpenInterest: 1200, DTE: 2},
				{Symbol: "QQQCALL501", Expiration: expiration, Strike: 501, OptionType: "call", Bid: 1.80, Ask: 1.90, Mid: 1.85, Volume: 190, OpenInterest: 1100, DTE: 2},
				{Symbol: "QQQPUT500", Expiration: expiration, Strike: 500, OptionType: "put", Bid: 2.30, Ask: 2.40, Mid: 2.35, Volume: 210, OpenInterest: 1250, DTE: 2},
				{Symbol: "QQQPUT499", Expiration: expiration, Strike: 499, OptionType: "put", Bid: 1.90, Ask: 2.00, Mid: 1.95, Volume: 205, OpenInterest: 1180, DTE: 2},
			},
		},
	}
}
