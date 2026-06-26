// Package journal — outcome grading and stats review.
package journal

import "fmt"

// Stats summarizes journal performance over all graded entries.
type Stats struct {
	TotalEntries     int
	TradeDecisions   int
	WaitDecisions    int
	PendingCount     int
	CorrectWaitCount int
	BadWaitCount     int
	WinCount         int
	LossCount        int
	BreakevenCount   int
	WinRate          float64 // wins / (wins+losses+breakevens)
	WaitAccuracy     float64 // correct_waits / (correct_waits + bad_waits)
	AvgMaxRisk       float64
	AvgMaxProfit     float64
}

// ComputeStats calculates performance statistics from a set of entries.
func ComputeStats(entries []Entry) Stats {
	var s Stats
	s.TotalEntries = len(entries)
	totalRisk := 0.0
	totalProfit := 0.0

	for _, e := range entries {
		if e.Decision == "WAIT" {
			s.WaitDecisions++
		} else {
			s.TradeDecisions++
			totalRisk += e.MaxRisk
			totalProfit += e.MaxProfit
		}

		switch e.Outcome {
		case OutcomeWin:
			s.WinCount++
		case OutcomeLoss:
			s.LossCount++
		case OutcomeBreakeven:
			s.BreakevenCount++
		case OutcomeCorrectWait:
			s.CorrectWaitCount++
		case OutcomeBadWait, OutcomeBadWaitMissed:
			s.BadWaitCount++
		case OutcomePending:
			s.PendingCount++
		}
	}

	traded := s.WinCount + s.LossCount + s.BreakevenCount
	if traded > 0 {
		s.WinRate = float64(s.WinCount) / float64(traded)
	}
	waits := s.CorrectWaitCount + s.BadWaitCount
	if waits > 0 {
		s.WaitAccuracy = float64(s.CorrectWaitCount) / float64(waits)
	}
	if s.TradeDecisions > 0 {
		s.AvgMaxRisk = totalRisk / float64(s.TradeDecisions)
		s.AvgMaxProfit = totalProfit / float64(s.TradeDecisions)
	}
	return s
}

// FormatStats returns a human-readable stats summary.
func FormatStats(s Stats) string {
	return fmt.Sprintf(
		"Journal Summary (%d total entries)\n"+
			"  Decisions:     %d trades / %d waits\n"+
			"  Trades:        %d (wins %d / losses %d / breakeven %d)\n"+
			"  Win Rate:      %.1f%%\n"+
			"  Waits:         %d (correct %d / bad %d)\n"+
			"  Wait Accuracy: %.1f%%\n"+
			"  Avg Max Risk:  $%.2f\n"+
			"  Avg Max Profit:$%.2f\n"+
			"  Pending:       %d",
		s.TotalEntries,
		s.TradeDecisions, s.WaitDecisions,
		s.WinCount+s.LossCount+s.BreakevenCount, s.WinCount, s.LossCount, s.BreakevenCount,
		s.WinRate*100,
		s.CorrectWaitCount+s.BadWaitCount, s.CorrectWaitCount, s.BadWaitCount,
		s.WaitAccuracy*100,
		s.AvgMaxRisk,
		s.AvgMaxProfit,
		s.PendingCount,
	)
}
