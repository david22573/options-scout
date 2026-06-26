package autodata

import (
	"time"

	"github.com/davidmiguel22573/options-scout/internal/marketdata"
)

type MarketSession string

const (
	SessionPremarket  MarketSession = "PREMARKET"
	SessionRegular    MarketSession = "REGULAR"
	SessionAfterHours MarketSession = "AFTER_HOURS"
	SessionClosed     MarketSession = "CLOSED"
	SessionUnknown    MarketSession = "UNKNOWN"
)

type DataFreshnessStatus string

const (
	FreshnessFresh      DataFreshnessStatus = "FRESH"
	FreshnessStale      DataFreshnessStatus = "STALE"
	FreshnessIncomplete DataFreshnessStatus = "INCOMPLETE"
	FreshnessUnknown    DataFreshnessStatus = "UNKNOWN"
)

type ClockInfo struct {
	Timestamp time.Time
	IsOpen    bool
	NextOpen  time.Time
	NextClose time.Time
}

type MarketClock struct {
	Session           MarketSession       `json:"session"`
	IsMarketOpen      bool                `json:"is_market_open"`
	NextOpenTime      *time.Time          `json:"next_open_time,omitempty"`
	NextCloseTime     *time.Time          `json:"next_close_time,omitempty"`
	DataFreshness     DataFreshnessStatus `json:"data_freshness_status"`
	QuoteTimestamp    time.Time           `json:"quote_timestamp,omitempty"`
	Now               time.Time           `json:"now"`
	UsedProviderClock bool                `json:"used_provider_clock"`
}

type Readiness struct {
	Clock                 MarketClock `json:"clock"`
	HasFreshData          bool        `json:"has_fresh_data"`
	HasQuote              bool        `json:"has_quote"`
	OptionChainsLoaded    bool        `json:"option_chains_loaded"`
	OpeningRangeAvailable bool        `json:"opening_range_available"`
	VWAPAvailable         bool        `json:"vwap_available"`
	RecommendationAllowed bool        `json:"recommendation_allowed"`
	Reasons               []string    `json:"reasons,omitempty"`
}

func resolveMarketClock(provider Provider, quote *time.Time) MarketClock {
	var (
		info    *ClockInfo
		now     = time.Now()
		usedAPI bool
	)
	if provider != nil {
		if clock, err := provider.MarketClock(); err == nil && clock != nil {
			info = clock
			usedAPI = true
			if !clock.Timestamp.IsZero() {
				now = clock.Timestamp
			}
		}
	}

	session := classifySession(now, info)
	out := MarketClock{
		Session:           session,
		IsMarketOpen:      session == SessionRegular,
		DataFreshness:     FreshnessUnknown,
		Now:               now,
		UsedProviderClock: usedAPI,
	}
	if info != nil {
		if !info.NextOpen.IsZero() {
			nextOpen := info.NextOpen
			out.NextOpenTime = &nextOpen
		}
		if !info.NextClose.IsZero() {
			nextClose := info.NextClose
			out.NextCloseTime = &nextClose
		}
	}
	if out.NextOpenTime == nil || out.NextCloseTime == nil {
		nextOpen, nextClose := fallbackNextTimes(now, session)
		if out.NextOpenTime == nil && !nextOpen.IsZero() {
			out.NextOpenTime = &nextOpen
		}
		if out.NextCloseTime == nil && !nextClose.IsZero() {
			out.NextCloseTime = &nextClose
		}
	}
	if quote != nil {
		out.QuoteTimestamp = *quote
		out.DataFreshness = freshnessFromTimestamp(*quote, now)
	}
	return out
}

func fallbackNextTimes(now time.Time, session MarketSession) (time.Time, time.Time) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.Time{}, time.Time{}
	}
	t := now.In(ny)
	openToday := time.Date(t.Year(), t.Month(), t.Day(), 9, 30, 0, 0, ny)
	closeToday := time.Date(t.Year(), t.Month(), t.Day(), 16, 0, 0, 0, ny)

	switch session {
	case SessionRegular:
		return nextWeekdayOpen(openToday.Add(24*time.Hour), ny), closeToday
	case SessionPremarket:
		return openToday, closeToday
	case SessionAfterHours:
		return nextWeekdayOpen(openToday.Add(24*time.Hour), ny), closeToday
	case SessionClosed:
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday || t.After(closeToday) {
			return nextWeekdayOpen(openToday.Add(24*time.Hour), ny), closeToday
		}
		return openToday, closeToday
	default:
		return time.Time{}, time.Time{}
	}
}

func nextWeekdayOpen(start time.Time, loc *time.Location) time.Time {
	t := time.Date(start.Year(), start.Month(), start.Day(), 9, 30, 0, 0, loc)
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.Add(24 * time.Hour)
	}
	return t
}

func classifySession(now time.Time, info *ClockInfo) MarketSession {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		return SessionUnknown
	}
	t := now.In(ny)
	if info != nil && info.IsOpen {
		return SessionRegular
	}
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return SessionClosed
	}
	minutes := t.Hour()*60 + t.Minute()
	switch {
	case minutes >= 4*60 && minutes < 9*60+30:
		return SessionPremarket
	case minutes >= 9*60+30 && minutes < 16*60:
		return SessionRegular
	case minutes >= 16*60 && minutes < 20*60:
		return SessionAfterHours
	default:
		return SessionClosed
	}
}

func freshnessFromTimestamp(ts, now time.Time) DataFreshnessStatus {
	if ts.IsZero() {
		return FreshnessIncomplete
	}
	if now.Sub(ts) > 20*time.Minute {
		return FreshnessStale
	}
	return FreshnessFresh
}

func regularSessionCandles(candles []marketdata.Candle, now time.Time) []marketdata.Candle {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil
	}
	day := now.In(ny).Format("2006-01-02")
	out := make([]marketdata.Candle, 0, len(candles))
	for _, candle := range candles {
		ct := candle.Time.In(ny)
		if ct.Format("2006-01-02") != day {
			continue
		}
		minutes := ct.Hour()*60 + ct.Minute()
		if minutes < 9*60+30 || minutes >= 16*60 {
			continue
		}
		out = append(out, candle)
	}
	return out
}
