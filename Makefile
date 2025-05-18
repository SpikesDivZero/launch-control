export GOEXPERIMENT = synctest

ifdef RACE_TEST
export GOMAXPROCS  = 1
export GOTRACEBACK = all
TEST_FLAG = -race
endif

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
test-race:
	@$(MAKE) RACE_TEST=1 test

.PHONY: generate
generate:
	go generate ./...
