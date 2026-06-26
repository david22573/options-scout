// Package features — relative strength versus a benchmark.
package features

// RSResult is the relative strength of a symbol vs a benchmark.
type RSResult struct {
	SymbolReturn    float64 // period return of the symbol
	BenchmarkReturn float64 // period return of SPY/QQQ
	RSRatio         float64 // > 1.0 means symbol outperforming
	StrongerThan    bool
}

// RelativeStrength computes simple price-return RS.
// symbolCloses and benchmarkCloses should be same-length daily close slices.
func RelativeStrength(symbolCloses, benchmarkCloses []float64, period int) RSResult {
	if len(symbolCloses) < period+1 || len(benchmarkCloses) < period+1 {
		return RSResult{}
	}

	sStart := symbolCloses[len(symbolCloses)-period-1]
	sEnd := symbolCloses[len(symbolCloses)-1]
	bStart := benchmarkCloses[len(benchmarkCloses)-period-1]
	bEnd := benchmarkCloses[len(benchmarkCloses)-1]

	sReturn := 0.0
	if sStart > 0 {
		sReturn = (sEnd - sStart) / sStart
	}
	bReturn := 0.0
	if bStart > 0 {
		bReturn = (bEnd - bStart) / bStart
	}

	ratio := 1.0
	if bReturn != 0 {
		ratio = (1 + sReturn) / (1 + bReturn)
	}

	return RSResult{
		SymbolReturn:    sReturn,
		BenchmarkReturn: bReturn,
		RSRatio:         ratio,
		StrongerThan:    ratio > 1.0,
	}
}
