GIT_COMMIT = $(shell git rev-parse HEAD)
BUILD_DATE = $(shell date -u +"%FT%T%z")
APP_VERSION = "${GIT_COMMIT}_${BUILD_DATE}"

BINARY = migrato
LDFLAGS = -ldflags "-X main.Version=${APP_VERSION}"

.DEFAULT_GOAL=help

build: ## build the app
	GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY} main.go

.PHONY: setup
setup: ## setups up the environment
	go get -u github.com/cespare/reflex

.PHONY: run.local
run.local:
	ulimit -S -n 5000 && reflex -r '\.(go|json|yml)$$' -R '^vendor/' -s -- sh -c 'go build ${LDFLAGS} -o ${BINARY} *.go && ./cmd/${BINARY}'

.PHONY: help
help:
	@grep -E '^[a-zA-Z\._-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
