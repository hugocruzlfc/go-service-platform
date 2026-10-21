.PHONY: build fmt vet test test-race lint vuln integration ci

build:
	go build ./...

fmt:
	test -z "$$(gofmt -l .)"

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race -shuffle=on ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0 run

vuln:
	go tool govulncheck ./...

integration:
	go test -tags=integration -count=1 ./...

ci: test-race lint
