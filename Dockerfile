FROM golang:1.19-alpine as builder
RUN apk add --no-cache git make curl build-base
ENV GOOS=linux
COPY . /src
WORKDIR /src
RUN make test
RUN make certificator

FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata
RUN export PATH=$PATH:/app
WORKDIR /app
COPY --from=builder /src/bin/certificator /app/certificator
CMD ["/app/certificator"]
