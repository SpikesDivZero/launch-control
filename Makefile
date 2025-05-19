export GOEXPERIMENT = synctest

.PHONY: help
help:
	@echo "make test: Runs tests and updates coverage.html"
	@echo "make test-race: Same as 'make test', but runs it in race detection mode"
	@echo "make generate: Runs code generation"

.PHONY: test
test:
	go test ${TEST_FLAG} -coverprofile coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	golangci-lint run ./...

.PHONY: test-race
test-race: export GOMAXPROCS=1
test-race: export GOTRACEBACK=all
test-race: TEST_FLAG=-race -count 10
test-race: test

.PHONY: generate
generate:
	go generate ./...
