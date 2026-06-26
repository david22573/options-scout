package autodata

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/features"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
	"github.com/davidmiguel22573/options-scout/internal/packet"
	"github.com/davidmiguel22573/options-scout/internal/risk"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

type BuildInput struct {
	Symbol  string
	MaxRisk float64
	DTEMin  int
	DTEMax  int
}

type BuildResult struct {
	Packet      packet.Packet
	ReasonCodes []string
	Fatal       bool
	WaitReason  string
	Readiness   Readiness
}

func BuildAutoPacket(provider Provider, cfg *config.Config, in BuildInput) (BuildResult, error) {
	symbol := strings.ToUpper(strings.TrimSpace(in.Symbol))
	now := time.Now()
	result := BuildResult{
		Packet: packet.Packet{
			Symbol:         symbol,
			Timestamp:      now.Format(time.RFC3339),
			AccountMaxRisk: in.MaxRisk,
			TradeWindow:    chooseTradeWindow(in.DTEMin, in.DTEMax),
			Goal:           chooseGoal(in.DTEMax),
			Underlying: packet.Underlying{
				PremarketDirection: "UNKNOWN",
				SPYDirection:       "UNKNOWN",
				QQQDirection:       "UNKNOWN",
			},
			Setup: packet.Setup{
				SetupType:         packet.SetupUnknown,
				EarningsSoon:      false,
				MajorNews:         "UNKNOWN_NEWS_STATUS",
				ConfirmationNotes: "UNKNOWN_EARNINGS_STATUS; UNKNOWN_NEWS_STATUS",
			},
		},
		ReasonCodes: []string{
			"AUTO_PACKET",
			"DATA_PROVIDER_" + strings.ToUpper(provider.Name()),
		},
	}

	quote, err := provider.Quote(symbol)
	if err != nil {
		return result, fmt.Errorf("auto-packet: load quote for %s: %w", symbol, err)
	}
	var quoteTS *time.Time
	if quote != nil {
		quoteTS = &quote.Timestamp
	}
	result.Readiness.Clock = resolveMarketClock(provider, quoteTS)
	if quote == nil || quote.Last <= 0 {
		result.Fatal = true
		result.WaitReason = "underlying snapshot incomplete"
		result.Readiness.HasQuote = false
		return finalizeBuildResult(addReason(result, "UNDERLYING_SNAPSHOT_INCOMPLETE")), nil
	}
	result.Readiness.HasQuote = true
	result.Readiness.HasFreshData = result.Readiness.Clock.DataFreshness == FreshnessFresh
	switch result.Readiness.Clock.DataFreshness {
	case FreshnessStale:
		result.ReasonCodes = append(result.ReasonCodes, "STALE_DATA")
	case FreshnessIncomplete:
		result.ReasonCodes = append(result.ReasonCodes, "UNDERLYING_SNAPSHOT_INCOMPLETE")
	}

	daily, err := provider.DailyCandles(symbol, 20)
	if err != nil {
		return result, fmt.Errorf("auto-packet: load daily candles for %s: %w", symbol, err)
	}
	intraday, err := provider.IntradayCandles(symbol, "1m", 390)
	if err != nil {
		return result, fmt.Errorf("auto-packet: load intraday candles for %s: %w", symbol, err)
	}
	regularIntraday := regularSessionCandles(intraday, result.Readiness.Clock.Now)
	result.ReasonCodes = append(result.ReasonCodes, "UNDERLYING_SNAPSHOT_LOADED")

	result.Packet.Underlying.CurrentPrice = quote.Last
	result.Packet.Underlying.DayHigh = currentDayHigh(quote.Last, daily, intraday)
	result.Packet.Underlying.DayLow = currentDayLow(quote.Last, daily, intraday)
	result.Packet.Underlying.MajorSupport = majorSupport(quote.Last, daily)
	result.Packet.Underlying.MajorResistance = majorResistance(quote.Last, daily)
	result.Packet.Underlying.RelativeVolume = relativeVolume(quote.Volume, quote.RelVolume, daily, intraday)
	result.Packet.Underlying.PremarketDirection = premarketDirection(daily, intraday)
	result.Packet.Underlying.SPYDirection = marketDirection(provider, "SPY")
	result.Packet.Underlying.QQQDirection = marketDirection(provider, "QQQ")

	vwap := features.CalculateVWAP(regularIntraday)
	if vwap.VWAP > 0 {
		above := vwap.AboveVWAP
		result.Packet.Underlying.VWAP = vwap.VWAP
		result.Packet.Underlying.AboveVWAP = &above
		result.Readiness.VWAPAvailable = true
	} else {
		above := false
		result.Packet.Underlying.VWAP = quote.Last
		result.Packet.Underlying.AboveVWAP = &above
		result.ReasonCodes = append(result.ReasonCodes, "VWAP_UNKNOWN")
	}

	openingRange := features.OpeningRange(regularIntraday, 30)
	if openingRange.High > 0 && openingRange.Low > 0 {
		result.Packet.Underlying.OpeningRangeHigh = openingRange.High
		result.Packet.Underlying.OpeningRangeLow = openingRange.Low
		result.Readiness.OpeningRangeAvailable = len(regularIntraday) >= 30
	} else {
		result.Packet.Underlying.OpeningRangeHigh = quote.Last
		result.Packet.Underlying.OpeningRangeLow = quote.Last
		result.ReasonCodes = append(result.ReasonCodes, "OPENING_RANGE_UNKNOWN")
	}

	applySetup(&result.Packet)
	if result.Packet.Bias == packet.BiasUnknown {
		result.ReasonCodes = append(result.ReasonCodes, "MARKET_CONTEXT_MIXED", "WAIT_FOR_MORNING_CONFIRMATION")
	}

	chain, err := provider.Chain(symbol)
	if err != nil {
		result.Fatal = true
		result.WaitReason = "option chain unavailable"
		return finalizeBuildResult(addReason(result, "NO_VALID_SPREADS")), nil
	}
	if chain == nil || len(chain.Contracts) == 0 {
		result.Fatal = true
		result.WaitReason = "missing option chain"
		return finalizeBuildResult(addReason(result, "NO_VALID_SPREADS")), nil
	}
	result.ReasonCodes = append(result.ReasonCodes, "OPTION_CHAIN_LOADED")
	result.Readiness.OptionChainsLoaded = true
	if chain.UnderlyingLast <= 0 {
		chain.UnderlyingLast = quote.Last
	}

	candidates, candidateReasons := generateCandidates(chain, quote.Last, in.MaxRisk, in.DTEMin, in.DTEMax, cfg)
	result.ReasonCodes = append(result.ReasonCodes, candidateReasons...)
	result.Packet.OptionCandidates = candidates
	result.Packet.TemplateMode = false
	result.Packet.Underlying.Notes = strings.Join(uniqueStrings(result.ReasonCodes), ", ")

	if len(candidates) == 0 {
		result.Fatal = true
		if hasReason(result.ReasonCodes, "NO_EXPIRATION_IN_DTE_RANGE") {
			result.WaitReason = "no expiration in DTE range"
		} else {
			result.WaitReason = "no valid spreads"
			result.ReasonCodes = append(result.ReasonCodes, "NO_VALID_SPREADS")
		}
	}
	return finalizeBuildResult(result), nil
}

