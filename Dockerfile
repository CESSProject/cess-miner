FROM golang:alpine AS builder

# go_proxy
ARG go_proxy
ENV GOPROXY ${go_proxy}

# Workdir
WORKDIR /opt/target

# Copy file
COPY . /opt/target/

# Build
RUN cd /opt/target/ \
  && go mod download \
  && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -gcflags '-N -l' -o cess-bucket cmd/main.go

# Run
FROM alpine AS runner
WORKDIR /opt/cess
COPY --from=builder /opt/target/cess-bucket ./
