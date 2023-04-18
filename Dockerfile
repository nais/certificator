FROM cgr.dev/chainguard/go:1.19 as builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY . /src/
RUN make test
RUN make certificator

FROM cgr.dev/chainguard/static
WORKDIR /app
COPY --from=builder /src/bin/certificator /app/certificator
CMD ["/app/certificator"]
