.PHONY: help
help:
	@echo "make test: Runs tests and updates coverage.html"

.PHONY: test
test:
	GOEXPERIMENT=synctest go test -coverprofile coverage.out -v ./...
	go tool cover -html=coverage.out -o coverage.html
