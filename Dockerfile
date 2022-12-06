# syntax=docker/dockerfile:1

# Alpine is chosen for its small footprint
# compared to Ubuntu
FROM docker.io/library/golang:1.19.4

WORKDIR /app

# install curl (healthcheck)
RUN go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.8.7

# Download necessary Go modules
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# build an app
COPY *.go ./
RUN go build -v -buildmode=plugin -o /opi-nvidia-bridge.so ./frontend.go ./spdk.go ./jsonrpc.go

EXPOSE 50051
CMD [ "go", "run", "main.go", "-port=50051" ]
HEALTHCHECK CMD grpcurl -plaintext localhost:50051 list || exit 1
