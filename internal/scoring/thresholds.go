// Package scoring — decision thresholds and grade labels.
package scoring

// Grade is the quality label derived from a total score.
type Grade string

const (
	GradeA     Grade = "A" // 85-100: enter
	GradeB     Grade = "B" // 70-84:  enter
	GradeWatch Grade = "C" // 60-69:  watch only
	GradeWait  Grade = "D" // 0-59:   WAIT
)

// Classify maps a total score to a Grade.
func Classify(total int) Grade {
	switch {
	case total >= 85:
		return GradeA
	case total >= 70:
		return GradeB
	case total >= 60:
		return GradeWatch
	default:
		return GradeWait
	}
}

// ShouldTrade returns true if the score meets the minimum trade threshold.
func ShouldTrade(s Score, minScore int) bool {
	return s.Total >= minScore
}
