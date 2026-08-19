.PHONY: test lint fmt-check run build

test:
	go test ./...

lint: fmt-check
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || \
	  (echo "Go files need gofmt"; gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1)

run:
	go run ./cmd/personal-agent

build:
	go build ./cmd/personal-agent
