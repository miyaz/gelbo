FROM golang:1.18 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPROXY direct

# install protoc
WORKDIR /protoc
RUN apt update && apt-get install -y unzip
RUN wget https://github.com/protocolbuffers/protobuf/releases/download/v3.20.1/protoc-3.20.1-linux-x86_64.zip && \
    unzip protoc-3.20.1-linux-x86_64.zip && \
    ln -s /protoc/bin/protoc /bin/protoc
RUN go install github.com/golang/protobuf/protoc-gen-go@latest

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN ./protoc.sh

RUN openssl req -x509 -nodes -newkey rsa:2048 -days 3650 -keyout cert/server-key.pem -out cert/server-cert.pem -subj "/CN=localhost"

RUN go build -o /go/bin/gelbo -ldflags '-s -w'

FROM alpine as runner

EXPOSE 80
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/gelbo /app/gelbo
COPY --from=builder /protoc/cert /cert
ENTRYPOINT ["/app/gelbo"]
