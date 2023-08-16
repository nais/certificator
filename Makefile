BUILDTIME = $(shell date "+%s")
DATE = $(shell date "+%Y-%m-%d")
LAST_COMMIT = $(shell git rev-parse --short HEAD)
LDFLAGS := -X github.com/nais/certificator/pkg/version.Revision=$(LAST_COMMIT) -X github.com/nais/certificator/pkg/version.Date=$(DATE) -X github.com/nais/certificator/pkg/version.BuildUnixTime=$(BUILDTIME)

.PHONY: certificator test check alpine docker

certificator:
	go build -o bin/certificator -ldflags "-s $(LDFLAGS)" cmd/certificator/*.go

test:
	go test -count 1 -v ./...

check:
	go run honnef.co/go/tools/cmd/staticcheck ./...
	go run golang.org/x/vuln/cmd/govulncheck -show=traces ./...

docker:
	docker build -t ghcr.io/nais/certificator:latest .
