.PHONY: help
help:
	@echo "make test: Runs tests and updates coverage.html"
	@echo "make test-race: Runs go tests in race detection mode (no coverage report)"
	@echo "make generate: Runs code generation"

.PHONY: test
test:
	GOEXPERIMENT=synctest go test -coverprofile coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html
	GOEXPERIMENT=synctest golangci-lint run ./...

test-race:
	GOMAXPROCS=1 GOTRACEBACK=all GOEXPERIMENT=synctest go test -race -count 10 ./...

.PHONY: generate
generate:
	go generate ./...
