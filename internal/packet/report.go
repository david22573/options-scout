package packet

import (
	"fmt"
	"strings"
)

func RenderValidation(v ValidationResult) string {
	var sb strings.Builder
	sb.WriteString("# Packet Validation\n\n")
	sb.WriteString(fmt.Sprintf("- Valid: `%t`\n", v.Valid))
	sb.WriteString(fmt.Sprintf("- Template mode: `%t`\n", v.TemplateMode))
	sb.WriteString(fmt.Sprintf("- Candidate count: `%d`\n", v.CandidateCount))
	if len(v.MissingFields) > 0 {
		sb.WriteString("- Missing fields:\n")
		for _, f := range v.MissingFields {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", f))
		}
	}
	if len(v.Errors) > 0 {
		sb.WriteString("- Errors:\n")
		for _, e := range v.Errors {
			sb.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}
	if len(v.Warnings) > 0 {
		sb.WriteString("- Warnings:\n")
		for _, w := range v.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", w))
		}
	}
	return sb.String()
}

func RenderRecommendation(r RecommendationResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Packet Recommendation — %s\n\n", r.Symbol))
	sb.WriteString(fmt.Sprintf("- Decision: `%s`\n", r.Decision))
	sb.WriteString(fmt.Sprintf("- Grade: `%s`\n", r.Grade))
	sb.WriteString(fmt.Sprintf("- Score: `%d`\n", r.Score))
	if r.Decision == "WAIT" && strings.TrimSpace(r.WaitReason) != "" {
		sb.WriteString(fmt.Sprintf("- Wait reason: `%s`\n", r.WaitReason))
	}
	if len(r.MissingFields) > 0 {
		sb.WriteString("- Missing fields:\n")
		for _, f := range r.MissingFields {
			sb.WriteString(fmt.Sprintf("  - `%s`\n", f))
		}
	}
	if r.Decision != "WAIT" {
		sb.WriteString(fmt.Sprintf("- Setup: `%s`\n", r.Setup))
		sb.WriteString(fmt.Sprintf("- Entry trigger: `%s`\n", r.EntryTrigger))
		sb.WriteString(fmt.Sprintf("- Invalidation: `%s`\n", r.Invalidation))
		sb.WriteString(fmt.Sprintf("- Spread: `%s`\n", r.Spread))
		sb.WriteString(fmt.Sprintf("- Expiration: `%s`\n", r.Expiration))
		sb.WriteString(fmt.Sprintf("- Debit/Credit: `%s`\n", r.DebitOrCredit))
		sb.WriteString(fmt.Sprintf("- Max risk: `$%.2f`\n", r.MaxRisk))
		sb.WriteString(fmt.Sprintf("- Max profit: `$%.2f`\n", r.MaxProfit))
		sb.WriteString(fmt.Sprintf("- Breakeven: `%.2f`\n", r.Breakeven))
		sb.WriteString(fmt.Sprintf("- Take profit plan: %s\n", r.TakeProfitPlan))
		sb.WriteString(fmt.Sprintf("- Stop condition: %s\n", r.StopCondition))
	}
	if len(r.ReasonCodes) > 0 {
		sb.WriteString("- Reason codes: ")
		for i, code := range r.ReasonCodes {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("`" + code + "`")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
