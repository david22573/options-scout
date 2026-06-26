package packet

import (
	"fmt"
	"strings"
)

func Validate(p Packet) ValidationResult {
	var missing []string
	var errs []string
	var warnings []string

	requireString(&missing, p.Symbol, "symbol")
	if p.Timestamp == "" {
		missing = append(missing, "timestamp")
	} else if _, err := ParseTimestamp(p.Timestamp); err != nil {
		errs = append(errs, fmt.Sprintf("timestamp: %v", err))
	}
	requirePositive(&missing, p.AccountMaxRisk, "account_max_risk")
	requireEnum(&missing, &errs, string(p.TradeWindow), "trade_window",
		string(TradeWindowSameDay), string(TradeWindowDTE13), string(TradeWindowDTE17), string(TradeWindowDTE1445))
	requireEnum(&missing, &errs, string(p.Bias), "bias",
		string(BiasBullish), string(BiasBearish), string(BiasUnknown))
	requireEnum(&missing, &errs, string(p.Goal), "goal",
		string(GoalScalp), string(GoalDayTrade), string(GoalHoldOvernight), string(GoalSwing))

	requirePositive(&missing, p.Underlying.CurrentPrice, "underlying.current_price")
	requirePositive(&missing, p.Underlying.DayHigh, "underlying.day_high")
	requirePositive(&missing, p.Underlying.DayLow, "underlying.day_low")
	requirePositive(&missing, p.Underlying.VWAP, "underlying.vwap")
	if p.Underlying.AboveVWAP == nil {
		missing = append(missing, "underlying.above_vwap")
	}
	requirePositive(&missing, p.Underlying.OpeningRangeHigh, "underlying.opening_range_high")
	requirePositive(&missing, p.Underlying.OpeningRangeLow, "underlying.opening_range_low")
	requirePositive(&missing, p.Underlying.MajorSupport, "underlying.major_support")
	requirePositive(&missing, p.Underlying.MajorResistance, "underlying.major_resistance")
	requirePositive(&missing, p.Underlying.RelativeVolume, "underlying.relative_volume")
	requireString(&missing, p.Underlying.PremarketDirection, "underlying.premarket_direction")
	requireString(&missing, p.Underlying.SPYDirection, "underlying.spy_direction")
	requireString(&missing, p.Underlying.QQQDirection, "underlying.qqq_direction")

	requireEnum(&missing, &errs, string(p.Setup.SetupType), "setup.setup_type",
		string(SetupBreakout), string(SetupBreakdown), string(SetupVWAPReclaim), string(SetupVWAPReject),
		string(SetupFailedRetest), string(SetupNewsPump), string(SetupChop), string(SetupUnknown))

	if p.Setup.SetupType != SetupChop && p.Setup.SetupType != SetupUnknown {
		requirePositive(&missing, p.Setup.TriggerLevel, "setup.trigger_level")
		requirePositive(&missing, p.Setup.InvalidationLevel, "setup.invalidation_level")
	}

	if len(p.OptionCandidates) == 0 {
		missing = append(missing, "option_candidates")
	}
	for i, c := range p.OptionCandidates {
		prefix := fmt.Sprintf("option_candidates[%d]", i)
		requireEnum(&missing, &errs, string(c.StrategyHint), prefix+".strategy_hint",
			string(StrategyCallDebit), string(StrategyPutDebit), string(StrategyBearCall), string(StrategyBullPut), string(StrategyUnknown))
		requireString(&missing, c.Expiration, prefix+".expiration")
		requireEnum(&missing, &errs, string(c.LongLegType), prefix+".long_leg_type", string(OptionTypeCall), string(OptionTypePut))
		requirePositive(&missing, c.LongLegStrike, prefix+".long_leg_strike")
		requirePositive(&missing, c.LongLegBid, prefix+".long_leg_bid")
		requirePositive(&missing, c.LongLegAsk, prefix+".long_leg_ask")
		requirePositiveInt(&missing, c.LongLegVolume, prefix+".long_leg_volume")
		requirePositiveInt(&missing, c.LongLegOpenInterest, prefix+".long_leg_open_interest")
		requireEnum(&missing, &errs, string(c.ShortLegType), prefix+".short_leg_type", string(OptionTypeCall), string(OptionTypePut))
		requirePositive(&missing, c.ShortLegStrike, prefix+".short_leg_strike")
		requirePositive(&missing, c.ShortLegBid, prefix+".short_leg_bid")
		requirePositive(&missing, c.ShortLegAsk, prefix+".short_leg_ask")
		requirePositiveInt(&missing, c.ShortLegVolume, prefix+".short_leg_volume")
		requirePositiveInt(&missing, c.ShortLegOpenInterest, prefix+".short_leg_open_interest")
		if c.EstimatedDebit <= 0 && c.EstimatedCredit <= 0 {
			missing = append(missing, prefix+".estimated_debit_or_credit")
		}
	}

	if p.TemplateMode {
		warnings = append(warnings, "packet is marked template_mode")
	}
	if p.Underlying.DayLow > 0 && p.Underlying.DayHigh > 0 && p.Underlying.DayLow > p.Underlying.DayHigh {
		errs = append(errs, "underlying.day_low cannot exceed underlying.day_high")
	}
	if p.Underlying.OpeningRangeLow > 0 && p.Underlying.OpeningRangeHigh > 0 &&
		p.Underlying.OpeningRangeLow > p.Underlying.OpeningRangeHigh {
		errs = append(errs, "underlying.opening_range_low cannot exceed underlying.opening_range_high")
	}
	if len(missing) > 0 {
		warnings = append(warnings, fmt.Sprintf("missing %d required field(s)", len(missing)))
	}

	return ValidationResult{
		Valid:          len(missing) == 0 && len(errs) == 0 && !p.TemplateMode,
		TemplateMode:   p.TemplateMode,
		MissingFields:  uniqueStrings(missing),
		Errors:         errs,
		Warnings:       warnings,
		CandidateCount: len(p.OptionCandidates),
	}
}

func requireString(missing *[]string, value, field string) {
	if strings.TrimSpace(value) == "" {
		*missing = append(*missing, field)
	}
}

func requirePositive(missing *[]string, value float64, field string) {
	if value <= 0 {
		*missing = append(*missing, field)
	}
}

func requirePositiveInt(missing *[]string, value int, field string) {
	if value <= 0 {
		*missing = append(*missing, field)
	}
}

func requireEnum(missing *[]string, errs *[]string, value, field string, allowed ...string) {
	if strings.TrimSpace(value) == "" {
		*missing = append(*missing, field)
		return
	}
	for _, v := range allowed {
		if value == v {
			return
		}
	}
	*errs = append(*errs, fmt.Sprintf("%s: invalid value %q", field, value))
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
