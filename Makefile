PWD=$(shell pwd)
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

BINARY = migrato
#LDFLAGS = -ldflags "-X main.Version=${APP_VERSION}"

.DEFAULT_GOAL=help

build: ## build the app
	go build -o ${BINARY} cmd/migrato/main.go

.PHONY: setup
setup: ## setups up the environment
	go get -u github.com/cespare/reflex

.PHONY: run.local
run.local:
	ulimit -S -n 5000 && reflex -r '\.(go|json|yml)$$' -R '^vendor/' -s -- sh -c 'go build ${LDFLAGS} -o ${BINARY} cmd/migrato/main.go && ./${BINARY}'

.PHONY: docker.start
docker.start:
	@echo "$(OK_COLOR)==> Start local docker environment $(NO_COLOR)"
	docker-compose up -d --build

.PHONY: docker.stop
docker.stop:
	@echo "$(OK_COLOR)==> Stop local docker environment $(NO_COLOR)"
	docker kill activity-feed.consul
	docker kill activity-feed.mysql

.PHONY: help
help:
	@grep -E '^[a-zA-Z\._-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
