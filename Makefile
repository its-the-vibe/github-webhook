.PHONY: build test lint

build:
	go build -o webhook-server .

test:
	go test ./...

lint:
	test -z $$(gofmt -l .)
