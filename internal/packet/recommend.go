package packet

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/risk"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

type RecommendationResult struct {
	Symbol           string              `json:"symbol"`
	Timestamp        time.Time           `json:"timestamp"`
	PacketPath       string              `json:"packet_path"`
	PacketSHA256     string              `json:"packet_sha256"`
	Decision         strategies.Decision `json:"decision"`
	WaitReason       string              `json:"wait_reason,omitempty"`
	Grade            string              `json:"grade"`
	Score            int                 `json:"score"`
	Setup            string              `json:"setup"`
	EntryTrigger     string              `json:"entry_trigger"`
	Invalidation     string              `json:"invalidation"`
	Spread           string              `json:"spread"`
	Expiration       string              `json:"expiration"`
	DebitOrCredit    string              `json:"debit_or_credit"`
	MaxRisk          float64             `json:"max_risk"`
	MaxProfit        float64             `json:"max_profit"`
	Breakeven        float64             `json:"breakeven"`
	TakeProfitPlan   string              `json:"take_profit_plan"`
	StopCondition    string              `json:"stop_condition"`
	ReasonCodes      []string            `json:"reason_codes"`
	MissingFields    []string            `json:"missing_fields,omitempty"`
	Validation       ValidationResult    `json:"validation"`
	CandidateSummary string              `json:"candidate_summary,omitempty"`
}

func Recommend(p Packet, packetPath string, packetSHA string, maxRisk float64, cfg *config.Config) RecommendationResult {
	now := time.Now()
	validation := Validate(p)
	result := RecommendationResult{
		Symbol:       p.Symbol,
		Timestamp:    now,
		PacketPath:   packetPath,
		PacketSHA256: packetSHA,
		Decision:     strategies.DecisionWait,
		Grade:        "WAIT",
		Score:        0,
		Setup:        string(p.Setup.SetupType),
		Validation:   validation,
	}

	if len(validation.MissingFields) > 0 || len(validation.Errors) > 0 || validation.TemplateMode {
		result.ReasonCodes = append(result.ReasonCodes, "MISSING_FIELDS")
		if containsString(validation.MissingFields, "setup.trigger_level") {
			result.ReasonCodes = append(result.ReasonCodes, "NO_TRIGGER")
		}
		if containsString(validation.MissingFields, "setup.invalidation_level") {
			result.ReasonCodes = append(result.ReasonCodes, "NO_INVALIDATION")
		}
		if validation.TemplateMode {
			result.ReasonCodes = append(result.ReasonCodes, "TEMPLATE_MODE")
		}
		if len(validation.Errors) > 0 {
			result.ReasonCodes = append(result.ReasonCodes, "INVALID_PACKET")
		}
		result.MissingFields = validation.MissingFields
		return result
	}

	if maxRisk <= 0 {
		maxRisk = p.AccountMaxRisk
	}

	if p.Setup.TriggerLevel <= 0 || p.Setup.InvalidationLevel <= 0 {
		result.ReasonCodes = []string{"NO_TRIGGER", "NO_INVALIDATION"}
		return result
	}

	if shouldWaitForChop(p) {
		result.ReasonCodes = []string{"CHOP_NEAR_VWAP", "NO_CONFIRMATION"}
		return result
	}

	bestIdx := -1
	bestScore := -1.0
	var best recommendationCandidate
	for i, c := range p.OptionCandidates {
		eval := evaluateCandidate(p, c, maxRisk, cfg)
		if eval.waitReason != "" {
			continue
		}
		if eval.score > bestScore {
			bestIdx = i
			bestScore = eval.score
			best = eval
		}
	}

	if bestIdx == -1 {
		for _, c := range p.OptionCandidates {
			eval := evaluateCandidate(p, c, maxRisk, cfg)
			if eval.waitReason == "" {
				continue
			}
			result.ReasonCodes = append(result.ReasonCodes, eval.reasonCodes...)
			break
		}
		if len(result.ReasonCodes) == 0 {
			result.ReasonCodes = []string{"NO_VALID_CANDIDATE"}
		}
		return result
	}

	grade := scoring.Classify(best.totalScore)
	result.Decision = best.decision
	result.Grade = string(grade)
	result.Score = best.totalScore
	result.Setup = best.setup
	result.EntryTrigger = fmt.Sprintf("%.2f", p.Setup.TriggerLevel)
	result.Invalidation = fmt.Sprintf("%.2f", p.Setup.InvalidationLevel)
	result.Spread = best.spread
	result.Expiration = best.expiration.Format("2006-01-02")
	result.DebitOrCredit = best.debitOrCredit
	result.MaxRisk = best.math.MaxRisk
	result.MaxProfit = best.math.MaxProfit
	result.Breakeven = best.math.Breakeven
	result.TakeProfitPlan = best.takeProfit
	result.StopCondition = best.stopCondition
	result.ReasonCodes = best.reasonCodes
	result.CandidateSummary = fmt.Sprintf("selected option_candidates[%d]", bestIdx)
	return result
}