func BuildWaitRecommendation(br BuildResult, packetPath, packetSHA string) packet.RecommendationResult {
	return packet.RecommendationResult{
		Symbol:       br.Packet.Symbol,
		Timestamp:    time.Now(),
		PacketPath:   packetPath,
		PacketSHA256: packetSHA,
		Decision:     strategies.DecisionWait,
		Grade:        "WAIT",
		Setup:        string(br.Packet.Setup.SetupType),
		WaitReason:   br.WaitReason,
		Validation:   packet.Validate(br.Packet),
		ReasonCodes:  uniqueStrings(br.ReasonCodes),
	}
}

type scoredCandidate struct {
	candidate packet.OptionCandidate
	score     float64
}

func generateCandidates(chain *optionsdata.Chain, price, maxRisk float64, dteMin, dteMax int, cfg *config.Config) ([]packet.OptionCandidate, []string) {
	expirations := eligibleExpirations(chain.Contracts, dteMin, dteMax)
	if len(expirations) == 0 {
		return nil, []string{"NO_EXPIRATION_IN_DTE_RANGE"}
	}

	var (
		callScored []scoredCandidate
		putScored  []scoredCandidate
		reasons    []string
	)

	for _, expiration := range expirations {
		contracts := contractsForExpiration(chain.Contracts, expiration)
		callCandidates, callReasons := buildDebitCandidates(contracts, price, maxRisk, cfg, true)
		putCandidates, putReasons := buildDebitCandidates(contracts, price, maxRisk, cfg, false)
		callScored = append(callScored, callCandidates...)
		putScored = append(putScored, putCandidates...)
		reasons = append(reasons, callReasons...)
		reasons = append(reasons, putReasons...)
	}

	sort.SliceStable(callScored, func(i, j int) bool { return callScored[i].score > callScored[j].score })
	sort.SliceStable(putScored, func(i, j int) bool { return putScored[i].score > putScored[j].score })

	var out []packet.OptionCandidate
	out = append(out, topCandidates(callScored, 2)...)
	out = append(out, topCandidates(putScored, 2)...)
	if len(out) == 0 {
		reasons = append(reasons, "NO_VALID_SPREADS")
	}
	return out, uniqueStrings(reasons)
}

