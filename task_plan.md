# Options Scout — Task Plan
Created: 2026-06-25
Updated: 2026-06-25

## Goal
Build `options-scout` Go CLI: liquid, defined-risk options spread recommender.
No live trading. Manual execution only. WAIT is default output.

## Status: COMPLETE (V1 MVP)

## Milestones
| # | Name | Status |
|---|------|--------|
| M1 | Repo scaffold + planning files | complete |
| M2 | Core types, config, watchlist loader | complete |
| M3 | Spread math + scoring engine | complete |
| M4 | Strategies (call/put debit, credit, wait) | complete |
| M5 | Journal + report writers | complete |
| M6 | CLI commands (nightly, morning, recommend, journal, grade, review) | complete |
| M7 | Example chain + build verified | complete |

## Build Command
```bash
cd ~/Github/options-scout
GOWORK=off go build -buildvcs=false -o options-scout ./cmd/options-scout
```

## Quick Test
```bash
./options-scout recommend --symbol QQQ --manual-chain examples/chain_QQQ.json --max-risk 150 --dte-min 7
./options-scout review
```

## Next Steps (V2)
- Connect Polygon/Alpaca live data providers
- Improve morning scanner with real SPY intraday comparison
- Add `options-scout grade` interactive TUI
- Build 30-decision journal baseline

## Decisions Log
| Date | Decision | Reason |
|------|----------|--------|
| 2026-06-25 | Manual chain first (no live API) | Proves spread math before API complexity |
| 2026-06-25 | shopspring/decimal for money math | Avoids float rounding on options pricing |
| 2026-06-25 | GOWORK=off for builds | Parent go.work at ~/Github doesn't include this module yet |