type recommendationCandidate struct {
	decision      strategies.Decision
	setup         string
	expiration    time.Time
	math          risk.SpreadMath
	debitOrCredit string
	spread        string
	takeProfit    string
	stopCondition string
	reasonCodes   []string
	totalScore    int
	score         float64
	waitReason    string
}

func evaluateCandidate(p Packet, c OptionCandidate, maxRisk float64, cfg *config.Config) recommendationCandidate {
	exp, err := time.Parse("2006-01-02", c.Expiration)
	if err != nil {
		return recommendationCandidate{
			waitReason:  "bad expiration",
			reasonCodes: []string{"INVALID_EXPIRATION"},
		}
	}

	if wideBidAsk(c, cfg.Filters.MaxBidAskSpreadPct) {
		return recommendationCandidate{
			waitReason:  "wide bid ask",
			reasonCodes: []string{"WIDE_BID_ASK"},
		}
	}

	if c.LongLegVolume < cfg.Filters.MinVolume || c.ShortLegVolume < cfg.Filters.MinVolume ||
		c.LongLegOpenInterest < cfg.Filters.MinOpenInterest || c.ShortLegOpenInterest < cfg.Filters.MinOpenInterest {
		return recommendationCandidate{
			waitReason:  "low liquidity",
			reasonCodes: []string{"LOW_LIQUIDITY"},
		}
	}

	var math risk.SpreadMath
	var decision strategies.Decision
	var setup string
	var debitOrCredit string

	switch c.StrategyHint {
	case StrategyCallDebit:
		if c.LongLegType != OptionTypeCall || c.ShortLegType != OptionTypeCall || c.ShortLegStrike <= c.LongLegStrike {
			return recommendationCandidate{waitReason: "invalid call debit structure", reasonCodes: []string{"INVALID_STRUCTURE"}}
		}
		debit := resolvedDebit(c)
		math, err = risk.CalcDebitSpread(c.LongLegStrike, c.ShortLegStrike, debit, true)
		decision = strategies.DecisionCallDebitSpread
		setup = "CALL_DEBIT_SPREAD"
		debitOrCredit = fmt.Sprintf("DEBIT %.2f", debit)
	case StrategyPutDebit:
		if c.LongLegType != OptionTypePut || c.ShortLegType != OptionTypePut || c.LongLegStrike <= c.ShortLegStrike {
			return recommendationCandidate{waitReason: "invalid put debit structure", reasonCodes: []string{"INVALID_STRUCTURE"}}
		}
		debit := resolvedDebit(c)
		math, err = risk.CalcDebitSpread(c.LongLegStrike, c.ShortLegStrike, debit, false)
		decision = strategies.DecisionPutDebitSpread
		setup = "PUT_DEBIT_SPREAD"
		debitOrCredit = fmt.Sprintf("DEBIT %.2f", debit)
	case StrategyBearCall:
		if c.LongLegType != OptionTypeCall || c.ShortLegType != OptionTypeCall || c.LongLegStrike <= c.ShortLegStrike {
			return recommendationCandidate{waitReason: "invalid bear call structure", reasonCodes: []string{"INVALID_STRUCTURE"}}
		}
		credit := resolvedCredit(c)
		math, err = risk.CalcCreditSpread(c.ShortLegStrike, c.LongLegStrike, credit, true)
		decision = strategies.DecisionBearCallCredit
		setup = "BEAR_CALL_CREDIT_SPREAD"
		debitOrCredit = fmt.Sprintf("CREDIT %.2f", credit)
	case StrategyBullPut:
		if c.LongLegType != OptionTypePut || c.ShortLegType != OptionTypePut || c.ShortLegStrike <= c.LongLegStrike {
			return recommendationCandidate{waitReason: "invalid bull put structure", reasonCodes: []string{"INVALID_STRUCTURE"}}
		}
		credit := resolvedCredit(c)
		math, err = risk.CalcCreditSpread(c.ShortLegStrike, c.LongLegStrike, credit, false)
		decision = strategies.DecisionBullPutCredit
		setup = "BULL_PUT_CREDIT_SPREAD"
		debitOrCredit = fmt.Sprintf("CREDIT %.2f", credit)
	default:
		return recommendationCandidate{waitReason: "unknown strategy", reasonCodes: []string{"UNKNOWN_STRATEGY"}}
	}
	if err != nil {
		return recommendationCandidate{waitReason: err.Error(), reasonCodes: []string{"INVALID_SPREAD_MATH"}}
	}
	if math.MaxRisk > maxRisk {
		return recommendationCandidate{
			waitReason:  "max risk exceeded",
			reasonCodes: []string{"MAX_RISK_EXCEEDED"},
		}
	}

	totalScore := scorePacket(p, c, math)
	if !scoring.ShouldTrade(scoring.Score{Total: totalScore}, cfg.Scoring.MinScoreTrade) {
		return recommendationCandidate{waitReason: "score below trade threshold", reasonCodes: []string{"LOW_SCORE"}}
	}

	return recommendationCandidate{
		decision:      decision,
		setup:         setup,
		expiration:    exp,
		math:          math,
		debitOrCredit: debitOrCredit,
		spread:        formatSpread(c, exp),
		takeProfit:    takeProfitPlan(c.StrategyHint),
		stopCondition: fmt.Sprintf("invalidate below/above %.2f or spread value hits stop zone", p.Setup.InvalidationLevel),
		reasonCodes:   scoreReasonCodes(p, c, math),
		totalScore:    totalScore,
		score:         math.RiskReward + float64(totalScore),
	}
}