func buildDebitCandidates(contracts []optionsdata.Contract, price, maxRisk float64, cfg *config.Config, isCall bool) ([]scoredCandidate, []string) {
	var (
		longs   []optionsdata.Contract
		reasons []string
		out     []scoredCandidate
	)
	for _, contract := range contracts {
		if isCall && contract.OptionType == "call" {
			longs = append(longs, contract)
		}
		if !isCall && contract.OptionType == "put" {
			longs = append(longs, contract)
		}
	}

	sort.SliceStable(longs, func(i, j int) bool {
		return strikeDistance(longs[i], price, isCall) < strikeDistance(longs[j], price, isCall)
	})

	for _, long := range longs {
		short, ok := matchingShort(long, longs, isCall)
		if !ok {
			continue
		}
		if long.Bid <= 0 || long.Ask <= 0 || short.Bid <= 0 || short.Ask <= 0 {
			continue
		}

		if long.BidAskSpreadPct() > cfg.Filters.MaxBidAskSpreadPct || short.BidAskSpreadPct() > cfg.Filters.MaxBidAskSpreadPct {
			reasons = append(reasons, "WIDE_BID_ASK")
			continue
		}
		if long.OpenInterest < cfg.Filters.MinOpenInterest || short.OpenInterest < cfg.Filters.MinOpenInterest ||
			long.Volume < cfg.Filters.MinVolume || short.Volume < cfg.Filters.MinVolume {
			reasons = append(reasons, "LOW_OPTION_LIQUIDITY")
			continue
		}

		debit := long.Ask - short.Bid
		if debit <= 0 {
			continue
		}

		math, err := risk.CalcDebitSpread(long.Strike, short.Strike, debit, isCall)
		if err != nil {
			continue
		}
		if math.MaxRisk > maxRisk {
			reasons = append(reasons, "MAX_RISK_EXCEEDED")
			continue
		}

		hint := packet.StrategyCallDebit
		longType := packet.OptionTypeCall
		shortType := packet.OptionTypeCall
		if !isCall {
			hint = packet.StrategyPutDebit
			longType = packet.OptionTypePut
			shortType = packet.OptionTypePut
		}

		out = append(out, scoredCandidate{
			candidate: packet.OptionCandidate{
				StrategyHint:         hint,
				Expiration:           long.Expiration.Format("2006-01-02"),
				LongLegType:          longType,
				LongLegStrike:        long.Strike,
				LongLegBid:           long.Bid,
				LongLegAsk:           long.Ask,
				LongLegVolume:        long.Volume,
				LongLegOpenInterest:  long.OpenInterest,
				ShortLegType:         shortType,
				ShortLegStrike:       short.Strike,
				ShortLegBid:          short.Bid,
				ShortLegAsk:          short.Ask,
				ShortLegVolume:       short.Volume,
				ShortLegOpenInterest: short.OpenInterest,
				EstimatedDebit:       debit,
				Notes:                "AUTO_PACKET",
			},
			score: candidateScore(long, short, price, debit),
		})
	}
	return out, reasons
}

func matchingShort(long optionsdata.Contract, sameType []optionsdata.Contract, isCall bool) (optionsdata.Contract, bool) {
	bestScore := math.MaxFloat64
	var best optionsdata.Contract
	for _, candidate := range sameType {
		width := candidate.Strike - long.Strike
		if !isCall {
			width = long.Strike - candidate.Strike
		}
		if width < 0.99 || width > 2.01 {
			continue
		}
		score := math.Abs(width-1.0) + candidate.BidAskSpreadPct()
		if score < bestScore {
			bestScore = score
			best = candidate
		}
	}
	return best, bestScore != math.MaxFloat64
}

func strikeDistance(contract optionsdata.Contract, price float64, isCall bool) float64 {
	distance := math.Abs(contract.Strike - price)
	if isCall && contract.Strike <= price {
		distance -= 0.15
	}
	if !isCall && contract.Strike >= price {
		distance -= 0.15
	}
	return distance
}

