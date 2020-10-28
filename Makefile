#
# Makefile
# Created by Masatoshi Fukunaga on 20/10/28
#
LINT_OPT=--issues-exit-code=0 \
		--enable-all \
		--tests=false \
		--disable=funlen \
		--disable=gochecknoinits \
		--disable=gochecknoglobals \
		--disable=gocognit \
		--disable=godox \
		--disable=lll \
		--disable=maligned \
		--disable=prealloc \
		--disable=wsl \
		--exclude=ifElseChain

.EXPORT_ALL_VARIABLES:

.PHONY: all test lint coverage clean

all: test

test:
	go test -timeout 1m -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html coverage.out -o coverage.out.html

lint:
	golangci-lint run $(LINT_OPT) ./...

coverage: test
	go tool cover -func=coverage.out

clean:
	go clean
