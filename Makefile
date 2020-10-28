#
# Makefile
# Created by Masatoshi Fukunaga on 20/10/28
#
DEPS_DIR := $(PWD)/vendor
COVER_PATH := coverage.out
GOCMD:=GOPATH=$(DEPS_DIR) go
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test -timeout 1m
GOTOOL := $(GOCMD) tool
GOLINT := `which golangci-lint`
PKGS=$(addprefix ./,$(filter-out _%/ vendor/,$(sort $(dir $(wildcard */*)))))
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

PATH:=$(DEPS_DIR)/bin:$(PATH)


.PHONY: all test lint coverage clean

all: test

test:
	$(GOTEST) -coverprofile=$(COVER_PATH) -covermode=atomic . $(PKGS)
	$(GOTOOL) cover -html $(COVER_PATH) -o $(COVER_PATH).html

lint:
	$(GOLINT) run $(LINT_OPT) . $(PKGS)

coverage: test
	$(GOTOOL) cover -func=$(COVER_PATH)

clean:
	$(GOCLEAN)
