FROM golang:1.24-alpine3.21 as builder

RUN apk add --no-cache make ca-certificates gcc musl-dev linux-headers git jq bash

COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum

WORKDIR /app

RUN go mod download

# build flashloan-scanner with the shared go.mod & go.sum files
COPY . /app/flashloan-scanner

WORKDIR /app/flashloan-scanner

RUN make flashloan-scanner

FROM alpine:3.18

COPY --from=builder /app/flashloan-scanner/flashloan-scanner /usr/local/bin
COPY --from=builder /app/flashloan-scanner/flashloan-scanner.yaml /app/flashloan-scanner/flashloan-scanner.yaml
COPY --from=builder /app/flashloan-scanner/migrations /app/flashloan-scanner/migrations

ENV GAS_ORACLE_CONFIG="/app/flashloan-scanner/migrations"
ENV GAS_ORACLE_MIGRATIONS_DIR="/app/flashloan-scanner/flashloan-scanner.yaml"
WORKDIR /app/flashloan-scanner

