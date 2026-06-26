.PHONY: build test smoke recommend-review packet-template packet-validate recommend-packet auto-packet-qqq auto-plan-qqq morning-ready auto-recommend-qqq auto-watchlist fmt-check verify

CACHE_DIR := $(CURDIR)/.cache
GO_BUILD_CACHE := $(CACHE_DIR)/go-build
GO_MOD_CACHE := $(CACHE_DIR)/go-mod
GO_ENV := GOTOOLCHAIN=local GOWORK=off GOCACHE=$(GO_BUILD_CACHE) GOMODCACHE=$(GO_MOD_CACHE)

$(GO_BUILD_CACHE) $(GO_MOD_CACHE):
	mkdir -p $@

build:
	$(GO_ENV) go build -buildvcs=false -o options-scout ./cmd/options-scout

test: $(GO_BUILD_CACHE) $(GO_MOD_CACHE)
	$(GO_ENV) go test ./...

fmt-check:
	@test -z "$$(find . -path './.cache' -prune -o -name '*.go' -print | xargs gofmt -l)" || (echo "gofmt needed:" && find . -path './.cache' -prune -o -name '*.go' -print | xargs gofmt -l && exit 1)

smoke: build
	./options-scout --help
	./options-scout env-check
	./options-scout review || true
	./options-scout packet-template --symbol QQQ --out runs/packets/QQQ_packet_template.json
	./options-scout packet-validate --packet runs/packets/QQQ_packet_template.json

recommend-review: build
	./options-scout recommend --symbol QQQ --manual-chain examples/chain_QQQ.json --max-risk 150
	./options-scout review

packet-template: build
	./options-scout packet-template --symbol QQQ --out runs/packets/QQQ_packet_template.json

packet-validate: build
	./options-scout packet-validate --packet runs/packets/QQQ_packet_template.json

recommend-packet: build
	./options-scout recommend-packet --packet examples/packets/qqq_good_call_packet.json --max-risk 50 --no-journal

auto-packet-qqq: build
	./options-scout auto-packet --symbol QQQ --max-risk 50 --dte-min 1 --dte-max 7 --out runs/packets/QQQ_auto_`date +%F`.json

auto-plan-qqq: build
	./options-scout auto-plan --symbol QQQ --max-risk 50 --dte-min 1 --dte-max 7

morning-ready: build
	./options-scout morning-ready --symbols QQQ,SPY --max-risk 50 --dte-min 1 --dte-max 3

auto-recommend-qqq: build
	./options-scout auto-recommend --symbol QQQ --max-risk 50 --dte-min 1 --dte-max 7 --no-journal

auto-watchlist: build
	./options-scout auto-watchlist --symbols QQQ,SPY --max-risk 50 --dte-min 1 --dte-max 7

verify: fmt-check test build smoke recommend-packet
