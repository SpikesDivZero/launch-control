.PHONY: help
help:
	@echo "make test: Runs tests and updates coverage.html"
	@echo "make generate: Runs code generation"

.PHONY: test
test:
	GOEXPERIMENT=synctest go test -coverprofile coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: generate
generate:
	go generate ./...
