FROM golang:1.20 as builder

WORKDIR /app
COPY go.mod go.sum ./
COPY *.go ./

RUN apt update -y && apt install -y libpcap-dev
RUN go mod download
RUN go build -o /app/main

FROM debian:trixie-slim

WORKDIR /app
COPY --from=builder /app/main /app/main
RUN apt update -y && apt install libpcap-dev -y
CMD ["/app/main"]
