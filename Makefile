export GO15VENDOREXPERIMENT=1

exe = github.com/barnybug/gohome/processes/gohome

.PHONY: all build install test coverage deps release

all: install

install:
	go install -v $(exe)

test:
	go test ./config/... ./lib/... ./processes/... ./pubsub/... ./services/... ./util/... .

coverage:
	go test -coverprofile=/tmp/coverage.out gohome/config
	go tool cover -func=/tmp/coverage.out
	go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html
	xdg-open /tmp/coverage.html

release:
	GOOS=linux GOARCH=amd64 go build -o release/gohome-linux-amd64 $(exe)
	GOOS=linux GOARCH=386 go build -o release/gohome-linux-386 $(exe)
	GOOS=linux GOARCH=arm GOARM=5 go build -o release/gohome-linux-arm5 $(exe)
	upx --best release/*
