FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETARCH
ENV GOOS=linux
ENV GOPROXY=direct

# install protoc (apt install)
RUN apt-get update && apt install -y protobuf-compiler

# install protoc (download from github)
#RUN apt-get update && apt install -y unzip
#ENV PROTOC_VERSION=31.1
#RUN curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip \
# && unzip protoc-${PROTOC_VERSION}-linux-x86_64.zip -d /usr/local \
# && rm -f protoc-${PROTOC_VERSION}-linux-x86_64.zip

# install gRPC plugin for Go
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /go/src/work
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN protoc --proto_path ./grpc/proto \
           --go_out=grpc/pb/ --go_opt=paths=source_relative \
           --go-grpc_out=grpc/pb/ --go-grpc_opt=paths=source_relative \
           gelbo.proto

RUN mkdir -p cert \
 && openssl req -x509 -nodes -newkey rsa:2048 -days 3650 -keyout cert/server-key.pem -out cert/server-cert.pem -subj "/CN=localhost"
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -buildvcs=false -trimpath -ldflags '-w -s' -o /go/bin/gelbo

FROM alpine AS runner

EXPOSE 80
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/gelbo /app/gelbo

ENTRYPOINT ["/app/gelbo"]
