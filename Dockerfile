FROM cesslab/cess-pbc-env:latest AS builder

ARG go_proxy
ENV GOPROXY ${go_proxy}

# Download packages first so they can be cached.
COPY go.mod go.sum /opt/target/
RUN cd /opt/target/ && go mod download

COPY . /opt/target/

# Build the thing.
RUN cd /opt/target/ \
  && go build -o cess-bucket cmd/main/main.go

FROM cesslab/cess-pbc-env:latest
WORKDIR /opt/cess
COPY --from=builder /opt/target/cess-bucket ./

