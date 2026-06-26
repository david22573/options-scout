package autodata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/davidmiguel22573/options-scout/internal/config"
	"github.com/davidmiguel22573/options-scout/internal/marketdata"
	"github.com/davidmiguel22573/options-scout/internal/optionsdata"
)

const alpacaBaseURL = "https://data.alpaca.markets"

type AlpacaProvider struct {
	apiKey      string
	apiSecret   string
	stockFeed   string
	optionsFeed string
	client      *http.Client
}

func NewAlpacaProvider(cfg config.DataConfig) *AlpacaProvider {
	return &AlpacaProvider{
		apiKey:      cfg.AlpacaAPIKey,
		apiSecret:   cfg.AlpacaAPISecret,
		stockFeed:   defaultString(cfg.AlpacaFeed, "iex"),
		optionsFeed: defaultString(cfg.OptionsFeed, "opra"),
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *AlpacaProvider) Name() string {
	return "alpaca"
}

func (p *AlpacaProvider) Quote(symbol string) (*marketdata.Quote, error) {
	path := fmt.Sprintf("/v2/stocks/%s/snapshot", url.PathEscape(strings.ToUpper(symbol)))
	var payload stockSnapshotResponse
	if err := p.get(path, url.Values{"feed": []string{p.stockFeed}}, &payload); err != nil {
		return nil, err
	}

	last := payload.LatestTrade.Price
	if last <= 0 {
		last = payload.MinuteBar.Close
	}
	if last <= 0 {
		last = payload.DailyBar.Close
	}
	if last <= 0 && payload.LatestQuote.BidPrice > 0 && payload.LatestQuote.AskPrice > 0 {
		last = (payload.LatestQuote.BidPrice + payload.LatestQuote.AskPrice) / 2
	}

	ts := parseTimestamp(payload.LatestTrade.Timestamp)
	if ts.IsZero() {
		ts = parseTimestamp(payload.LatestQuote.Timestamp)
	}
	if ts.IsZero() {
		ts = parseTimestamp(payload.MinuteBar.Timestamp)
	}

	return &marketdata.Quote{
		Symbol:    strings.ToUpper(symbol),
		Last:      last,
		Bid:       payload.LatestQuote.BidPrice,
		Ask:       payload.LatestQuote.AskPrice,
		Volume:    payload.DailyBar.Volume,
		RelVolume: 0,
		Timestamp: ts,
	}, nil
}

func (p *AlpacaProvider) DailyCandles(symbol string, count int) ([]marketdata.Candle, error) {
	return p.bars(symbol, "1Day", count)
}

func (p *AlpacaProvider) IntradayCandles(symbol, resolution string, count int) ([]marketdata.Candle, error) {
	return p.bars(symbol, normalizeTimeframe(resolution), count)
}

func (p *AlpacaProvider) Chain(symbol string) (*optionsdata.Chain, error) {
	path := fmt.Sprintf("/v1beta1/options/snapshots/%s", url.PathEscape(strings.ToUpper(symbol)))
	var payload optionChainResponse
	if err := p.get(path, url.Values{"feed": []string{p.optionsFeed}}, &payload); err != nil {
		return nil, err
	}

	snapshots := payload.Snapshots
	if len(snapshots) == 0 {
		snapshots = payload.RootSnapshots
	}
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("alpaca options chain: no snapshots returned for %s", symbol)
	}

	chain := &optionsdata.Chain{Symbol: strings.ToUpper(symbol)}
	for contractSymbol, snap := range snapshots {
		expiration, strike, optionType, err := parseOptionSymbol(contractSymbol)
		if err != nil {
			continue
		}
		bid := firstPositive(snap.LatestQuote.BidPrice, snap.BidPrice)
		ask := firstPositive(snap.LatestQuote.AskPrice, snap.AskPrice)
		mid := 0.0
		if bid > 0 && ask > 0 {
			mid = (bid + ask) / 2
		}
		volume := firstPositiveInt(snap.LatestTrade.Size, snap.Day.Volume, snap.Volume)
		openInterest := firstPositiveInt(snap.OpenInterest, snap.OpenInterestAlt)
		iv := firstPositive(snap.ImpliedVolatility, snap.IV)
		delta := snap.Greeks.Delta
		theta := snap.Greeks.Theta
		dte := int(expiration.Sub(time.Now()).Hours()/24) + 1

		chain.Contracts = append(chain.Contracts, optionsdata.Contract{
			Symbol:       contractSymbol,
			Expiration:   expiration,
			Strike:       strike,
			OptionType:   optionType,
			Bid:          bid,
			Ask:          ask,
			Mid:          mid,
			Volume:       volume,
			OpenInterest: openInterest,
			IV:           iv,
			Delta:        delta,
			Theta:        theta,
			DTE:          dte,
		})
	}
	if len(chain.Contracts) == 0 {
		return nil, fmt.Errorf("alpaca options chain: no parseable contracts returned for %s", symbol)
	}
	return chain, nil
}

func (p *AlpacaProvider) MarketClock() (*ClockInfo, error) {
	var payload struct {
		Timestamp string `json:"timestamp"`
		IsOpen    bool   `json:"is_open"`
		NextOpen  string `json:"next_open"`
		NextClose string `json:"next_close"`
	}
	if err := p.get("/v2/clock", nil, &payload); err != nil {
		return nil, err
	}
	return &ClockInfo{
		Timestamp: parseTimestamp(payload.Timestamp),
		IsOpen:    payload.IsOpen,
		NextOpen:  parseTimestamp(payload.NextOpen),
		NextClose: parseTimestamp(payload.NextClose),
	}, nil
}

