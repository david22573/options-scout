// Package strategies — re-exported score type alias for CLI use.
package strategies

import "github.com/davidmiguel22573/options-scout/internal/scoring"

// ScoreValue is an alias so the CLI can use strategies.ScoreValue
// without importing the scoring package directly.
type ScoreValue = scoring.Score
