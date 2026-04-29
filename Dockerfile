FROM golang:1.26 AS builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test -count 1 -v ./...
RUN BUILDTIME=$(date "+%s") \
    && DATE=$(date "+%Y-%m-%d") \
    && LAST_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
    && go build -o bin/certificator \
       -ldflags "-s \
         -X github.com/nais/certificator/pkg/version.Revision=${LAST_COMMIT} \
         -X github.com/nais/certificator/pkg/version.Date=${DATE} \
         -X github.com/nais/certificator/pkg/version.BuildUnixTime=${BUILDTIME}" \
       cmd/certificator/*.go

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=builder /src/bin/certificator /app/certificator
CMD ["/app/certificator"]