func scorePacket(p Packet, c OptionCandidate, math risk.SpreadMath) int {
	liquidity := 18
	if !wideBidAsk(c, 0.05) {
		liquidity += 4
	}
	if c.LongLegOpenInterest >= 500 && c.ShortLegOpenInterest >= 500 {
		liquidity += 3
	}

	setupScore := 10
	switch p.Setup.SetupType {
	case SetupBreakout, SetupBreakdown, SetupVWAPReclaim, SetupVWAPReject, SetupFailedRetest:
		setupScore = 20
	case SetupNewsPump:
		setupScore = 14
	case SetupChop, SetupUnknown:
		setupScore = 5
	}
	if strings.TrimSpace(p.Setup.ConfirmationNotes) != "" {
		setupScore += 3
	}

	market := 10
	if p.Underlying.RelativeVolume >= 1.5 {
		market += 4
	}
	if alignedWithBias(p, c) {
		market += 4
	}

	optionQuality := 12
	if c.LongLegVolume >= 100 && c.ShortLegVolume >= 100 {
		optionQuality += 4
	}
	if c.Notes != "" {
		optionQuality += 2
	}

	rr := 4
	if math.RiskReward >= 1.5 {
		rr = 8
	}
	if math.RiskReward >= 2.0 {
		rr = 10
	}

	return scoring.Calculate(liquidity, setupScore, market, optionQuality, rr).Total
}

func alignedWithBias(p Packet, c OptionCandidate) bool {
	switch p.Bias {
	case BiasBullish:
		return c.StrategyHint == StrategyCallDebit || c.StrategyHint == StrategyBullPut
	case BiasBearish:
		return c.StrategyHint == StrategyPutDebit || c.StrategyHint == StrategyBearCall
	default:
		return false
	}
}

func wideBidAsk(c OptionCandidate, maxSpread float64) bool {
	return legSpread(c.LongLegBid, c.LongLegAsk) > maxSpread || legSpread(c.ShortLegBid, c.ShortLegAsk) > maxSpread
}

func legSpread(bid, ask float64) float64 {
	mid := (bid + ask) / 2
	if mid <= 0 {
		return math.MaxFloat64
	}
	return (ask - bid) / mid
}

func resolvedDebit(c OptionCandidate) float64 {
	if c.EstimatedDebit > 0 {
		return c.EstimatedDebit
	}
	return ((c.LongLegBid+c.LongLegAsk)/2 - (c.ShortLegBid+c.ShortLegAsk)/2)
}

func resolvedCredit(c OptionCandidate) float64 {
	if c.EstimatedCredit > 0 {
		return c.EstimatedCredit
	}
	return ((c.ShortLegBid+c.ShortLegAsk)/2 - (c.LongLegBid+c.LongLegAsk)/2)
}

func shouldWaitForChop(p Packet) bool {
	if p.Setup.SetupType != SetupChop && p.Setup.SetupType != SetupUnknown {
		return false
	}
	nearVWAP := math.Abs(p.Underlying.CurrentPrice-p.Underlying.VWAP) <= math.Max(0.25, p.Underlying.CurrentPrice*0.0015)
	return nearVWAP && strings.TrimSpace(p.Setup.ConfirmationNotes) == ""
}

func formatSpread(c OptionCandidate, exp time.Time) string {
	suffix := "C"
	if c.LongLegType == OptionTypePut {
		suffix = "P"
	}
	return fmt.Sprintf("%.0f%s/%.0f%s %s", c.LongLegStrike, suffix, c.ShortLegStrike, suffix, exp.Format("2006-01-02"))
}

func takeProfitPlan(hint StrategyHint) string {
	switch hint {
	case StrategyBearCall, StrategyBullPut:
		return "Take 30%-50% of max profit early; exit faster on adverse move."
	default:
		return "Scale at 30%-50%, leave runner to 70% if trend and volume hold."
	}
}

func scoreReasonCodes(p Packet, c OptionCandidate, math risk.SpreadMath) []string {
	codes := []string{"MANUAL_PACKET"}
	if alignedWithBias(p, c) {
		codes = append(codes, "BIAS_ALIGNED")
	}
	if math.RiskReward >= 1.5 {
		codes = append(codes, "GOOD_RR")
	}
	if p.Underlying.RelativeVolume >= 1.5 {
		codes = append(codes, "RELVOL_CONFIRMED")
	}
	if strings.TrimSpace(p.Setup.ConfirmationNotes) != "" {
		codes = append(codes, "CONFIRMED")
	}
	return codes
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
