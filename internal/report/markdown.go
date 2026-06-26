// Package report — markdown report generation.
package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/scanner"
	"github.com/davidmiguel22573/options-scout/internal/scoring"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

// SpreadScoreDetail returns a formatted score breakdown string.
func SpreadScoreDetail(sc scoring.Score) string {
	return scoring.Explain(sc)
}

// RenderNightly produces a markdown nightly report.
func RenderNightly(r *scanner.NightlyReport, date time.Time) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Options Scout — Nightly Report\n**Date:** %s\n\n", date.Format("2006-01-02")))
	sb.WriteString("> ⚠️ Night-before plans are NOT entries. Confirm at market open.\n\n")

	if len(r.Candidates) == 0 {
		sb.WriteString("## Result: No Candidates\n\n")
		sb.WriteString(r.Note + "\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("## Candidates (%d)\n\n", len(r.Candidates)))
	for _, c := range r.Candidates {
		sb.WriteString(fmt.Sprintf("---\n### %s — `%s`\n\n", c.Symbol, c.Label))
		sb.WriteString(fmt.Sprintf("**Bias:** %s\n\n", c.Bias))
		sb.WriteString(fmt.Sprintf("**Plan:** %s\n\n", c.Plan))
		sb.WriteString(fmt.Sprintf("**Trigger:** %s\n\n", c.Trigger))
		sb.WriteString(fmt.Sprintf("**Invalidation:** %s\n\n", c.Invalidation))
		sb.WriteString(fmt.Sprintf("**Suggested DTE:** %s | **Max Risk:** %s\n\n", c.SuggestedDTE, c.MaxRisk))
		if c.Note != "" {
			sb.WriteString(fmt.Sprintf("> %s\n\n", c.Note))
		}
		sb.WriteString("**Status:** `WATCHLIST_ONLY` — do not enter without morning confirmation\n\n")
	}
	return sb.String()
}

// RenderMorning produces a markdown morning report.
func RenderMorning(recs []strategies.Recommendation, notes []string, date time.Time) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Options Scout — Morning Report\n**Date:** %s\n\n", date.Format("2006-01-02")))

	for i, rec := range recs {
		sb.WriteString(fmt.Sprintf("---\n### %s — `%s`\n\n", rec.Symbol, rec.Decision))
		if rec.Decision == strategies.DecisionWait {
			sb.WriteString(fmt.Sprintf("**Decision:** WAIT\n\n"))
			sb.WriteString(fmt.Sprintf("**Reason:** %s\n\n", rec.WaitReason))
		} else {
			sb.WriteString(fmt.Sprintf("**Decision:** %s\n\n", rec.Decision))
			sb.WriteString(fmt.Sprintf("**Setup:** %s\n\n", rec.Setup))
			sb.WriteString(fmt.Sprintf("**Underlying:** %.2f\n\n", rec.Underlying))
			if !rec.Expiration.IsZero() {
				sb.WriteString(fmt.Sprintf("**Expiration:** %s\n\n", rec.Expiration.Format("2006-01-02")))
			}
			sb.WriteString(fmt.Sprintf("**Buy:** %.0f | **Sell:** %.0f\n\n", rec.LongStrike, rec.ShortStrike))
			if rec.Debit >= 0 {
				sb.WriteString(fmt.Sprintf("**Estimated debit:** %.2f\n\n", rec.Debit))
			} else {
				sb.WriteString(fmt.Sprintf("**Estimated credit:** %.2f\n\n", -rec.Debit))
			}
			sb.WriteString(fmt.Sprintf("**Max Risk:** $%.2f | **Max Profit:** $%.2f | **Breakeven:** %.2f\n\n",
				rec.MaxRisk, rec.MaxProfit, rec.Breakeven))
			sb.WriteString(fmt.Sprintf("**Entry trigger:** %s\n\n", rec.EntryTrigger))
			sb.WriteString(fmt.Sprintf("**Invalidation:** %s\n\n", rec.Invalidation))
			sb.WriteString(fmt.Sprintf("**Take profit:** +%.0f%% to +%.0f%%\n\n",
				rec.TakeProfitPct[0]*100, rec.TakeProfitPct[1]*100))
			sb.WriteString(fmt.Sprintf("**Stop:** -%.0f%% to -%.0f%%\n\n",
				rec.StopLossPct[0]*100, rec.StopLossPct[1]*100))
			sb.WriteString(fmt.Sprintf("**Confidence:** %s | **Score:** %d/100\n\n",
				rec.Confidence, rec.ScoreTotal))
		}
		if i < len(notes) && notes[i] != "" {
			sb.WriteString("```\n" + notes[i] + "\n```\n\n")
		}
	}
	return sb.String()
}

// RenderRecommendation renders a single recommendation to markdown.
func RenderRecommendation(rec strategies.Recommendation) string {
	return RenderMorning([]strategies.Recommendation{rec}, []string{rec.ScoreDetail}, time.Now())
}
