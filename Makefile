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
	go tool golangci-lint run

vuln:
	go tool govulncheck ./...

integration:
	go test -tags=integration -count=1 ./...

ci: test-race lint
