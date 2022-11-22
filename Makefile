.PHONY: certificator

certificator:
	go build -o bin/certificator cmd/certificator/*.go

test:
	go test -count 1 -v ./...
