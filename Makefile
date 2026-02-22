.PHONY: build run test lint clean

build:
	go build -o bin/server .

run: build
	./bin/server

test:
	go test ./... -v -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
