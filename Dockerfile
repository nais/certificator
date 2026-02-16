FROM golang:1.26 AS builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY . .
RUN make check
RUN make test
RUN make certificator

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=builder /src/bin/certificator /app/certificator
CMD ["/app/certificator"]
