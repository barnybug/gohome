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
	GOOS=linux GOARCH=arm go build -o dist/gohome-linux-arm/gohome-linux-arm $(exe)

release-arm64:
	GOOS=linux GOARCH=arm64 go build -o dist/gohome-linux-arm64/gohome-linux-arm64 $(exe)

release-amd64:
	GOOS=linux GOARCH=amd64 go build -o dist/gohome-linux-amd64/gohome-linux-amd64 $(exe)

release-386:
	GOOS=linux GOARCH=386 go build -o dist/gohome-linux-386/gohome-linux-386 $(exe)

release-bthome:
	# armv6 for Pi zero
	GOOS=linux GOARCH=arm GOARM=6 go build -o dist/bthome-linux-armv6l/bthome ./cmd/bthome

upx:
	upx dist/gohome-*/gohome-*

release: release-amd64 release-386 release-arm upx
