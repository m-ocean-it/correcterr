.PHONY: build run test clean

SRC := $(shell find . -type f -name '*.go' ! -name '*_test.go')

build: ./bin/correcterr

run: build
	./bin/correcterr

test:
	go test ./...

clean:
	rm -rf ./bin

./bin/correcterr: $(SRC) | ./bin
	go build -o ./bin/correcterr ./cmd/correcterr

./bin:
	mkdir -p ./bin
