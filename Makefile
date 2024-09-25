.PHONY: lint test mock-gen

CUR_DIR := $(shell pwd)


test:
	go test -race ./... -v

lint:
	staticcheck ./...
	golangci-lint run

build: build-operator build-signer

build-operator:
	go build -o ./bin/finality-gadget-operator ./finality-gadget/operator/cmd 

build-signer:
	go build -o ./bin/finality-gadget-signer ./finality-gadget/signer/cmd 
