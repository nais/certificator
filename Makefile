.PHONY: certificator

certificator:
	go build -o bin/certificator cmd/certificator/*.go
