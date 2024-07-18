FROM golang:1.21-alpine3.18 AS builder

# go_proxy
ARG go_proxy
ENV GOPROXY ${go_proxy}

# Workdir
WORKDIR /opt/target

# Download packages first so they can be cached.
COPY go.mod go.sum ./
RUN go mod download

# Copy file
COPY . ./

# Build
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -gcflags '-N -l' -o cess-miner cmd/main.go

# Run
FROM alpine:3.18 AS runner
# RUN apk add curl
WORKDIR /opt/cess
COPY --from=builder /opt/target/cess-miner /usr/local/bin/
ENTRYPOINT ["cess-miner"]
