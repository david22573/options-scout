package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestConfigRedactsSecretsInStringAndJSON(t *testing.T) {
	cfg := Default()
	cfg.Data.Provider = "alpaca"
	cfg.Data.AlpacaAPIKey = "secret-key"
	cfg.Data.AlpacaAPISecret = "secret-secret"

	rendered := cfg.String()
	if strings.Contains(rendered, "secret-key") || strings.Contains(rendered, "secret-secret") {
		t.Fatalf("config string leaked secret: %s", rendered)
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(data)
	if strings.Contains(serialized, "secret-key") || strings.Contains(serialized, "secret-secret") {
		t.Fatalf("config json leaked secret: %s", serialized)
	}
	if !strings.Contains(serialized, "\"alpaca_api_key\":\"set\"") {
		t.Fatalf("expected redacted json, got %s", serialized)
	}
}
