# Safety Rules

This project is a research and recommendation assistant.

## It MUST NOT:
- Place live trades automatically
- Automate Robinhood or any broker UI
- Use unofficial broker APIs
- Bypass broker protections
- Store broker credentials in committed files
- Recommend undefined-risk options (naked calls/puts, naked short shares)
- Recommend trades without a max-loss calculation

## It MUST:
- Default to WAIT unless the setup is clean
- Calculate and display max risk on every recommendation
- Include entry trigger and invalidation on every recommendation
- Respect configured per-trade and per-day risk limits
- Log every recommendation to the journal for later review

## All execution is manual.

No order placement logic shall be added to this codebase.
