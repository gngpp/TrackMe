FROM golang:1.20 as builder

WORKDIR /app

RUN apt update -y && apt install -y libpcap-dev
COPY go.mod go.sum ./
COPY *.go ./
RUN go mod download
RUN go build -o /app/main

FROM debian:trixie-slim

WORKDIR /app
RUN apt update -y && apt install libpcap-dev -y
COPY --from=builder /app/main /app/main
CMD ["/app/main"]
