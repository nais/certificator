FROM golang:1.21 as builder
ENV GOOS=linux
ENV CGO_ENABLED=0
WORKDIR /src
COPY . /src/
RUN make test
RUN make certificator

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=builder /src/bin/certificator /app/certificator
CMD ["/app/certificator"]
