default: lint

test: export CGO_ENABLED = 0
test:
	@go test -tags netgo --count 1 ./...

race: export CGO_ENABLED = 1
race:
	go test -tags netgo -race --count 1 ./...

bench: export CGO_ENABLED = 1
bench:
	go test -tags netgo --count 1 -run Bench -bench=. ./...

lint:
	@golangci-lint run

.PHONY: test race bench lint
