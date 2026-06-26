# options-scout

> Liquid, defined-risk options spread recommender. No auto-trading. Default output: **WAIT**.

## What It Does

`options-scout` scans a liquid watchlist and recommends one of:

```text
CALL_DEBIT_SPREAD
PUT_DEBIT_SPREAD
BEAR_CALL_CREDIT_SPREAD
BULL_PUT_CREDIT_SPREAD
WAIT
```

Every recommendation includes entry trigger, invalidation level, max risk, max profit, and breakeven. All execution is manual. See [SAFETY.md](SAFETY.md).

## Quick Start

```bash
# Build
GOWORK=off go build -buildvcs=false -o options-scout ./cmd/options-scout

# Recommend a single symbol (V1: uses a manual chain file)
./options-scout recommend --symbol QQQ --manual-chain examples/chain_QQQ.json --max-risk 150

# Nightly watchlist scan
./options-scout nightly --watchlist configs/watchlists/liquid_core.yaml

# Morning confirmation
./options-scout morning --watchlist configs/watchlists/liquid_core.yaml --dte-min 1 --dte-max 7

# Journal review stats
./options-scout review
```

## Commands

| Command | Purpose |
|---------|---------|
| `recommend` | Single-symbol: best spread or WAIT |
| `nightly` | Build conditional trade plans for tomorrow |
| `morning` | Confirm yesterday's plans into entries or WAIT |
| `snapshot` | Save market snapshot JSON |
| `journal` | Append a decision record |
| `grade` | Update outcome on a journal entry |
| `review` | Print journal performance stats |

## Scoring Model (100 points)

| Component | Weight |
|-----------|--------|
| Liquidity | 0–25 |
| Setup | 0–25 |
| Market Context | 0–20 |
| Option Quality | 0–20 |
| Risk/Reward | 0–10 |

Thresholds:
- **85–100**: A setup → enter
- **70–84**: B setup → enter
- **60–69**: Watch only
- **0–59**: WAIT

## V1 Workflow

V1 uses manually saved option chain JSON files (`examples/chain_SYMBOL.json`).

After the spread math and scoring engine are validated, live data providers (Polygon, Alpaca) can be plugged in via `configs/config.yaml`.

## Chain File Format

```json
{
  "symbol": "QQQ",
  "underlying_last": 484.50,
  "contracts": [
    {
      "expiration": "2026-07-02",
      "strike": 485.0,
      "option_type": "call",
      "bid": 3.90,
      "ask": 4.10,
      "volume": 528,
      "open_interest": 5100,
      "iv": 0.20,
      "delta": 0.42,
      "theta": -0.20
    }
  ]
}
```

## Project Structure

```text
options-scout/
  cmd/options-scout/main.go        — CLI entry point (cobra)
  internal/
    config/          — YAML config + watchlist loader
    marketdata/      — Provider interface + manual file loader
    optionsdata/     — Chain types + FileProvider + liquidity checks
    features/        — Trend, ATR, VWAP, relative strength, opening range
    scanner/         — Nightly + morning scan orchestrators
    strategies/      — Call/put debit + credit spread selectors
    scoring/         — 100-pt composite score
    risk/            — Spread math, position sizing, daily limits
    journal/         — JSONL append-only log + stats
    report/          — Markdown + JSON writers
  configs/
    config.yaml              — Risk limits, scoring thresholds
    watchlists/liquid_core.yaml
  examples/
    chain_QQQ.json           — Sample chain for testing
  runs/
    snapshots/
    reports/
    journal/
```

## Risk Rules (defaults)

```text
Max risk per trade:  $150
Max risk per day:    $300
Max open risk:       $500
Max same direction:  2 trades
```

Override in `configs/config.yaml` or via `--max-risk` flag.

## First Success Target

Generate 30 journaled recommendations. After 30:
- Calculate win rate, expectancy, avg R
- Measure WAIT accuracy (did WAIT avoid bad days?)
- Run `options-scout review` to see the full summary
