all: lint build

lint:
	go fmt
	golangci-lint run

build:
	go build
