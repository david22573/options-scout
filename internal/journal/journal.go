// Package journal — append-only decision record for options-scout.
package journal

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/scoring"
	"github.com/davidmiguel22573/options-scout/internal/strategies"
)

// Outcome labels for a journaled decision.
type Outcome string

const (
	OutcomePending           Outcome = "PENDING"
	OutcomeWin               Outcome = "WIN"
	OutcomeLoss              Outcome = "LOSS"
	OutcomeBreakeven         Outcome = "BREAKEVEN"
	OutcomeCorrectWait       Outcome = "CORRECT_WAIT"
	OutcomeBadWait           Outcome = "BAD_WAIT"
	OutcomeBadWaitMissed     Outcome = "BAD_WAIT_MISSED_MOVE"
	OutcomeBadTradeDirection Outcome = "BAD_TRADE_DIRECTION"
	OutcomeBadTradeTiming    Outcome = "BAD_TRADE_TIMING"
	OutcomeBadLiquidity      Outcome = "BAD_LIQUIDITY"
	OutcomeInsufficientData  Outcome = "INSUFFICIENT_DATA"
)

// Entry is a single journal record.
type Entry struct {
	ID             string              `json:"id"`
	Timestamp      time.Time           `json:"timestamp"`
	Symbol         string              `json:"symbol"`
	SnapshotRef    string              `json:"snapshot_ref"`
	SnapshotSHA256 string              `json:"snapshot_sha256,omitempty"`
	PacketSHA256   string              `json:"packet_sha256,omitempty"`
	Decision       strategies.Decision `json:"decision"`
	Setup          string              `json:"setup"`
	Mode           string              `json:"mode"`
	EntryTrigger   string              `json:"entry_trigger"`
	Invalidation   string              `json:"invalidation"`
	Spread         string              `json:"spread"` // "428P/424P 2026-06-26"
	Debit          float64             `json:"debit"`
	MaxRisk        float64             `json:"max_risk"`
	MaxProfit      float64             `json:"max_profit"`
	Breakeven      float64             `json:"breakeven"`
	Outcome        Outcome             `json:"outcome"`
	Notes          string              `json:"notes"`
	ScoreTotal     int                 `json:"score_total"`
	ScoreBand      string              `json:"score_band,omitempty"`
	ReasonCodes    []string            `json:"reason_codes,omitempty"`
}

// Append writes a single Entry as a JSON line to the journal file.
func Append(path string, e Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("journal: mkdir %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("journal: open %s: %w", path, err)
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("journal: marshal entry: %w", err)
	}
	_, err = fmt.Fprintln(f, string(data))
	return err
}

// LoadAll reads all entries from a JSONL journal file.
func LoadAll(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("journal: read %s: %w", path, err)
	}

	var entries []Entry
	for i, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("journal: parse line %d: %w", i+1, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// FromRecommendation creates a journal entry from a Recommendation.
func FromRecommendation(rec strategies.Recommendation, snapshotRef string) Entry {
	now := time.Now()
	spread := "WAIT"
	setup := rec.Setup
	reasonCodes := []string(nil)
	if rec.Decision != strategies.DecisionWait {
		spread = fmt.Sprintf("%.0fC/%.0fC %s",
			rec.LongStrike, rec.ShortStrike, rec.Expiration.Format("2006-01-02"))
		if rec.Decision == strategies.DecisionPutDebitSpread ||
			rec.Decision == strategies.DecisionBullPutCredit {
			spread = fmt.Sprintf("%.0fP/%.0fP %s",
				rec.LongStrike, rec.ShortStrike, rec.Expiration.Format("2006-01-02"))
		}
	} else {
		setup = waitSetup(rec.ScoreTotal)
		reasonCodes = waitReasonCodes(rec.WaitReason)
	}
	return Entry{
		ID:             fmt.Sprintf("%s_%s_%s", rec.Symbol, rec.Decision, now.Format("20060102_150405")),
		Timestamp:      now,
		Symbol:         rec.Symbol,
		SnapshotRef:    snapshotRef,
		SnapshotSHA256: snapshotSHA256(snapshotRef),
		Decision:       rec.Decision,
		Setup:          setup,
		Mode:           "paper",
		EntryTrigger:   rec.EntryTrigger,
		Invalidation:   rec.Invalidation,
		Spread:         spread,
		Debit:          rec.Debit,
		MaxRisk:        rec.MaxRisk,
		MaxProfit:      rec.MaxProfit,
		Breakeven:      rec.Breakeven,
		Outcome:        OutcomePending,
		ScoreTotal:     rec.ScoreTotal,
		ScoreBand:      scoreBand(rec.ScoreTotal),
		ReasonCodes:    reasonCodes,
	}
}

func snapshotSHA256(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

func scoreBand(total int) string {
	switch scoring.Classify(total) {
	case scoring.GradeA:
		return "A"
	case scoring.GradeB:
		return "B"
	case scoring.GradeWatch:
		return "WATCH"
	default:
		return "WAIT"
	}
}

func waitSetup(total int) string {
	if total >= 60 && total <= 69 {
		return "WATCH_ONLY"
	}
	return "WAIT"
}

func waitReasonCodes(reason string) []string {
	reason = strings.ToLower(reason)
	var codes []string

	switch {
	case strings.Contains(reason, "liquid"):
		codes = append(codes, "LOW_LIQUIDITY")
	case strings.Contains(reason, "risk"):
		codes = append(codes, "RISK_LIMIT")
	case strings.Contains(reason, "score"):
		codes = append(codes, "LOW_SCORE")
	}

	if len(codes) == 0 {
		codes = append(codes, "WAIT_FILTER")
	}
	return codes
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
