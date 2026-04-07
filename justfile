# Run tests with race detector and coverage
test:
    go install github.com/mfridman/tparse@latest
    go test -race -json -shuffle=on -covermode=atomic ./... | tparse -progress

# Run benchmarks with memory allocation stats
bench:
    go test -bench=. -benchmem -count=3 ./...

# Run golangci-lint
lint:
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
    golangci-lint run ./...

# Run the go vulnerability checker
vuln:
    go install golang.org/x/vuln/cmd/govulncheck@latest
    govulncheck ./...

# Tidy up
tidy:
    go mod tidy
    go fix ./...
    go fmt ./...

# Actionlint
actionlint:
    go install github.com/rhysd/actionlint/cmd/actionlint@latest
    actionlint
    @if command -v zizmor > /dev/null 2>&1; then \
        zizmor .github/workflows/; \
    else \
        echo "zizmor is not installed. Install it to enable Actions security linting:"; \
        echo "  https://docs.zizmor.sh/installation/"; \
    fi
