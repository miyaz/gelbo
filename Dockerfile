FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETARCH
ENV GOOS=linux
ENV GOPROXY=direct
WORKDIR /go/src/work

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN mkdir -p cert \
 && openssl req -x509 -nodes -newkey rsa:2048 -days 3650 -keyout cert/server-key.pem -out cert/server-cert.pem -subj "/CN=localhost"
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -buildvcs=false -trimpath -ldflags '-w -s' -o /go/bin/gelbo

FROM alpine AS runner

EXPOSE 80
RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/gelbo /app/gelbo

ENTRYPOINT ["/app/gelbo"]