func (p *AlpacaProvider) bars(symbol, timeframe string, count int) ([]marketdata.Candle, error) {
	if count <= 0 {
		count = 1
	}
	var payload stockBarsResponse
	err := p.get("/v2/stocks/bars", url.Values{
		"symbols":   []string{strings.ToUpper(symbol)},
		"timeframe": []string{timeframe},
		"limit":     []string{strconv.Itoa(count)},
		"sort":      []string{"asc"},
		"feed":      []string{p.stockFeed},
	}, &payload)
	if err != nil {
		return nil, err
	}
	rows := payload.Bars[strings.ToUpper(symbol)]
	candles := make([]marketdata.Candle, 0, len(rows))
	for _, row := range rows {
		candles = append(candles, marketdata.Candle{
			Time:   parseTimestamp(row.Timestamp),
			Open:   row.Open,
			High:   row.High,
			Low:    row.Low,
			Close:  row.Close,
			Volume: row.Volume,
		})
	}
	return candles, nil
}

func (p *AlpacaProvider) get(path string, query url.Values, dst any) error {
	u := alpacaBaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("APCA-API-KEY-ID", p.apiKey)
	req.Header.Set("APCA-API-SECRET-KEY", p.apiSecret)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("alpaca request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Message != "" {
			return fmt.Errorf("alpaca request failed: %s", apiErr.Message)
		}
		return fmt.Errorf("alpaca request failed: http %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("alpaca decode failed: %w", err)
	}
	return nil
}

type stockSnapshotResponse struct {
	LatestTrade snapshotTrade `json:"latestTrade"`
	LatestQuote snapshotQuote `json:"latestQuote"`
	MinuteBar   barRow        `json:"minuteBar"`
	DailyBar    barRow        `json:"dailyBar"`
}

type stockBarsResponse struct {
	Bars map[string][]barRow `json:"bars"`
}

type barRow struct {
	Timestamp string  `json:"t"`
	Open      float64 `json:"o"`
	High      float64 `json:"h"`
	Low       float64 `json:"l"`
	Close     float64 `json:"c"`
	Volume    float64 `json:"v"`
}

type snapshotTrade struct {
	Timestamp string  `json:"t"`
	Price     float64 `json:"p"`
	Size      int     `json:"s"`
}

type snapshotQuote struct {
	Timestamp string  `json:"t"`
	BidPrice  float64 `json:"bp"`
	AskPrice  float64 `json:"ap"`
}

type optionChainResponse struct {
	Snapshots     map[string]optionSnapshot `json:"snapshots"`
	RootSnapshots map[string]optionSnapshot `json:"-"`
}

func (r *optionChainResponse) UnmarshalJSON(data []byte) error {
	type alias optionChainResponse
	var withWrapper alias
	if err := json.Unmarshal(data, &withWrapper); err == nil && len(withWrapper.Snapshots) > 0 {
		*r = optionChainResponse(withWrapper)
		return nil
	}
	var root map[string]optionSnapshot
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	r.RootSnapshots = root
	return nil
}

type optionSnapshot struct {
	LatestQuote       snapshotQuote `json:"latestQuote"`
	LatestTrade       snapshotTrade `json:"latestTrade"`
	Greeks            greeks        `json:"greeks"`
	Day               optionDay     `json:"day"`
	BidPrice          float64       `json:"bid_price"`
	AskPrice          float64       `json:"ask_price"`
	OpenInterest      int           `json:"open_interest"`
	OpenInterestAlt   int           `json:"oi"`
	Volume            int           `json:"volume"`
	ImpliedVolatility float64       `json:"implied_volatility"`
	IV                float64       `json:"iv"`
}

type optionDay struct {
	Volume int `json:"volume"`
}

type greeks struct {
	Delta float64 `json:"delta"`
	Theta float64 `json:"theta"`
}

func normalizeTimeframe(resolution string) string {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "1m", "1min":
		return "1Min"
	case "5m", "5min":
		return "5Min"
	case "15m", "15min":
		return "15Min"
	case "30m", "30min":
		return "30Min"
	case "1h", "60m":
		return "1Hour"
	default:
		return "1Min"
	}
}

func parseTimestamp(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339}
	for _, layout := range layouts {
		ts, err := time.Parse(layout, v)
		if err == nil {
			return ts
		}
	}
	return time.Time{}
}

func parseOptionSymbol(symbol string) (time.Time, float64, string, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	for i := 0; i < len(symbol); i++ {
		if i+15 > len(symbol) {
			break
		}
		datePart := symbol[i : i+6]
		typePart := symbol[i+6 : i+7]
		strikePart := symbol[i+7 : i+15]
		if _, err := time.Parse("060102", datePart); err != nil {
			continue
		}
		if typePart != "C" && typePart != "P" {
			continue
		}
		rawStrike, err := strconv.Atoi(strikePart)
		if err != nil {
			continue
		}
		expiration, _ := time.Parse("060102", datePart)
		optionType := "call"
		if typePart == "P" {
			optionType = "put"
		}
		return expiration, float64(rawStrike) / 1000.0, optionType, nil
	}
	return time.Time{}, 0, "", fmt.Errorf("unrecognized option symbol %q", symbol)
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func firstPositive(values ...float64) float64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func firstPositiveInt(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
