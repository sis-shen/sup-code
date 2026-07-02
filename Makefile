# SupCode Makefile

.PHONY: all lint test coverage build clean vet verify

all: vet test build

lint:
	bash scripts/lint.sh

test:
	go test -race -count=1 ./...

coverage:
	bash scripts/coverage.sh

build:
	go build ./...

vet:
	go vet ./...

verify: vet test build
	go mod verify

clean:
	rm -rf build/
	go clean -cache
	@echo "Cleaned."
