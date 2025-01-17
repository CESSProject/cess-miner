FROM golang:1.22-alpine3.18 AS builder

ARG go_proxy
ARG TARGETARCH

ENV GOPROXY ${go_proxy}

WORKDIR /opt/target

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN GOOS=linux GOARCH=$TARGETARCH CGO_ENABLED=0 go build -ldflags '-w -s' -gcflags '-N -l' -o cess-miner cmd/main.go

FROM alpine:3.18 AS runner

WORKDIR /opt/cess

COPY --from=builder /opt/target/cess-miner /usr/local/bin/

ENTRYPOINT ["cess-miner"]