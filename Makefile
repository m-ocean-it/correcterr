.PHONY: build run test clean

SRC := $(shell find . -type f -name '*.go' ! -name '*_test.go')

.PHONY: build
build: ./bin/correcterr

run: build
	./bin/correcterr

test:
	go test ./...

clean:
	rm -rf ./bin

.PHONY: ./bin/correcterr
./bin/correcterr: $(SRC) | ./bin
	go build -o ./bin/correcterr ./cmd/correcterr

./bin:
	mkdir -p ./bin
