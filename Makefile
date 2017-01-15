export GO15VENDOREXPERIMENT=1

exe = github.com/barnybug/gohome/cmd/gohome

.PHONY: all build install test coverage deps release

all: install

install:
	go install -v $(exe)

test:
	go test ./config/... ./lib/... ./cmd/... ./pubsub/... ./services/... ./util/...

coverage:
	go test -coverprofile=/tmp/coverage.out gohome/config
	go tool cover -func=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html
	xdg-open /tmp/coverage.html

release-arm:
	GOOS=linux GOARCH=arm go build -o release/gohome-linux-arm $(exe)

release-amd64:
	GOOS=linux GOARCH=amd64 go build -o release/gohome-linux-amd64 $(exe)

release-386:
	GOOS=linux GOARCH=386 go build -o release/gohome-linux-386 $(exe)

upx:
	upx release/*

release: release-amd64 release-386 release-arm upx
