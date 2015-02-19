.PHONY: all build install test coverage deps release

all: install

build:
	goop go install -v ./...

install: build
	goop exec bash -c 'cp -b $$GOBIN/* ~/go/bin'

test:
	goop go test ./...

coverage:
	goop go test -coverprofile=/tmp/coverage.out gohome/config
	goop go tool cover -func=/tmp/coverage.out
	goop go tool cover -html=/tmp/coverage.out -o /tmp/coverage.html
	xdg-open /tmp/coverage.html

deps:
	goop install

release:
	docker build -t gohome-crossbuild crossbuild
	docker run --rm -v "$(PWD)":/go/src/github.com/barnybug/gohome -w /go/src/github.com/barnybug/gohome gohome-crossbuild
