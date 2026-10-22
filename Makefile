.PHONY: build test run fmt docker
build:
	go build -o bin/grange ./cmd/grange
test:
	go vet ./... && go test ./...
run:
	GRANGE_ADMIN_TOKEN=dev go run ./cmd/grange serve
fmt:
	gofmt -w .
docker:
	docker build -t grange:dev .
