FROM golang:1.20 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOPROXY direct
WORKDIR /go/src/work

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN mkdir -p cert \
 && openssl req -x509 -nodes -newkey rsa:2048 -days 3650 -keyout cert/server-key.pem -out cert/server-cert.pem -subj "/CN=localhost"
RUN go build -o /go/bin/gelbo -ldflags '-s -w'

FROM alpine as runner

EXPOSE 80
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/gelbo /app/gelbo

ENTRYPOINT ["/app/gelbo"]
