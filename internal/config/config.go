// Package config loads and validates options-scout configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration struct.
type Config struct {
	Account AccountConfig `yaml:"account"`
	Scoring ScoringConfig `yaml:"scoring"`
	Filters FilterConfig  `yaml:"filters"`
	Exit    ExitConfig    `yaml:"exit"`
	Data    DataConfig    `yaml:"data"`
}

// AccountConfig holds per-trade and per-day risk limits.
type AccountConfig struct {
	MaxRiskPerTrade   float64 `yaml:"max_risk_per_trade"`
	MaxRiskPerDay     float64 `yaml:"max_risk_per_day"`
	MaxOpenOptionRisk float64 `yaml:"max_open_option_risk"`
	MaxSameDirection  int     `yaml:"max_same_direction"`
}

// ScoringConfig holds decision thresholds.
type ScoringConfig struct {
	MinScoreTrade int `yaml:"min_score_trade"`
	MinScoreWatch int `yaml:"min_score_watch"`
}

// FilterConfig holds option quality filters.
type FilterConfig struct {
	MaxBidAskSpreadPct float64 `yaml:"max_bid_ask_spread_pct"`
	MinOpenInterest    int     `yaml:"min_open_interest"`
	MinVolume          int     `yaml:"min_volume"`
	MaxDTEDebit        int     `yaml:"max_dte_debit"`
	MinDTEDebit        int     `yaml:"min_dte_debit"`
	MaxDTECredit       int     `yaml:"max_dte_credit"`
	MinDTECredit       int     `yaml:"min_dte_credit"`
}

// ExitConfig holds profit-taking and stop-loss parameters.
type ExitConfig struct {
	TakeProfitMinPct float64 `yaml:"take_profit_min_pct"`
	TakeProfitMaxPct float64 `yaml:"take_profit_max_pct"`
	StopLossMinPct   float64 `yaml:"stop_loss_min_pct"`
	StopLossMaxPct   float64 `yaml:"stop_loss_max_pct"`
}

// DataConfig holds market data provider settings.
type DataConfig struct {
	Provider        string `yaml:"provider" json:"provider"`
	PolygonAPIKey   string `yaml:"polygon_api_key" json:"polygon_api_key,omitempty"`
	AlpacaAPIKey    string `yaml:"alpaca_api_key" json:"alpaca_api_key,omitempty"`
	AlpacaAPISecret string `yaml:"alpaca_api_secret" json:"alpaca_api_secret,omitempty"`
	AlpacaFeed      string `yaml:"alpaca_feed" json:"alpaca_feed,omitempty"`
	OptionsFeed     string `yaml:"options_feed" json:"options_feed,omitempty"`
}

// Load reads a YAML config file and returns a Config.
// Environment variables override file values for sensitive keys.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}

	// Override with env vars if set.
	if v := os.Getenv("MARKETDATA_PROVIDER"); v != "" {
		cfg.Data.Provider = v
	}
	if v := os.Getenv("POLYGON_API_KEY"); v != "" {
		cfg.Data.PolygonAPIKey = v
	}
	if v := os.Getenv("ALPACA_API_KEY"); v != "" {
		cfg.Data.AlpacaAPIKey = v
	}
	if v := os.Getenv("ALPACA_API_KEY_ID"); v != "" {
		cfg.Data.AlpacaAPIKey = v
	}
	if v := os.Getenv("ALPACA_API_SECRET"); v != "" {
		cfg.Data.AlpacaAPISecret = v
	}
	if v := os.Getenv("ALPACA_API_SECRET_KEY"); v != "" {
		cfg.Data.AlpacaAPISecret = v
	}
	if v := os.Getenv("ALPACA_FEED"); v != "" {
		cfg.Data.AlpacaFeed = v
	}
	if v := os.Getenv("OPTIONS_FEED"); v != "" {
		cfg.Data.OptionsFeed = v
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Default returns a safe default configuration.
func Default() *Config {
	return &Config{
		Account: AccountConfig{
			MaxRiskPerTrade:   150,
			MaxRiskPerDay:     300,
			MaxOpenOptionRisk: 500,
			MaxSameDirection:  2,
		},
		Scoring: ScoringConfig{
			MinScoreTrade: 70,
			MinScoreWatch: 60,
		},
		Filters: FilterConfig{
			MaxBidAskSpreadPct: 0.15,
			MinOpenInterest:    100,
			MinVolume:          10,
			MaxDTEDebit:        7,
			MinDTEDebit:        1,
			MaxDTECredit:       45,
			MinDTECredit:       14,
		},
		Exit: ExitConfig{
			TakeProfitMinPct: 0.30,
			TakeProfitMaxPct: 0.70,
			StopLossMinPct:   0.25,
			StopLossMaxPct:   0.40,
		},
		Data: DataConfig{
			Provider:    "manual",
			AlpacaFeed:  "iex",
			OptionsFeed: "opra",
		},
	}
}

func (cfg Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "config<redacted>"
	}
	return string(data)
}

func (d DataConfig) String() string {
	return fmt.Sprintf(
		"provider=%s polygon_api_key=%s alpaca_api_key=%s alpaca_api_secret=%s alpaca_feed=%s options_feed=%s",
		d.Provider,
		maskSecret(d.PolygonAPIKey),
		maskSecret(d.AlpacaAPIKey),
		maskSecret(d.AlpacaAPISecret),
		d.AlpacaFeed,
		d.OptionsFeed,
	)
}

func (d DataConfig) MarshalJSON() ([]byte, error) {
	type redactedDataConfig struct {
		Provider        string `json:"provider"`
		PolygonAPIKey   string `json:"polygon_api_key,omitempty"`
		AlpacaAPIKey    string `json:"alpaca_api_key,omitempty"`
		AlpacaAPISecret string `json:"alpaca_api_secret,omitempty"`
		AlpacaFeed      string `json:"alpaca_feed,omitempty"`
		OptionsFeed     string `json:"options_feed,omitempty"`
	}
	return json.Marshal(redactedDataConfig{
		Provider:        d.Provider,
		PolygonAPIKey:   maskSecret(d.PolygonAPIKey),
		AlpacaAPIKey:    maskSecret(d.AlpacaAPIKey),
		AlpacaAPISecret: maskSecret(d.AlpacaAPISecret),
		AlpacaFeed:      d.AlpacaFeed,
		OptionsFeed:     d.OptionsFeed,
	})
}

func maskSecret(v string) string {
	if v == "" {
		return ""
	}
	return "set"
}

func validate(cfg *Config) error {
	if cfg.Account.MaxRiskPerTrade <= 0 {
		return fmt.Errorf("config: max_risk_per_trade must be > 0")
	}
	if cfg.Account.MaxRiskPerDay < cfg.Account.MaxRiskPerTrade {
		return fmt.Errorf("config: max_risk_per_day must be >= max_risk_per_trade")
	}
	return nil
}
