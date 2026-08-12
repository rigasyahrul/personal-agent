.PHONY: test run build

test:
	go test ./...

run:
	go run ./cmd/personal-agent

build:
	go build ./cmd/personal-agent