func candidateScore(long, short optionsdata.Contract, price, debit float64) float64 {
	liquidity := float64(long.OpenInterest+short.OpenInterest+long.Volume+short.Volume) / 100.0
	spreadPenalty := long.BidAskSpreadPct()*100 + short.BidAskSpreadPct()*100
	distancePenalty := math.Abs(long.Strike-price) * 5
	debitPenalty := debit * 10
	return liquidity - spreadPenalty - distancePenalty - debitPenalty
}

func topCandidates(in []scoredCandidate, limit int) []packet.OptionCandidate {
	if len(in) < limit {
		limit = len(in)
	}
	out := make([]packet.OptionCandidate, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, item := range in {
		key := fmt.Sprintf("%s|%s|%.2f|%.2f", item.candidate.StrategyHint, item.candidate.Expiration, item.candidate.LongLegStrike, item.candidate.ShortLegStrike)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item.candidate)
		if len(out) == limit {
			break
		}
	}
	return out
}

func eligibleExpirations(contracts []optionsdata.Contract, dteMin, dteMax int) []time.Time {
	seen := make(map[string]time.Time)
	for _, contract := range contracts {
		if contract.DTE < dteMin || contract.DTE > dteMax {
			continue
		}
		seen[contract.Expiration.Format("2006-01-02")] = contract.Expiration
	}
	out := make([]time.Time, 0, len(seen))
	for _, expiration := range seen {
		out = append(out, expiration)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

func contractsForExpiration(contracts []optionsdata.Contract, expiration time.Time) []optionsdata.Contract {
	var out []optionsdata.Contract
	for _, contract := range contracts {
		if contract.Expiration.Format("2006-01-02") == expiration.Format("2006-01-02") {
			out = append(out, contract)
		}
	}
	return out
}

func chooseTradeWindow(dteMin, dteMax int) packet.TradeWindow {
	switch {
	case dteMax <= 0:
		return packet.TradeWindowSameDay
	case dteMax <= 3:
		return packet.TradeWindowDTE13
	case dteMax <= 7:
		return packet.TradeWindowDTE17
	default:
		return packet.TradeWindowDTE1445
	}
}

func chooseGoal(dteMax int) packet.Goal {
	if dteMax <= 3 {
		return packet.GoalDayTrade
	}
	return packet.GoalSwing
}

func applySetup(p *packet.Packet) {
	price := p.Underlying.CurrentPrice
	orHigh := p.Underlying.OpeningRangeHigh
	orLow := p.Underlying.OpeningRangeLow
	vwap := p.Underlying.VWAP

	switch {
	case price >= orHigh && price >= vwap:
		p.Bias = packet.BiasBullish
		p.Setup.SetupType = packet.SetupBreakout
		p.Setup.TriggerLevel = orHigh
		p.Setup.InvalidationLevel = math.Min(vwap, orLow)
		p.Setup.ConfirmationNotes = strings.TrimSpace(p.Setup.ConfirmationNotes + "; opening range breakout above VWAP")
	case price <= orLow && price <= vwap:
		p.Bias = packet.BiasBearish
		p.Setup.SetupType = packet.SetupBreakdown
		p.Setup.TriggerLevel = orLow
		p.Setup.InvalidationLevel = math.Max(vwap, orHigh)
		p.Setup.ConfirmationNotes = strings.TrimSpace(p.Setup.ConfirmationNotes + "; opening range breakdown below VWAP")
	default:
		p.Bias = packet.BiasUnknown
		p.Setup.SetupType = packet.SetupChop
		p.Setup.TriggerLevel = price
		p.Setup.InvalidationLevel = price
	}
}

func currentDayHigh(last float64, daily, intraday []marketdata.Candle) float64 {
	high := last
	if len(daily) > 0 && daily[len(daily)-1].High > high {
		high = daily[len(daily)-1].High
	}
	for _, candle := range intraday {
		if candle.High > high {
			high = candle.High
		}
	}
	return high
}

func currentDayLow(last float64, daily, intraday []marketdata.Candle) float64 {
	low := last
	if len(daily) > 0 && daily[len(daily)-1].Low > 0 && daily[len(daily)-1].Low < low {
		low = daily[len(daily)-1].Low
	}
	for _, candle := range intraday {
		if candle.Low > 0 && candle.Low < low {
			low = candle.Low
		}
	}
	return low
}

func majorSupport(last float64, daily []marketdata.Candle) float64 {
	if len(daily) == 0 {
		return last
	}
	support := daily[0].Low
	for _, candle := range daily {
		if candle.Low > 0 && candle.Low < support {
			support = candle.Low
		}
	}
	return support
}

func majorResistance(last float64, daily []marketdata.Candle) float64 {
	if len(daily) == 0 {
		return last
	}
	resistance := daily[0].High
	for _, candle := range daily {
		if candle.High > resistance {
			resistance = candle.High
		}
	}
	return resistance
}

func relativeVolume(quoteVolume, relVolume float64, daily, intraday []marketdata.Candle) float64 {
	if relVolume > 0 {
		return relVolume
	}
	if len(intraday) == 0 || len(daily) < 2 {
		return 1
	}
	var intradayVolume float64
	for _, candle := range intraday {
		intradayVolume += candle.Volume
	}
	if intradayVolume == 0 {
		intradayVolume = quoteVolume
	}
	var avg float64
	var count int
	for i := 0; i < len(daily)-1; i++ {
		if daily[i].Volume <= 0 {
			continue
		}
		avg += daily[i].Volume
		count++
	}
	if count == 0 || avg == 0 {
		return 1
	}
	return intradayVolume / (avg / float64(count))
}

func premarketDirection(daily, intraday []marketdata.Candle) string {
	if len(daily) < 2 || len(intraday) == 0 {
		return "UNKNOWN"
	}
	prevClose := daily[len(daily)-2].Close
	firstOpen := intraday[0].Open
	return directionFromPrices(firstOpen, prevClose)
}

func marketDirection(provider Provider, symbol string) string {
	quote, err := provider.Quote(symbol)
	if err != nil || quote == nil {
		return "UNKNOWN"
	}
	daily, err := provider.DailyCandles(symbol, 2)
	if err != nil || len(daily) < 2 {
		return "UNKNOWN"
	}
	return directionFromPrices(quote.Last, daily[len(daily)-2].Close)
}

func directionFromPrices(current, base float64) string {
	switch {
	case current > base:
		return "UP"
	case current < base:
		return "DOWN"
	default:
		return "FLAT"
	}
}

func dedupeReasons(result BuildResult) BuildResult {
	result.ReasonCodes = uniqueStrings(result.ReasonCodes)
	result.Readiness.Reasons = uniqueStrings(result.Readiness.Reasons)
	return result
}

func addReason(result BuildResult, code string) BuildResult {
	result.ReasonCodes = append(result.ReasonCodes, code)
	return dedupeReasons(result)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func hasReason(codes []string, target string) bool {
	for _, code := range codes {
		if code == target {
			return true
		}
	}
	return false
}

func readinessReasons(result BuildResult) []string {
	var reasons []string
	switch result.Readiness.Clock.Session {
	case SessionClosed:
		reasons = append(reasons, "WAIT_FOR_MARKET_OPEN")
	case SessionAfterHours:
		reasons = append(reasons, "WAIT_FOR_MARKET_OPEN")
	case SessionPremarket:
		reasons = append(reasons, "PREMARKET_SESSION")
	}
	switch result.Readiness.Clock.DataFreshness {
	case FreshnessStale:
		reasons = append(reasons, "STALE_DATA")
	case FreshnessIncomplete:
		reasons = append(reasons, "UNDERLYING_SNAPSHOT_INCOMPLETE")
	}
	if !result.Readiness.OptionChainsLoaded {
		reasons = append(reasons, "OPTION_CHAIN_UNAVAILABLE")
	}
	if !result.Readiness.OpeningRangeAvailable {
		reasons = append(reasons, "OPENING_RANGE_UNAVAILABLE")
	}
	if !result.Readiness.VWAPAvailable {
		reasons = append(reasons, "VWAP_UNAVAILABLE")
	}
	return uniqueStrings(reasons)
}

func recommendationAllowed(readiness Readiness) bool {
	return readiness.Clock.Session == SessionRegular &&
		readiness.Clock.DataFreshness == FreshnessFresh &&
		readiness.OptionChainsLoaded &&
		readiness.OpeningRangeAvailable &&
		readiness.VWAPAvailable
}

func finalizeBuildResult(result BuildResult) BuildResult {
	result.Readiness.Reasons = readinessReasons(result)
	result.Readiness.RecommendationAllowed = recommendationAllowed(result.Readiness)
	return dedupeReasons(result)
}
